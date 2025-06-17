package svc

import (
	"errors"

	"github.com/cryptellation/ticks/api"
	"github.com/cryptellation/ticks/svc/internal/signals"
	"go.temporal.io/sdk/workflow"
)

// UnregisterFromTicksListeningWorkflow will unregister a workflow from listening to ticks.
func (wf *workflows) UnregisterFromTicksListeningWorkflow(
	ctx workflow.Context,
	params api.UnregisterFromTicksListeningWorkflowParams,
) (api.UnregisterFromTicksListeningWorkflowResults, error) {
	// Ensure the required parameters are provided
	if params.CallbackWorkflowName == "" {
		return api.UnregisterFromTicksListeningWorkflowResults{}, errors.New("callbackWorkflowName must be provided")
	}
	if params.Exchange == "" {
		return api.UnregisterFromTicksListeningWorkflowResults{}, errors.New("exchange must be provided")
	}
	if params.Pair == "" {
		return api.UnregisterFromTicksListeningWorkflowResults{}, errors.New("pair must be provided")
	}

	// Prepare the signal parameters for unregistering
	signalParams := signals.UnregisterFromTicksListeningSignalParams{
		CallbackWorkflowName: params.CallbackWorkflowName,
	}

	// Send the unregister signal to the sentry workflow
	err := workflow.SignalExternalWorkflow(
		ctx,
		sentryWorkflowName(params.Exchange, params.Pair), // Use the sentry workflow ID
		"", // RunID is empty to target the latest run
		signals.UnregisterFromTicksListeningSignalName, // Signal name
		signalParams, // Signal parameters
	).Get(ctx, nil)
	if err != nil {
		// Return an error if signaling fails
		return api.UnregisterFromTicksListeningWorkflowResults{}, err
	}

	// Return an empty result on success
	return api.UnregisterFromTicksListeningWorkflowResults{}, nil
}
