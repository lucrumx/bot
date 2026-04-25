package arbitragebot

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"github.com/shopspring/decimal"

	"github.com/lucrumx/bot/internal/exchange"
	"github.com/lucrumx/bot/internal/models"
	"github.com/lucrumx/bot/internal/notifier"
)

// Engine handles spread signals, executes arbitrage orders, and sends notifications.
type Engine struct {
	clients     map[string]exchange.Provider
	instruments map[string]map[string]exchange.Instrument // instruments[exchange][symbol]
	orderRepo   OrderRepository
	spreadRepo  ArbitrageSpreadRepository
	notif       notifier.Notifier
	logger      zerolog.Logger

	signalCh chan *SpreadEvent

	mu        sync.Mutex
	positions map[string]*OpenPosition
}

// NewEngine creates a new Engine.
func NewEngine(
	clients []exchange.Provider,
	orderRepo OrderRepository,
	spreadRepo ArbitrageSpreadRepository,
	notif notifier.Notifier,
	logger zerolog.Logger,
) *Engine {
	clientMap := make(map[string]exchange.Provider, len(clients))
	for _, c := range clients {
		clientMap[c.GetExchangeName()] = c
	}
	return &Engine{
		clients:     clientMap,
		instruments: make(map[string]map[string]exchange.Instrument),
		orderRepo:   orderRepo,
		spreadRepo:  spreadRepo,
		notif:       notif,
		logger:      logger,
		signalCh:    make(chan *SpreadEvent, 1000),
		positions:   make(map[string]*OpenPosition),
	}
}

// LoadInstruments fetches contract specifications from all exchanges.
func (e *Engine) LoadInstruments(ctx context.Context, clients []exchange.Provider) error {
	for _, client := range clients {
		instruments, err := client.GetInstruments(ctx)
		if err != nil {
			return fmt.Errorf("failed to load instruments from %s: %w", client.GetExchangeName(), err)
		}
		e.instruments[client.GetExchangeName()] = instruments
	}
	return nil
}

// ListenExecutions subscribes to execution events from all clients.
func (e *Engine) ListenExecutions(ctx context.Context) error {
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

// Run processes spread signals from the channel.
func (e *Engine) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case event := <-e.signalCh:
			switch event.Status {
			case models.ArbitrageSpreadOpened:
				e.handleOpen(ctx, event)
			case models.ArbitrageSpreadUpdated:
				e.handleUpdate(ctx, event)
			case models.ArbitrageSpreadClosed:
				e.handleClose(ctx, event)
			}
		}
	}
}

// HandleSignal enqueues spread events for processing.
func (e *Engine) HandleSignal(events []*SpreadEvent) {
	for _, ev := range events {
		e.signalCh <- ev
	}
}

// --- signal handling ---

func (e *Engine) handleOpen(ctx context.Context, event *SpreadEvent) {
	e.openPosition(ctx, event)

	go func() {
		spreadStr := strconv.FormatFloat(event.FromSpreadPercent, 'f', 2, 64)

		e.logger.Warn().
			Str("pair", event.Symbol).
			Str("spread", spreadStr).
			Str("buy on", event.BuyOnExchange).
			Str("sell on", event.SellOnExchange).
			Msg("🔥 SPREAD DETECTED")

		msg := fmt.Sprintf(
			"<b>🔔 ARBITRAGE: Ticker - %s</b>\n\n"+
				"Spread: <code>%s%%</code>\n\n"+
				"🟢 Buy:  %s - <b>%s</b>\n"+
				"🔴 Sell: %s - <b>%s</b>",
			event.Symbol,
			spreadStr,
			event.BuyOnExchange, formatPrice(event.BuyPrice),
			event.SellOnExchange, formatPrice(event.SellPrice),
		)

		if err := e.notif.Send(msg); err != nil {
			e.logger.Warn().Err(err).Msg("failed to send telegram notification")
		}
	}()

	go func() {
		err := e.spreadRepo.Create(ctx, &models.ArbitrageSpread{
			Symbol:           event.Symbol,
			BuyOnExchange:    event.BuyOnExchange,
			SellOnExchange:   event.SellOnExchange,
			BuyPrice:         decimal.NewFromFloat(event.BuyPrice),
			SellPrice:        decimal.NewFromFloat(event.SellPrice),
			SpreadPercent:    decimal.NewFromFloat(event.FromSpreadPercent),
			MaxSpreadPercent: decimal.NewFromFloat(event.MaxSpreadPercent),
			Status:           event.Status,
		})
		if err != nil {
			e.logger.Warn().Err(err).Msgf("failed to create arbitrage spread, symbol: %s, buy on: %s, sell on: %s",
				event.Symbol, event.BuyOnExchange, event.SellOnExchange)
		}
	}()
}

