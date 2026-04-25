package arbitragebot

import (
	"context"
	"sync"

	"github.com/rs/zerolog"
	"github.com/shopspring/decimal"

	"github.com/lucrumx/bot/internal/exchange"
	"github.com/lucrumx/bot/internal/models"
)

// ExecutionEngine opens and closes arbitrage positions across two exchanges.
type ExecutionEngine struct {
	clients         map[string]exchange.Provider
	instrumentCache *instrumentCache
	orderRepo       OrderRepository
	logger          zerolog.Logger

	mu        sync.Mutex
	positions map[string]*OpenPosition // positionKey -> position
}

// newExecutionEngine creates a new ExecutionEngine.
func newExecutionEngine(clients []exchange.Provider, instruments *instrumentCache, orderRepo OrderRepository, logger zerolog.Logger) *ExecutionEngine {
	clientMap := make(map[string]exchange.Provider, len(clients))
	for _, c := range clients {
		clientMap[c.GetExchangeName()] = c
	}
	return &ExecutionEngine{
		clients:         clientMap,
		instrumentCache: instruments,
		orderRepo:       orderRepo,
		logger:          logger,
		positions:       make(map[string]*OpenPosition),
	}
}

// ListenExecutions subscribes to execution events from all clients and processes them.
func (e *ExecutionEngine) ListenExecutions(ctx context.Context) error {
	for _, client := range e.clients {
		ch, err := client.SubscribeExecutions(ctx)
		if err != nil {
			return err
		}
		if ch == nil {
			continue
		}
		go e.consumeExecutions(ctx, ch)
	}
	return nil
}

func (e *ExecutionEngine) consumeExecutions(ctx context.Context, ch <-chan exchange.OrderExecutionEvent) {
	for {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-ch:
			if !ok {
				return
			}
			e.handleExecution(ctx, event)
		}
	}
}

func (e *ExecutionEngine) handleExecution(ctx context.Context, event exchange.OrderExecutionEvent) {
	e.mu.Lock()
	defer e.mu.Unlock()

	for _, pos := range e.positions {
		if pos.State != PositionStateOpening {
			continue
		}

		if pos.BuyOrderID == event.OrderID {
			pos.BuyConfirmed = true
			e.logger.Info().Str("symbol", pos.Symbol).Str("exchange", pos.BuyExchange).Msg("execution: buy leg confirmed")
			go e.markOrderFilled(ctx, event)
		} else if pos.SellOrderID == event.OrderID {
			pos.SellConfirmed = true
			e.logger.Info().Str("symbol", pos.Symbol).Str("exchange", pos.SellExchange).Msg("execution: sell leg confirmed")
			go e.markOrderFilled(ctx, event)
		} else {
			continue
		}

		// continue to work with position only if both legs are confirmed - this means the position is officially open.
		if !pos.bothConfirmed() {
			continue
		}

		if pos.ShouldClose {
			e.logger.Info().Str("symbol", pos.Symbol).Msg("execution: both legs confirmed, closing immediately (spread already closed)")
			pos.State = PositionStateClosing
			go e.submitCloseLegs(ctx, pos)
		} else {
			pos.State = PositionStateOpen
			e.logger.Info().Str("symbol", pos.Symbol).Msg("execution: position open")
		}

		return
	}
}

