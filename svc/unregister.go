package svc

import (
	"github.com/cryptellation/ticks/api"
	"go.temporal.io/sdk/workflow"
)

// UnregisterFromTicksListeningWorkflow will unregister a workflow from listening to ticks.
func (wf *workflows) UnregisterFromTicksListeningWorkflow(
	_ workflow.Context,
	_ api.UnregisterFromTicksListeningWorkflowParams,
) (api.UnregisterFromTicksListeningWorkflowResults, error) {
	// TODO(#8): Implement UnregisterFromTicksListeningWorkflow
	return api.UnregisterFromTicksListeningWorkflowResults{}, nil
}
