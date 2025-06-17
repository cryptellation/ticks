package clients

import (
	"github.com/cryptellation/ticks/api"
	"go.temporal.io/sdk/workflow"
)

// WfClient is a client for the cryptellation ticks service from a workflow perspective.
type WfClient interface {
	// ListenToTicks listens to ticks from the given exchange and pair.
	ListenToTicks(
		ctx workflow.Context,
		params api.RegisterForTicksListeningWorkflowParams,
	) (api.RegisterForTicksListeningWorkflowResults, error)

	// StopListeningToTicks unregisters a callback workflow from ticks for a given exchange and pair.
	StopListeningToTicks(
		ctx workflow.Context,
		params api.UnregisterFromTicksListeningWorkflowParams,
	) (api.UnregisterFromTicksListeningWorkflowResults, error)
}

type wfClient struct{}

// NewWfClient creates a new workflow client.
// This client is used to call workflows from within other workflows.
// It is not used to call workflows from outside the workflow environment.
func NewWfClient() WfClient {
	return wfClient{}
}

// ListenToTicks listens to ticks from the given exchange and pair.
func (c wfClient) ListenToTicks(
	ctx workflow.Context,
	params api.RegisterForTicksListeningWorkflowParams,
) (api.RegisterForTicksListeningWorkflowResults, error) {
	// Set options
	childWorkflowOptions := workflow.ChildWorkflowOptions{
		TaskQueue: api.WorkerTaskQueueName,
	}
	ctx = workflow.WithChildOptions(ctx, childWorkflowOptions)

	// Execute child workflow
	var res api.RegisterForTicksListeningWorkflowResults
	err := workflow.ExecuteChildWorkflow(ctx, api.RegisterForTicksListeningWorkflowName, params).Get(ctx, &res)
	if err != nil {
		return api.RegisterForTicksListeningWorkflowResults{}, err
	}

	return res, nil
}

// StopListeningToTicks unregisters a callback workflow from ticks for a given exchange and pair.
func (c wfClient) StopListeningToTicks(
	ctx workflow.Context,
	params api.UnregisterFromTicksListeningWorkflowParams,
) (api.UnregisterFromTicksListeningWorkflowResults, error) {
	// Set options
	childWorkflowOptions := workflow.ChildWorkflowOptions{
		TaskQueue: api.WorkerTaskQueueName,
	}
	ctx = workflow.WithChildOptions(ctx, childWorkflowOptions)

	// Execute child workflow
	var res api.UnregisterFromTicksListeningWorkflowResults
	err := workflow.ExecuteChildWorkflow(ctx, api.UnregisterFromTicksListeningWorkflowName, params).Get(ctx, &res)
	if err != nil {
		return api.UnregisterFromTicksListeningWorkflowResults{}, err
	}

	return res, nil
}