func (e *Engine) handleUpdate(ctx context.Context, event *SpreadEvent) {
	go func() {
		spreadStr := strconv.FormatFloat(event.MaxSpreadPercent, 'f', 2, 64)

		e.logger.Warn().
			Str("pair", event.Symbol).
			Str("spread", spreadStr).
			Str("buy on", event.BuyOnExchange).
			Str("sell on", event.SellOnExchange).
			Msg("🔥 SPREAD GROWING")

		msg := fmt.Sprintf(
			"<b>🔔 ARBITRAGE: Ticker - %s</b>\n\n"+
				"Spread is growing, new spread: <code>%s%%</code>",
			event.Symbol,
			spreadStr,
		)

		if err := e.notif.Send(msg); err != nil {
			e.logger.Warn().Err(err).Msg("failed to send telegram notification")
		}
	}()

	go func() {
		err := e.spreadRepo.Update(ctx, &models.ArbitrageSpread{
			Status:           models.ArbitrageSpreadUpdated,
			MaxSpreadPercent: decimal.NewFromFloat(event.MaxSpreadPercent),
			UpdatedAt:        time.Now(),
		}, FindFilter{
			Symbol: event.Symbol,
			BuyEx:  event.BuyOnExchange,
			SellEx: event.SellOnExchange,
			Status: []models.ArbitrageSpreadStatus{models.ArbitrageSpreadOpened, models.ArbitrageSpreadUpdated},
		})
		if err != nil {
			e.logger.Warn().Err(err).Msgf("failed to update arbitrage spread, symbol: %s, buy exchange: %s, sell exchange: %s",
				event.Symbol, event.BuyOnExchange, event.SellOnExchange)
		}
	}()
}

func (e *Engine) handleClose(ctx context.Context, event *SpreadEvent) {
	e.closePosition(ctx, event)

	go func() {
		e.logger.Warn().
			Str("pair", event.Symbol).
			Msg("🔥 SPREAD CLOSED")

		msg := fmt.Sprintf(
			"<b>🔔 ARBITRAGE: Ticker - %s</b>\n\n"+
				"Spread closed",
			event.Symbol,
		)

		if err := e.notif.Send(msg); err != nil {
			e.logger.Warn().Err(err).Msg("failed to send telegram notification")
		}
	}()

	go func() {
		err := e.spreadRepo.Update(ctx, &models.ArbitrageSpread{
			Status:    models.ArbitrageSpreadClosed,
			UpdatedAt: time.Now(),
		}, FindFilter{
			Symbol: event.Symbol,
			BuyEx:  event.BuyOnExchange,
			SellEx: event.SellOnExchange,
			Status: []models.ArbitrageSpreadStatus{models.ArbitrageSpreadOpened, models.ArbitrageSpreadUpdated},
		})
		if err != nil {
			e.logger.Warn().Err(err).Msgf("failed to update arbitrage spread, symbol: %s, buy exchange: %s, sell exchange: %s",
				event.Symbol, event.BuyOnExchange, event.SellOnExchange)
		}
	}()
}

func formatPrice(price float64) string {
	switch {
	case price == 0:
		return "0"
	case math.Abs(price) >= 1000:
		return strconv.FormatFloat(price, 'f', 2, 64)
	case math.Abs(price) >= 1:
		return strconv.FormatFloat(price, 'f', 4, 64)
	case math.Abs(price) >= 0.01:
		return strconv.FormatFloat(price, 'f', 6, 64)
	default:
		return strconv.FormatFloat(price, 'f', 8, 64)
	}
}

