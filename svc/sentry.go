package svc

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/cryptellation/runtime"
	"github.com/cryptellation/ticks/api"
	"github.com/cryptellation/ticks/pkg/tick"
	"github.com/cryptellation/ticks/svc/exchanges"
	"github.com/cryptellation/ticks/svc/internal/signals"
	"github.com/google/uuid"
	"github.com/iancoleman/strcase"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// ticksSentryWorkflowName is the name of the TicksSentryWorkflow which is
// a long running workflow that listens to the ticks stream and sends them to
// listeners.
const ticksSentryWorkflowName = "TicksSentryWorkflow"

type (
	// ticksSentryWorkflowParams is the input params for the TicksSentryWorkflow.
	ticksSentryWorkflowParams struct {
		Exchange string
		Symbol   string
	}

	// ticksSentryWorkflowResults is the output results for the TicksSentryWorkflow.
	ticksSentryWorkflowResults struct{}
)

func (wf *workflows) ticksSentryWorkflow(
	ctx workflow.Context,
	params ticksSentryWorkflowParams,
) (ticksSentryWorkflowResults, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("Listening to ticks",
		"exchange", params.Exchange,
		"symbol", params.Symbol)

	// Get signal channels
	registerSignalChannel := workflow.GetSignalChannel(ctx, signals.RegisterToTicksListeningSignalName)
	unregisterSignalChannel := workflow.GetSignalChannel(ctx, signals.UnregisterFromTicksListeningSignalName)
	newTickReceivedSignalChannel := workflow.GetSignalChannel(ctx, signals.NewTickReceivedSignalName)

	// Start listening to ticks
	cancelListening := wf.sentryStartListeningActivity(ctx, params)

	// Create listeners
	listeners := make(map[string]workflow.Channel)
	handleListenTicksSignals(ctx, listeners, registerSignalChannel, unregisterSignalChannel)

	// Loop over ticks
	var t tick.Tick
	for len(listeners) > 0 {
		// Get next tick
		logger.Debug("Listening to next tick",
			"listeners_count", listeners)
		newTickReceivedSignalChannel.Receive(ctx, &t)

		// Handle new signals
		// TODO(#4): make new ticks signal handling more asynchronous
		handleListenTicksSignals(ctx, listeners, registerSignalChannel, unregisterSignalChannel)

		// Send event to all listeners
		logger.Debug("Sending tick to listeners",
			"tick", t,
			"listeners_count", len(listeners))
		keys := workflow.DeterministicKeys(listeners)
		for _, k := range keys {
			_ = listeners[k].SendAsync(t)
		}
	}

	// Cancel listening and cleanup signals
	logger.Debug("No more listeners, cancel listening")
	cancelListening()

	// Cleanup remaining signals
	// TODO(#5): clean up new ticks signals when quitting workflow
	// TODO(#6): clean up unregister signals when quitting workflow
	// TODO(#7): clean up register signals and trigger a new workflow if needed when quitting workflow

	logger.Info("Stop listening to ticks",
		"exchange", params.Exchange,
		"symbol", params.Symbol)

	return ticksSentryWorkflowResults{}, nil
}

func (wf *workflows) sentryStartListeningActivity(
	ctx workflow.Context,
	params ticksSentryWorkflowParams,
) func() {
	// Set activity options
	activityOptions := exchanges.DefaultActivityOptions()
	activityOptions.ScheduleToCloseTimeout = 365 * 24 * time.Hour
	activityOptions.StartToCloseTimeout = 365 * 24 * time.Hour
	activityOptions.HeartbeatTimeout = time.Second
	ctx = workflow.WithActivityOptions(ctx, activityOptions)

	// Execute activity with cancel
	ctx, cancelActivity := workflow.WithCancel(ctx)
	_ = workflow.ExecuteActivity(
		ctx, wf.exchangesAdapter.ListenSymbolActivity, exchanges.ListenSymbolParams{
			ParentWorkflowID: workflow.GetInfo(ctx).WorkflowExecution.ID,
			Exchange:         params.Exchange,
			Symbol:           params.Symbol,
		})

	return cancelActivity
}

func handleListenTicksSignals(
	ctx workflow.Context,
	listeners map[string]workflow.Channel,
	registerSignalChannel, unregisterSignalChannel workflow.ReceiveChannel,
) {
	logger := workflow.GetLogger(ctx)
	logger.Debug("Handling signals",
		"listeners_count", len(listeners))

	handleRegisterSignals(ctx, listeners, registerSignalChannel)
	handleUnregisterSignals(ctx, listeners, unregisterSignalChannel)
}

