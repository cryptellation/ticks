package clients

import (
	"context"
	"fmt"
	"strings"

	"github.com/cryptellation/ticks/api"
	temporalutils "github.com/cryptellation/ticks/pkg/temporal"
	"github.com/google/uuid"
	"github.com/iancoleman/strcase"
	temporalclient "go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
)

// ListenerParams holds information for registering a callback workflow.
type ListenerParams struct {
	Name      string
	Callback  func(ctx workflow.Context, params api.ListenToTicksCallbackWorkflowParams) error
	Worker    worker.Worker
	TaskQueue string
}

// Client is a client for the cryptellation ticks service.
type Client interface {
	// ListenToTicks listens to ticks from the given exchange and pair.
	ListenToTicks(
		ctx context.Context,
		listener ListenerParams,
		exchange, pair string,
	) error
	// StopListeningToTicks unregisters a callback workflow from ticks for a given exchange and pair.
	StopListeningToTicks(
		ctx context.Context,
		listener string,
		exchange string,
		pair string,
	) error
	// Info calls the service info.
	Info(ctx context.Context) (api.ServiceInfoResults, error)
	// TemporalClient returns the underlying temporal client.
	TemporalClient() temporalclient.Client
}

type client struct {
	temporal  temporalclient.Client
	userAgent string
}

// ClientOptions holds configuration options for creating a new client.
type ClientOptions struct {
	UserAgent string
}

// New creates a new client to execute temporal workflows.
func New(cl temporalclient.Client, opts ...ClientOptions) Client {
	// Generate a default user agent if none provided
	var agent string
	if len(opts) > 0 && opts[0].UserAgent != "" {
		agent = opts[0].UserAgent
	} else {
		agent = fmt.Sprintf("go-client-%s", uuid.New().String())
	}

	return &client{
		temporal:  cl,
		userAgent: agent,
	}
}

// ListenToTicks listens to ticks from the given exchange and pair.
func (c client) ListenToTicks(
	ctx context.Context,
	listener ListenerParams,
	exchange, pair string,
) error {
	// Require the provided task queue
	if listener.TaskQueue == "" {
		return fmt.Errorf("TaskQueue must be provided in CallbackInfo")
	}

	// Return an error if there is no listener name
	listenerName := listener.Name
	if listenerName == "" {
		return fmt.Errorf("listener name must be provided")
	}

	// Register the workflow with the provided worker
	listener.Worker.RegisterWorkflowWithOptions(listener.Callback, workflow.RegisterOptions{
		Name: listenerName,
	})

	// Listen to ticks
	_, err := c.registerForTicks(ctx,
		api.RegisterForTicksListeningWorkflowParams{
			Exchange: exchange,
			Pair:     pair,
			Callback: temporalutils.CallbackWorkflow{
				Name:          listenerName,
				TaskQueueName: listener.TaskQueue,
			},
		})
	if err != nil {
		return err
	}

	return nil
}

func (c client) registerForTicks(
	ctx context.Context,
	registerParams api.RegisterForTicksListeningWorkflowParams,
) (res api.RegisterForTicksListeningWorkflowResults, err error) {
	// Generate a unique ID for the workflow
	id := fmt.Sprintf(
		"RegisterForTicks%s%s-%s",
		strcase.ToCamel(registerParams.Exchange),
		strings.ReplaceAll(registerParams.Pair, "-", ""),
		c.userAgent,
	)

	// Execute register workflow
	exec, err := c.temporal.ExecuteWorkflow(ctx,
		temporalclient.StartWorkflowOptions{
			ID:        id,
			TaskQueue: api.WorkerTaskQueueName,
		},
		api.RegisterForTicksListeningWorkflowName,
		registerParams)
	if err != nil {
		return api.RegisterForTicksListeningWorkflowResults{}, err
	}

	// Get result and return
	err = exec.Get(ctx, &res)
	return res, err
}

// StopListeningToTicks unregisters a callback workflow from ticks for a given exchange and pair.
func (c client) StopListeningToTicks(
	ctx context.Context,
	listener string,
	exchange string,
	pair string,
) error {
	params := api.UnregisterFromTicksListeningWorkflowParams{
		CallbackWorkflowName: listener,
		Exchange:             exchange,
		Pair:                 pair,
	}

	// Generate a unique ID for the workflow
	id := fmt.Sprintf(
		"UnregisterForTicks%s%s-%s",
		strcase.ToCamel(exchange),
		strings.ReplaceAll(pair, "-", ""),
		c.userAgent,
	)

	// Execute unregister workflow
	exec, err := c.temporal.ExecuteWorkflow(ctx,
		temporalclient.StartWorkflowOptions{
			ID:        id,
			TaskQueue: api.WorkerTaskQueueName,
		},
		api.UnregisterFromTicksListeningWorkflowName,
		params)
	if err != nil {
		return err
	}

	// Wait for the workflow to complete and check for errors
	var res api.UnregisterFromTicksListeningWorkflowResults
	if err := exec.Get(ctx, &res); err != nil {
		return err
	}
	return nil
}

// Info calls the service info.
func (c client) Info(ctx context.Context) (res api.ServiceInfoResults, err error) {
	// Generate a unique ID for the workflow
	id := fmt.Sprintf(
		"Info-%s",
		c.userAgent,
	)

	workflowOptions := temporalclient.StartWorkflowOptions{
		ID:        id,
		TaskQueue: api.WorkerTaskQueueName,
	}

	// Execute workflow
	exec, err := c.temporal.ExecuteWorkflow(ctx, workflowOptions, api.ServiceInfoWorkflowName)
	if err != nil {
		return api.ServiceInfoResults{}, err
	}

	// Get result and return
	err = exec.Get(ctx, &res)
	return res, err
}

func (c client) TemporalClient() temporalclient.Client {
	return c.temporal
}
