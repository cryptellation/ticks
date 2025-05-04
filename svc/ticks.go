package svc

import (
	exchangesclients "github.com/cryptellation/exchanges/pkg/clients"
	"github.com/cryptellation/ticks/api"
	"github.com/cryptellation/ticks/svc/exchanges"
	"github.com/cryptellation/ticks/svc/internal/activities"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
)

// Ticks is the ticks domain.
type Ticks interface {
	Register(w worker.Worker)

	RegisterForTicksListeningWorkflow(
		ctx workflow.Context,
		params api.RegisterForTicksListeningWorkflowParams,
	) (api.RegisterForTicksListeningWorkflowResults, error)

	UnregisterFromTicksListeningWorkflow(
		ctx workflow.Context,
		params api.UnregisterFromTicksListeningWorkflowParams,
	) (api.UnregisterFromTicksListeningWorkflowResults, error)
}

// Check that the workflows implements the Ticks interface.
var _ Ticks = &workflows{}

type workflows struct {
	exchangesAdapter exchanges.Exchanges
	exchangesSvc     exchangesclients.WfClient
	activities       *activities.Activities
}

// New creates a new ticks workflows.
func New(temporalClient client.Client, exchanges exchanges.Exchanges) Ticks {
	return &workflows{
		exchangesSvc:     exchangesclients.NewWfClient(),
		exchangesAdapter: exchanges,
		activities:       activities.NewActivities(temporalClient),
	}
}

func (wf *workflows) Register(w worker.Worker) {
	// Register common activities
	w.RegisterActivity(wf.activities)

	// Private workflows
	w.RegisterWorkflowWithOptions(wf.ticksSentryWorkflow, workflow.RegisterOptions{
		Name: ticksSentryWorkflowName,
	})

	// Public workflows
	w.RegisterWorkflowWithOptions(wf.RegisterForTicksListeningWorkflow, workflow.RegisterOptions{
		Name: api.RegisterForTicksListeningWorkflowName,
	})
	w.RegisterWorkflowWithOptions(wf.UnregisterFromTicksListeningWorkflow, workflow.RegisterOptions{
		Name: api.UnregisterFromTicksListeningWorkflowName,
	})

	w.RegisterWorkflowWithOptions(ServiceInfoWorkflow, workflow.RegisterOptions{
		Name: api.ServiceInfoWorkflowName,
	})
}