func handleRegisterSignals(
	ctx workflow.Context,
	listeners map[string]workflow.Channel,
	registerSignalChannel workflow.ReceiveChannel,
) {
	logger := workflow.GetLogger(ctx)

	// Loop over register signals
	var registerParams signals.RegisterToTicksListeningSignalParams
	for detected := true; detected; {
		// Receive the next register signal
		detected = registerSignalChannel.ReceiveAsync(&registerParams)
		if detected {
			logger.Info("Received register signal", "params", registerParams)

			// Create a new listener
			listeners[registerParams.RequesterID.String()] = workflow.NewBufferedChannel(ctx, 0)

			// Start a new routine to send ticks to the listener
			workflow.Go(ctx, sendToTickListenerRoutine(
				registerParams.CallbackWorkflow,
				listeners,
				registerParams.RequesterID))
		}
	}
}

func handleUnregisterSignals(
	ctx workflow.Context,
	listeners map[string]workflow.Channel,
	unregisterSignalChannel workflow.ReceiveChannel,
) {
	logger := workflow.GetLogger(ctx)

	// Loop over unregister signals
	var unregisterParams signals.UnregisterFromTicksListeningSignalParams
	for detected := true; detected; {
		// Receive the next unregister signal
		detected = unregisterSignalChannel.ReceiveAsync(&unregisterParams)
		if detected {
			// Log the received unregister signal
			logger.Info("Received unregister signal", "params", unregisterParams)

			// Remove the listener
			delete(listeners, unregisterParams.RequesterID.String())
		}
	}
}

func sendToTickListenerRoutine(
	callback runtime.CallbackWorkflow,
	listeners map[string]workflow.Channel,
	requesterID uuid.UUID,
) func(ctx workflow.Context) {
	ch := listeners[requesterID.String()]

	return func(ctx workflow.Context) {
		processTicksForListener(ctx, ch, callback, requesterID, listeners)
	}
}

func createChildWorkflowOptions(callback runtime.CallbackWorkflow) workflow.ChildWorkflowOptions {
	opts := workflow.ChildWorkflowOptions{
		TaskQueue:                callback.TaskQueueName,            // Execute in the client queue
		ParentClosePolicy:        enums.PARENT_CLOSE_POLICY_ABANDON, // Do not close if the parent workflow closes
		WorkflowExecutionTimeout: time.Second * 30,                  // Timeout if the child workflow does not complete
	}

	// Check if the timeout is set
	if callback.ExecutionTimeout > 0 {
		opts.WorkflowExecutionTimeout = callback.ExecutionTimeout
	}

	return opts
}

func processTicksForListener(
	ctx workflow.Context,
	ch workflow.Channel,
	callback runtime.CallbackWorkflow,
	requesterID uuid.UUID,
	listeners map[string]workflow.Channel,
) {
	logger := workflow.GetLogger(ctx)

	for {
		// Receive next event
		var t tick.Tick
		ch.Receive(ctx, &t)

		// Send tick to callback
		if err := sendTickToCallback(ctx, t, callback, requesterID); err != nil {
			if shouldStopProcessing(ctx, err, callback) {
				break
			}
		}
	}

	// Remove listener as it has been in error or stopped.
	logger.Debug("Removing listener", "callback", callback.Name)
	delete(listeners, requesterID.String())
}

func sendTickToCallback(
	ctx workflow.Context,
	t tick.Tick,
	callback runtime.CallbackWorkflow,
	requesterID uuid.UUID,
) error {
	// Create child workflow options
	opts := createChildWorkflowOptions(callback)

	// Generate a unique ID for the workflow
	opts.WorkflowID = fmt.Sprintf(
		"SendTick%s%s-%s",
		strcase.ToCamel(t.Exchange),
		strings.ReplaceAll(t.Pair, "-", ""),
		t.Time.Format(time.RFC3339Nano),
	)
	ctx = workflow.WithChildOptions(ctx, opts)

	// Start a new child workflow
	return workflow.ExecuteChildWorkflow(ctx, callback.Name, api.ListenToTicksCallbackWorkflowParams{
		RequesterID: requesterID,
		Tick:        t,
	}).Get(ctx, nil)
}

func shouldStopProcessing(ctx workflow.Context, err error, callback runtime.CallbackWorkflow) bool {
	logger := workflow.GetLogger(ctx)
	var timeoutErr *temporal.TimeoutError
	if errors.As(err, &timeoutErr) {
		logger.Debug("Listener has timed out, exiting", "callback", callback.Name)
		return true
	}

	logger.Error("Listener has errored, continuing", "error", err, "callback", callback.Name)
	return false
}

func sentryWorkflowName(exchange, pair string) string {
	return fmt.Sprintf("Sentry%s%s", strcase.ToCamel(exchange), strings.ReplaceAll(pair, "-", ""))
}
