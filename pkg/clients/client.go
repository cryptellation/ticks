package clients

import (
	"context"
	"fmt"

	"github.com/cryptellation/ticks/api"
	temporalutils "github.com/cryptellation/ticks/pkg/temporal"
	"github.com/google/uuid"
	temporalclient "go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
)

// Client is a client for the cryptellation ticks service.
type Client interface {
	// ListenToTicks listens to ticks from the given exchange and pair.
	ListenToTicks(
		ctx context.Context,
		exchange, pair string,
		callback func(ctx workflow.Context, params api.ListenToTicksCallbackWorkflowParams) error,
	) error
	// Info calls the service info.
	Info(ctx context.Context) (api.ServiceInfoResults, error)
}

type client struct {
	temporal temporalclient.Client
}

// New creates a new client to execute temporal workflows.
func New(cl temporalclient.Client) Client {
	return &client{temporal: cl}
}

// ListenToTicks listens to ticks from the given exchange and pair.
func (c client) ListenToTicks(
	ctx context.Context,
	exchange, pair string,
	callback func(ctx workflow.Context, params api.ListenToTicksCallbackWorkflowParams) error,
) error {
	// TODO: get worker from parameters instead of creating a new one

	// Create variables
	tq := fmt.Sprintf("ListenTicks-%s", uuid.New().String())
	workflowName := tq

	// Create temporary worker
	w := worker.New(c.temporal, tq, worker.Options{})
	w.RegisterWorkflowWithOptions(callback, workflow.RegisterOptions{
		Name: workflowName,
	})

	// Start worker
	go func() {
		if err := w.Run(nil); err != nil {
			panic(err)
		}
	}()
	defer w.Stop()

	// Listen to ticks
	_, err := c.registerForTicks(ctx,
		api.RegisterForTicksListeningWorkflowParams{
			Exchange: exchange,
			Pair:     pair,
			Callback: temporalutils.CallbackWorkflow{
				Name:          workflowName,
				TaskQueueName: tq,
			},
		})
	if err != nil {
		return err
	}

	// Wait for interrupt
	<-ctx.Done()

	return nil
}

func (c client) registerForTicks(
	ctx context.Context,
	registerParams api.RegisterForTicksListeningWorkflowParams,
) (res api.RegisterForTicksListeningWorkflowResults, err error) {
	// Execute register workflow
	exec, err := c.temporal.ExecuteWorkflow(ctx,
		temporalclient.StartWorkflowOptions{
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

// Info calls the service info.
func (c client) Info(ctx context.Context) (res api.ServiceInfoResults, err error) {
	workflowOptions := temporalclient.StartWorkflowOptions{
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
