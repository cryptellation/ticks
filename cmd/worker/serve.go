package main

import (
	"context"

	"github.com/cenkalti/backoff/v5"
	"github.com/cryptellation/health"
	"github.com/cryptellation/ticks/api"
	"github.com/cryptellation/ticks/configs"
	"github.com/cryptellation/ticks/svc"
	"github.com/cryptellation/ticks/svc/exchanges/aggregator"
	"github.com/cryptellation/ticks/svc/exchanges/binance"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.temporal.io/sdk/client"
	temporalwk "go.temporal.io/sdk/worker"
	"golang.org/x/sync/errgroup"
)

var serveCmd = &cobra.Command{
	Use:     "serve",
	Aliases: []string{"s"},
	Short:   "Launch the service",
	RunE:    serve,
}

func serve(cmd *cobra.Command, _ []string) error {
	eg, ctx := errgroup.WithContext(cmd.Context())

	// Init health server
	h, err := health.New(
		viper.GetString(configs.EnvHealthAddress),
	)
	if err != nil {
		return err
	}

	// Start health server
	eg.Go(func() error {
		return h.Serve()
	})
	defer func() {
		_ = h.Shutdown(ctx)
	}()

	// Create temporal client
	temporalClient, err := createTemporalClient(ctx)
	if err != nil {
		return err
	}
	defer temporalClient.Close()

	// Create a worker
	w := temporalwk.New(temporalClient, api.WorkerTaskQueueName, temporalwk.Options{})

	// Create exchange adapter aggregator and register activities
	exchs := aggregator.New(binance.New(
		temporalClient,
		viper.GetString(configs.EnvBinanceAPIKey),
		viper.GetString(configs.EnvBinanceSecretKey),
	))
	exchs.Register(w)

	// Create service core and register workflows
	service := svc.New(temporalClient, exchs)
	service.Register(w)

	// Run worker
	eg.Go(func() error {
		return w.Run(nil)
	})
	defer w.Stop()

	// Mark as ready
	h.Ready(true)
	defer h.Ready(false)

	// Wait for the context to be done
	// This will block until the context is done, or an error occurs
	return eg.Wait()
}

func createTemporalClient(ctx context.Context) (client.Client, error) {
	// Set backoff callback
	callback := func() (client.Client, error) {
		return client.Dial(client.Options{
			HostPort: viper.GetString(configs.EnvTemporalAddress),
		})
	}

	// Retry with backoff
	return backoff.Retry(ctx, callback,
		backoff.WithBackOff(backoff.NewExponentialBackOff()),
		backoff.WithMaxTries(10))
}