// --- execution ---

func (e *Engine) openPosition(ctx context.Context, event *SpreadEvent) {
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

	volStep, err := e.volStep(event.Symbol, event.BuyOnExchange, event.SellOnExchange)
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

func (e *Engine) closePosition(ctx context.Context, event *SpreadEvent) {
	e.mu.Lock()
	defer e.mu.Unlock()

	pos, exists := e.positions[positionKey(event.Symbol, event.BuyOnExchange, event.SellOnExchange)]
	if !exists {
		return
	}

	switch pos.State {
	case PositionStateOpening:
		pos.State = PositionStateOpeningPendingClose
		e.logger.Info().Str("symbol", pos.Symbol).Msg("execution: spread closed while opening, will close after confirmation")
	case PositionStateOpen:
		pos.State = PositionStateClosing
		e.logger.Info().Str("symbol", pos.Symbol).Msg("execution: closing position")
		go e.submitCloseLegs(ctx, pos)
	case PositionStateClosing, PositionStateOpeningPendingClose:
		e.logger.Debug().Str("symbol", pos.Symbol).Msg("execution: already closing or pending close")
	}
}

func (e *Engine) consumeExecutions(ctx context.Context, ch <-chan exchange.OrderExecutionEvent) {
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

func (e *Engine) handleExecution(ctx context.Context, event exchange.OrderExecutionEvent) {
	e.mu.Lock()
	defer e.mu.Unlock()

	for _, pos := range e.positions {
		if pos.State != PositionStateOpening && pos.State != PositionStateOpeningPendingClose {
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

		if !pos.bothConfirmed() {
			return
		}

		if pos.State == PositionStateOpeningPendingClose {
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

func (e *Engine) submitOpenLegs(ctx context.Context, pos *OpenPosition, buyOrder, sellOrder *models.Order) {
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

func (e *Engine) submitCloseLegs(ctx context.Context, pos *OpenPosition) {
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

// --- helpers ---

func (e *Engine) volStep(symbol, exchange1, exchange2 string) (decimal.Decimal, error) {
	step1, err := e.volStepFor(symbol, exchange1)
	if err != nil {
		return decimal.Zero, err
	}
	step2, err := e.volStepFor(symbol, exchange2)
	if err != nil {
		return decimal.Zero, err
	}
	return decimal.Max(step1, step2), nil
}

func (e *Engine) volStepFor(symbol, exchangeName string) (decimal.Decimal, error) {
	instruments, ok := e.instruments[exchangeName]
	if !ok {
		return decimal.Zero, fmt.Errorf("no instrument data for exchange %s", exchangeName)
	}
	instrument, ok := instruments[symbol]
	if !ok {
		return decimal.Zero, fmt.Errorf("no instrument %s on %s", symbol, exchangeName)
	}
	return instrument.VolStep, nil
}

func (e *Engine) makeOrder(symbol string, side models.OrderSide, qty decimal.Decimal, exchangeName string) (models.Order, error) {
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
func (e *Engine) notional() int64 {
	return 10
}

func positionKey(symbol, buyExchange, sellExchange string) string {
	return symbol + "#" + buyExchange + "#" + sellExchange
}

func (e *Engine) saveOrder(order *models.Order) {
	if e.orderRepo == nil {
		return
	}
	if err := e.orderRepo.Create(context.Background(), order); err != nil {
		e.logger.Error().Err(err).Str("order_id", order.ID.String()).Msg("execution: failed to save order")
	}
}

func (e *Engine) markOrderFilled(ctx context.Context, event exchange.OrderExecutionEvent) {
	if e.orderRepo == nil {
		return
	}
	if err := e.orderRepo.UpdateFilled(ctx, event.OrderID, event.ExecPrice, event.ExecQty); err != nil {
		e.logger.Error().Err(err).Str("order_id", event.OrderID.String()).Msg("execution: failed to mark order filled")
	}
}
