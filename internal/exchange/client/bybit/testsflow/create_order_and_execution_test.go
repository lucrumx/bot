package testsflow

import (
	"os"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"

	"github.com/lucrumx/bot/internal/exchange"
	"github.com/lucrumx/bot/internal/exchange/client/bybit"
	"github.com/lucrumx/bot/internal/models"
	"github.com/lucrumx/bot/internal/utils/testutils"
)

const symbol = "TONUSDT"
const side = models.OrderSideBuy
const market = models.OrderMarketLinear
const orderType = models.OrderTypeMarket
const notional = 7 // ордер на 7 USDT, с учетом плеча

func Test_CreateOrderAndExecution_Integration(t *testing.T) {
	if os.Getenv("INTEGRATION_TEST") != "1" {
		t.Skip("skip: set INTEGRATION_TEST=1 to run")
	}

	logger := zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr}).
		With().Timestamp().Logger()
	cfg := testutils.LoadTestConfig(t)
	ctx := t.Context()

	bybit := bybit.NewByBitClient(cfg, logger)

	// --- Check balance
	balance, err := bybit.GetBalances(ctx)
	require.NoError(t, err, "Error getting balances")
	require.NotEmpty(t, balance, "No balances found")

	var usdtBalance *models.Balance
	// testutils.PrintStruct(t, balance, "Balances")

	for _, b := range balance {
		if b.Asset == "USDT" {
			usdtBalance = &b
		}
	}
	require.NotNil(t, usdtBalance, "USDT balance not found")
	require.True(t, usdtBalance.Free.IsPositive(), "USDT balance must be positive")

	// Get ticker info
	ticker, err := bybit.GetTickers(ctx, []string{symbol}, exchange.CategoryLinear)
	require.NoError(t, err, "Error getting ticker")
	require.NotEmpty(t, ticker, "No ticker found")
	require.Len(t, ticker, 1, "Expected 1 ticker")
	require.Equal(t, symbol, ticker[0].Symbol, "Ticker symbol mismatch")
	// testutils.PrintStruct(t, ticker, "Ticker")

	// Get instrument info
	instruments, err := bybit.GetInstruments(ctx)
	require.NoError(t, err, "Error getting instruments")
	require.NotEmpty(t, instruments, "No instruments found")
	instrument, ok := instruments[symbol]
	require.True(t, ok, "Instrument not found for symbol: %s", symbol)
	// testutils.PrintStruct(t, instrument, "Instrument")

	// Create order
	qty := decimal.NewFromInt(notional).Div(ticker[0].LastPrice).Round(1)
	qty = qty.Div(instrument.VolStep).Floor().Mul(instrument.VolStep) // align to step

	order, err := makeOrder(&ticker[0], bybit, qty)
	require.NoError(t, err, "Error creating order")
	require.NotEmpty(t, order, "Order is empty")
	require.Equal(t, order.Symbol, symbol, "Order symbol mismatch")
	require.NotEqual(t, order.ID, uuid.Nil, "Order side mismatch")

	// Create ws private
	wg := sync.WaitGroup{}
	executionCh, err := bybit.SubscribeExecutions(ctx)
	require.NoError(t, err, "Error subscribing to executions ch")
	go func(order *models.Order) {
		for {
			select {
			case <-ctx.Done():
				return
			case execution := <-executionCh:
				testutils.PrintStruct(t, execution, "Execution")
				order.ID = execution.OrderID
				order.ExchangeOrderID = execution.ExchangeOrderID
				wg.Done()
			}
		}
	}(&order)

	wg.Add(1)
	err = bybit.CreateOrder(ctx, &order)
	require.NoError(t, err, "Error creating order")
	testutils.PrintStruct(t, order, "Order")

	wg.Wait()

	orderInfo, err := bybit.GetOrder(ctx, order.ID, order.ExchangeOrderID, order.Symbol)
	testutils.PrintStruct(t, orderInfo, "Order info")
	require.NoError(t, err, "Error getting order info")
	require.Greater(t, orderInfo.AvgPrice.InexactFloat64(), float64(0), "Avg price should be greater than 0")
	// require.Greater(t, orderInfo.Fees.InexactFloat64(), float64(0), "Fees should be greater than 0")
}

func makeOrder(ticker *exchange.Ticker, provider exchange.Provider, qty decimal.Decimal) (models.Order, error) {
	return exchange.MakeOrderStruct(exchange.CreateOrderDto{
		Market:       market,
		Symbol:       ticker.Symbol,
		Side:         side,
		Type:         orderType,
		Quantity:     qty,
		ExchangeName: provider.GetExchangeName(),
	})
}
