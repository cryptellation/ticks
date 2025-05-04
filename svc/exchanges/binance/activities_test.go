//go:build integration
// +build integration

package binance

import (
	"testing"

	"github.com/cryptellation/ticks/configs"
	"github.com/cryptellation/ticks/pkg/temporal"
	"github.com/cryptellation/ticks/svc/exchanges"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/suite"
	temporalclient "go.temporal.io/sdk/client"
	"go.uber.org/mock/gomock"
)

func TestBinanceSuite(t *testing.T) {
	suite.Run(t, new(BinanceSuite))
}

type BinanceSuite struct {
	suite.Suite
	temporal   temporalclient.Client
	activities exchanges.Exchanges
}

func (suite *BinanceSuite) SetupTest() {
	suite.temporal = temporal.NewMockClient(gomock.NewController(suite.T()))
	acts := New(suite.temporal,
		viper.GetString(configs.EnvBinanceAPIKey),
		viper.GetString(configs.EnvBinanceSecretKey))
	suite.activities = acts
}

func (suite *BinanceSuite) TestTicks() {
	// TODO: Implement this test
}
