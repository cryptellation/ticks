package svc

import (
	"fmt"

	exchangesapi "github.com/cryptellation/exchanges/api"
	"github.com/cryptellation/ticks/api"
	"github.com/cryptellation/ticks/svc/internal/activities"
	"github.com/cryptellation/ticks/svc/internal/signals"
	"go.temporal.io/sdk/workflow"
)

func (wf *workflows) RegisterForTicksListeningWorkflow(
	ctx workflow.Context,
	params api.RegisterForTicksListeningWorkflowParams,
) (api.RegisterForTicksListeningWorkflowResults, error) {
	// Check if exchange+pair exists
	if err := wf.checkPairAndExchange(ctx, params.Pair, params.Exchange); err != nil {
		return api.RegisterForTicksListeningWorkflowResults{}, err
	}

	// Send signal-with-start to listen for ticks
	if err := activities.ExecuteSignalWithStart(ctx, activities.SignalWithStartActivityParams{
		SignalName: signals.RegisterToTicksListeningSignalName,
		SignalParams: signals.RegisterToTicksListeningSignalParams{
			CallbackWorkflow: params.Callback,
		},
		WorkflowID:   sentryWorkflowName(params.Exchange, params.Pair),
		WorkflowName: ticksSentryWorkflowName,
		WorkflowParams: ticksSentryWorkflowParams{
			Exchange: params.Exchange,
			Symbol:   params.Pair,
		},
		TaskQueue: api.WorkerTaskQueueName,
	}); err != nil {
		return api.RegisterForTicksListeningWorkflowResults{}, err
	}

	return api.RegisterForTicksListeningWorkflowResults{}, nil
}

func (wf *workflows) checkPairAndExchange(ctx workflow.Context, pair string, exchange string) error {
	// Get exchange info
	result, err := wf.exchangesSvc.GetExchange(ctx, exchangesapi.GetExchangeWorkflowParams{
		Name: exchange,
	}, &workflow.ChildWorkflowOptions{
		TaskQueue: exchangesapi.WorkerTaskQueueName,
	})
	if err != nil {
		return err
	}

	// Check if pair exists
	for _, p := range result.Exchange.Pairs {
		if p == pair {
			return nil
		}
	}

	return fmt.Errorf("pair %q doesn't exist for exchange %q", pair, exchange)
}
