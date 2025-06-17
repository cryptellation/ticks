//go:build e2e
// +build e2e

package test

import (
	"context"
	"time"

	"github.com/cryptellation/ticks/api"
	"github.com/cryptellation/ticks/pkg/clients"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
)

// TestListenToTicksWithStopListening tests that StopListeningToTicks stops the listener as expected.
func (suite *EndToEndSuite) TestListenToTicks() {
	exchange := "binance"
	pair := "BTC-USDT"
	count := 0

	// Create a worker
	tq := "TestListenToTicks-TaskQueue"
	w := worker.New(suite.client.TemporalClient(), tq, worker.Options{})
	defer w.Stop()
	go func() {
		suite.Require().NoError(w.Run(nil))
	}()

	// Create a new client with user agent
	client := clients.New(suite.client.TemporalClient(), clients.ClientOptions{
		UserAgent: "TestListenToTicks",
	})

	// Prepare callback params
	params := clients.ListenerParams{
		Name: "TestListenToTicks",
		Callback: func(_ workflow.Context, params api.ListenToTicksCallbackWorkflowParams) error {
			suite.Require().Equal(exchange, params.Tick.Exchange)
			suite.Require().Equal(pair, params.Tick.Pair)
			count++
			return nil
		},
		Worker:    w,
		TaskQueue: tq,
	}

	// Start listening to ticks
	err := client.ListenToTicks(context.Background(), params, exchange, pair)
	suite.Require().NoError(err)

	// Wait until at least one tick is received
	suite.Eventually(func() bool {
		return count > 0
	}, 10*time.Minute, time.Second,
		"count should be greater than 0")

	// Stop listening
	err = client.StopListeningToTicks(context.Background(), params.Name, exchange, pair)
	suite.Require().NoError(err)

	// Wait a short period and check that the count does not increase further
	prevCount := count
	time.Sleep(5 * time.Second)
	suite.Require().Equal(prevCount, count, "count should not increase after stopping listening")
}