// Open submits buy and sell orders for the given spread event.
func (e *ExecutionEngine) Open(ctx context.Context, event *SpreadEvent) {
	e.mu.Lock()
	defer e.mu.Unlock()

	for _, pos := range e.positions {
		if pos.Symbol != event.Symbol {
			continue
		}
		if pos.BuyExchange == event.BuyOnExchange || pos.BuyExchange == event.SellOnExchange ||
			pos.SellExchange == event.BuyOnExchange || pos.SellExchange == event.SellOnExchange {
			e.logger.Debug().
				Str("symbol", event.Symbol).
				Str("buy_on", event.BuyOnExchange).
				Str("sell_on", event.SellOnExchange).
				Msg("execution: symbol already open on one of the exchanges, skipping")
			return
		}
	}

	volStep, err := e.instrumentCache.VolStep(event.Symbol, event.BuyOnExchange, event.SellOnExchange)
	if err != nil {
		e.logger.Error().Err(err).Str("symbol", event.Symbol).Msg("execution: failed to get vol step")
		return
	}

	if event.BuyPrice == 0 {
		e.logger.Error().Str("symbol", event.Symbol).Msg("execution: buy price is zero")
		return
	}

	notional := decimal.NewFromInt(e.notional())
	rawQty := notional.Div(decimal.NewFromFloat(event.BuyPrice))
	qty := AlignOrderQty(rawQty, volStep, volStep)

	if qty.IsZero() {
		e.logger.Error().Str("symbol", event.Symbol).Msg("execution: aligned qty is zero")
		return
	}

	buyOrder, err := e.makeOrder(event.Symbol, models.OrderSideBuy, qty, event.BuyOnExchange)
	if err != nil {
		e.logger.Error().Err(err).Str("symbol", event.Symbol).Msg("execution: failed to make buy order")
		return
	}

	sellOrder, err := e.makeOrder(event.Symbol, models.OrderSideSell, qty, event.SellOnExchange)
	if err != nil {
		e.logger.Error().Err(err).Str("symbol", event.Symbol).Msg("execution: failed to make sell order")
		return
	}

	pos := &OpenPosition{
		Symbol:       event.Symbol,
		BuyExchange:  event.BuyOnExchange,
		SellExchange: event.SellOnExchange,
		Qty:          qty,
		BuyOrderID:   buyOrder.ID,
		SellOrderID:  sellOrder.ID,
		State:        PositionStateOpening,
	}

	e.positions[positionKey(event.Symbol, event.BuyOnExchange, event.SellOnExchange)] = pos

	e.logger.Info().
		Str("symbol", event.Symbol).
		Str("buy_on", event.BuyOnExchange).
		Str("sell_on", event.SellOnExchange).
		Stringer("qty", qty).
		Msg("execution: opening position")

	go e.submitOpenLegs(ctx, pos, &buyOrder, &sellOrder)
}

// Close initiates closing of the position for the given spread.
func (e *ExecutionEngine) Close(ctx context.Context, event *SpreadEvent) {
	e.mu.Lock()
	defer e.mu.Unlock()

	pos, exists := e.positions[positionKey(event.Symbol, event.BuyOnExchange, event.SellOnExchange)]
	if !exists {
		return
	}

	switch pos.State {
	case PositionStateOpening:
		pos.ShouldClose = true
		e.logger.Info().Str("symbol", pos.Symbol).Msg("execution: spread closed while opening, will close after confirmation")
	case PositionStateOpen:
		pos.State = PositionStateClosing
		e.logger.Info().Str("symbol", pos.Symbol).Msg("execution: closing position")
		go e.submitCloseLegs(ctx, pos)
	case PositionStateClosing:
		e.logger.Debug().Str("symbol", pos.Symbol).Msg("execution: already closing")
	}
}

func (e *ExecutionEngine) submitOpenLegs(ctx context.Context, pos *OpenPosition, buyOrder, sellOrder *models.Order) {
	buyClient, sellClient := e.clients[pos.BuyExchange], e.clients[pos.SellExchange]

	var wg sync.WaitGroup
	wg.Add(2)

	var buyErr, sellErr error

	go func() {
		defer wg.Done()
		buyErr = buyClient.CreateOrder(ctx, buyOrder)
	}()

	go func() {
		defer wg.Done()
		sellErr = sellClient.CreateOrder(ctx, sellOrder)
	}()

	wg.Wait()

	if buyErr == nil {
		go e.saveOrder(buyOrder)
	}
	if sellErr == nil {
		go e.saveOrder(sellOrder)
	}

	if buyErr != nil || sellErr != nil {
		e.logger.Error().
			AnErr("buy_err", buyErr).
			AnErr("sell_err", sellErr).
			Str("symbol", pos.Symbol).
			Msg("execution: failed to submit open legs, emergency close")

		e.mu.Lock()
		delete(e.positions, positionKey(pos.Symbol, pos.BuyExchange, pos.SellExchange))
		e.mu.Unlock()

		// Emergency close: close whichever leg succeeded
		if buyErr == nil {
			go func() {
				if err := buyClient.CloseOrder(ctx, buyOrder); err != nil {
					e.logger.Error().Err(err).Str("symbol", pos.Symbol).Msg("execution: emergency close buy leg failed")
				}
			}()
		}
		if sellErr == nil {
			go func() {
				if err := sellClient.CloseOrder(ctx, sellOrder); err != nil {
					e.logger.Error().Err(err).Str("symbol", pos.Symbol).Msg("execution: emergency close sell leg failed")
				}
			}()
		}
	}
}

