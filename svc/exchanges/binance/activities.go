package binance

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strconv"
	"time"

	client "github.com/adshao/go-binance/v2"
	"github.com/cryptellation/candlesticks/pkg/pair"
	"github.com/cryptellation/ticks/pkg/temporal"
	"github.com/cryptellation/ticks/pkg/tick"
	"github.com/cryptellation/ticks/svc/exchanges"
	"github.com/cryptellation/ticks/svc/internal/signals"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/activity"
	temporalclient "go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
)

// Activities is the struct that will handle the exchanges.
type Activities struct {
	temporal temporalclient.Client
	client   *client.Client
}

// New will create a new binance exchanges.
func New(temporal temporalclient.Client, apiKey, secretKey string) (*Activities, error) {
	// Validate that API key and secret key are not empty
	if apiKey == "" {
		return nil, errors.New("API key cannot be empty")
	}
	if secretKey == "" {
		return nil, errors.New("secret key cannot be empty")
	}

	c := client.NewClient(apiKey, secretKey)
	c.Logger.SetOutput(io.Discard)

	return &Activities{
		temporal: temporal,
		client:   c,
	}, nil
}

// Name will return the name of the exchanges.
func (a *Activities) Name() string {
	return ExchangeName
}

// Register will register the exchanges.
func (a *Activities) Register(w worker.Worker) {
	w.RegisterActivityWithOptions(
		a.ListenSymbolActivity,
		activity.RegisterOptions{Name: exchanges.ListenSymbolActivityName})
}

// ListenSymbolActivity will listen to the symbol activity.
func (a *Activities) ListenSymbolActivity(
	ctx context.Context,
	params exchanges.ListenSymbolParams,
) (exchanges.ListenSymbolResults, error) {
	binanceSymbol, err := toBinanceSymbol(params.Symbol)
	if err != nil {
		return exchanges.ListenSymbolResults{}, err
	}

	// Start heartbeat on activity
	temporal.AsyncActivityHeartbeat(ctx, 300*time.Millisecond)

	// Listen to binance book ticker
	var lastBid, lastAsk string
	done, cancel, err := client.WsBookTickerServe(binanceSymbol, func(event *client.WsBookTickerEvent) {
		// Skip if same price as last tick
		if event.BestAskPrice == lastAsk && event.BestBidPrice == lastBid {
			return
		}
		lastAsk = event.BestAskPrice
		lastBid = event.BestBidPrice

		// Convert to tick
		t, err := toTick(params.Symbol, event.BestAskPrice, event.BestBidPrice)
		if err != nil {
			return
		}

		// Send it to main workflow through Signal
		err = a.temporal.SignalWorkflow(ctx, params.ParentWorkflowID, "", signals.NewTickReceivedSignalName, t)
		if err != nil {
			a.handleNewTickSignalError(ctx, err, params)
		}
	}, nil)
	if err != nil {
		return exchanges.ListenSymbolResults{}, err
	}

	// Wait for context to be done or cancelled
	select {
	case <-done:
		// If done, return error as listener stopped
		return exchanges.ListenSymbolResults{}, fmt.Errorf("binance listener stopped")
	case <-ctx.Done():
		// If context is done, cancel listener and return
		cancel <- struct{}{}
		return exchanges.ListenSymbolResults{}, nil
	}
}

func (a *Activities) handleNewTickSignalError(ctx context.Context, ntsErr error, params exchanges.ListenSymbolParams) {
	// Context was cancelled, this will stop listener
	if errors.Is(ntsErr, context.Canceled) {
		return
	}

	// Check if parent workflow is still running
	desc, err := a.temporal.DescribeWorkflowExecution(ctx, params.ParentWorkflowID, "")
	if err != nil {
		return
	} else if desc.WorkflowExecutionInfo.Status == enums.WORKFLOW_EXECUTION_STATUS_COMPLETED {
		// Workflow is already completed
		return
	}
}

func toTick(symbol, ask, bid string) (tick.Tick, error) {
	askPrice, err := strconv.ParseFloat(ask, 64)
	if err != nil {
		return tick.Tick{}, err
	}

	bidPrice, err := strconv.ParseFloat(bid, 64)
	if err != nil {
		return tick.Tick{}, err
	}

	return tick.Tick{
		Time:     time.Now().UTC(),
		Exchange: "binance",
		Pair:     symbol,
		Price:    (askPrice + bidPrice) / 2,
	}, nil
}

func toBinanceSymbol(symbol string) (string, error) {
	base, quote, err := pair.ParsePair(symbol)
	return fmt.Sprintf("%s%s", base, quote), err
}