func (e *ExecutionEngine) submitCloseLegs(ctx context.Context, pos *OpenPosition) {
	buyClient, sellClient := e.clients[pos.BuyExchange], e.clients[pos.SellExchange]

	buyOrder, err := e.makeOrder(pos.Symbol, models.OrderSideBuy, pos.Qty, pos.BuyExchange)
	if err != nil {
		e.logger.Error().Err(err).Str("symbol", pos.Symbol).Msg("execution: failed to make close buy order")
		return
	}
	sellOrder, err := e.makeOrder(pos.Symbol, models.OrderSideSell, pos.Qty, pos.SellExchange)
	if err != nil {
		e.logger.Error().Err(err).Str("symbol", pos.Symbol).Msg("execution: failed to make close sell order")
		return
	}

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		if err := buyClient.CloseOrder(ctx, &buyOrder); err != nil {
			e.logger.Error().Err(err).Str("symbol", pos.Symbol).Str("exchange", pos.BuyExchange).Msg("execution: failed to close buy leg")
		}
	}()

	go func() {
		defer wg.Done()
		if err := sellClient.CloseOrder(ctx, &sellOrder); err != nil {
			e.logger.Error().Err(err).Str("symbol", pos.Symbol).Str("exchange", pos.SellExchange).Msg("execution: failed to close sell leg")
		}
	}()

	wg.Wait()

	e.mu.Lock()
	delete(e.positions, positionKey(pos.Symbol, pos.BuyExchange, pos.SellExchange))
	e.mu.Unlock()

	e.logger.Info().Str("symbol", pos.Symbol).Msg("execution: position closed")
}

func (e *ExecutionEngine) makeOrder(symbol string, side models.OrderSide, qty decimal.Decimal, exchangeName string) (models.Order, error) {
	return exchange.MakeOrderStruct(exchange.CreateOrderDto{
		Symbol:       symbol,
		Side:         side,
		Type:         models.OrderTypeMarket,
		Market:       models.OrderMarketLinear,
		Quantity:     qty,
		ExchangeName: exchangeName,
	})
}

// notional returns the trade size in USDT. Hardcoded for now, will come from config.
func (e *ExecutionEngine) notional() int64 {
	return 10
}

func positionKey(symbol, buyExchange, sellExchange string) string {
	return symbol + "#" + buyExchange + "#" + sellExchange
}

func (e *ExecutionEngine) saveOrder(order *models.Order) {
	if e.orderRepo == nil {
		return
	}
	if err := e.orderRepo.Create(context.Background(), order); err != nil {
		e.logger.Error().Err(err).Str("order_id", order.ID.String()).Msg("execution: failed to save order")
	}
}

func (e *ExecutionEngine) markOrderFilled(ctx context.Context, event exchange.OrderExecutionEvent) {
	if e.orderRepo == nil {
		return
	}
	if err := e.orderRepo.UpdateFilled(ctx, event.OrderID, event.ExecPrice, event.ExecQty); err != nil {
		e.logger.Error().Err(err).Str("order_id", event.OrderID.String()).Msg("execution: failed to mark order filled")
	}
}
