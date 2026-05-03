package arbitragebot

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/shopspring/decimal"

	"github.com/lucrumx/bot/internal/config"
	"github.com/lucrumx/bot/internal/exchange"
	"github.com/lucrumx/bot/internal/models"
	"github.com/lucrumx/bot/internal/notifier"
	"github.com/lucrumx/bot/internal/utils"
)

// Engine handles spread signals, executes arbitrage orders, and sends notifications.
type Engine struct {
	cfg         *config.Config
	clients     map[string]exchange.Provider
	instruments map[string]map[string]exchange.Instrument // instruments[exchange][symbol]
	orderRepo   OrderRepository
	spreadRepo  ArbitrageSpreadRepository
	notif       notifier.Notifier
	logger      zerolog.Logger

	signalCh chan *SpreadEvent

	mu               sync.Mutex
	positions        map[string]*OpenPosition
	blacklistedSyms  map[string]struct{} // символы, для которых торговля невозможна (runtime blacklist)
	chOrdersToUpdate chan uuid.UUID      // channel for order IDs that need to be updated and have their spread profit recalculated
}

type ordersToOpenClosePosition struct {
	buyOrderID  uuid.UUID
	sellOrderID uuid.UUID
}

// NewEngine creates a new Engine.
func NewEngine(
	cfg *config.Config,
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
		cfg:              cfg,
		clients:          clientMap,
		instruments:      make(map[string]map[string]exchange.Instrument),
		orderRepo:        orderRepo,
		spreadRepo:       spreadRepo,
		notif:            notif,
		logger:           logger,
		signalCh:         make(chan *SpreadEvent, 1000),
		positions:        make(map[string]*OpenPosition),
		blacklistedSyms:  make(map[string]struct{}),
		chOrdersToUpdate: make(chan uuid.UUID, 30),
	}
}

// LoadInstruments fetches contract specifications from all exchanges.
// Also syncs the internal client map to match the provided clients list.
// TODO: вынести это из Engine
func (e *Engine) LoadInstruments(ctx context.Context, clients []exchange.Provider) error {
	e.clients = make(map[string]exchange.Provider, len(clients))
	for _, client := range clients {
		e.clients[client.GetExchangeName()] = client

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
			e.logger.Warn().Str("exchange", client.GetExchangeName()).Msg("execution: no execution channel, skipping")
			continue
		}
		e.logger.Info().Str("exchange", client.GetExchangeName()).Msg("execution: listening for executions")
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
	var orders *ordersToOpenClosePosition
	if !e.isSilentMode() {
		orders = e.openPosition(ctx, event)
	}

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
			event.BuyOnExchange, utils.FormatPrice(event.BuyPrice),
			event.SellOnExchange, utils.FormatPrice(event.SellPrice),
		)

		if err := e.notif.Send(msg); err != nil {
			e.logger.Warn().Err(err).Msg("failed to send telegram notification")
		}
	}()

	go func() {
		spread := &models.ArbitrageSpread{
			Symbol:           event.Symbol,
			BuyOnExchange:    event.BuyOnExchange,
			SellOnExchange:   event.SellOnExchange,
			BuyPrice:         decimal.NewFromFloat(event.BuyPrice),
			SellPrice:        decimal.NewFromFloat(event.SellPrice),
			SpreadPercent:    decimal.NewFromFloat(event.FromSpreadPercent),
			MaxSpreadPercent: decimal.NewFromFloat(event.MaxSpreadPercent),
			Status:           event.Status,
		}
		if orders != nil {
			spread.OpenBuyOrderID = orders.buyOrderID
			spread.OpenSellOrderID = orders.sellOrderID
		}
		err := e.spreadRepo.Create(ctx, spread)
		if err != nil {
			e.logger.Warn().Err(err).Msgf("failed to create arbitrage spread, symbol: %s, buy on: %s, sell on: %s",
				event.Symbol, event.BuyOnExchange, event.SellOnExchange)
		}
	}()
}

func (e *Engine) handleUpdate(ctx context.Context, event *SpreadEvent) {
	// TODO наращивать позу при росте спреда?
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
	if !e.isSilentMode() {
		e.closePosition(ctx, event)
	}

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

func (e *Engine) canBeOpened(event *SpreadEvent) bool {
	if _, ok := e.blacklistedSyms[event.Symbol]; ok {
		e.logger.Info().Str("symbol", event.Symbol).Msg("execution: symbol is blacklisted, skipping")
		return false
	}

	if len(e.positions) == 3 {
		e.logger.Info().Msg("execution: already have 3 open positions, skipping new ones")
		return false
	}

	if event.FromSpreadPercent > e.cfg.Exchange.ArbitrageBot.MaxSpreadPercentForOpen {
		spread := strconv.FormatFloat(event.FromSpreadPercent, 'f', 2, 64)
		e.logger.Info().
			Str("spread", spread).
			Str("symbol", event.Symbol).
			Str("buy_on", event.BuyOnExchange).
			Str("sell_on", event.SellOnExchange).
			Msg("execution: spread is too high, skipping")
		return false
	}

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
			return false
		}
	}

	if event.BuyPrice == 0 {
		e.logger.Error().Str("symbol", event.Symbol).Msg("execution: buy price is zero")
		return false
	}

	return true
}

// --- execution ---
// (byuOrderID, sellOrderID)
func (e *Engine) openPosition(ctx context.Context, event *SpreadEvent) *ordersToOpenClosePosition {
	e.mu.Lock()
	defer e.mu.Unlock()

	if !e.canBeOpened(event) {
		return nil
	}

	step, err := e.coinStep(event.Symbol, event.BuyOnExchange, event.SellOnExchange)
	if err != nil {
		e.logger.Error().Err(err).Str("symbol", event.Symbol).Msg("execution: failed to get coin step")
		return nil
	}

	notional := decimal.NewFromInt(e.notional())
	rawQty := notional.Div(decimal.NewFromFloat(event.BuyPrice))
	qty := rawQty.Div(step).Floor().Mul(step) // align to common coin step

	if qty.IsZero() {
		e.logger.Error().Str("symbol", event.Symbol).Msg("execution: aligned qty is zero")
		return nil
	}

	buyVol, err := e.qtyForExchange(qty, event.Symbol, event.BuyOnExchange)
	if err != nil {
		e.logger.Error().Err(err).Str("symbol", event.Symbol).Msg("execution: failed to convert buy qty")
		return nil
	}

	sellVol, err := e.qtyForExchange(qty, event.Symbol, event.SellOnExchange)
	if err != nil {
		e.logger.Error().Err(err).Str("symbol", event.Symbol).Msg("execution: failed to convert sell qty")
		return nil
	}

	buyOrder, err := e.makeOrder(event.Symbol, models.OrderSideBuy, buyVol, event.BuyOnExchange)
	if err != nil {
		e.logger.Error().Err(err).Str("symbol", event.Symbol).Msg("execution: failed to make buy order")
		return nil
	}

	sellOrder, err := e.makeOrder(event.Symbol, models.OrderSideSell, sellVol, event.SellOnExchange)
	if err != nil {
		e.logger.Error().Err(err).Str("symbol", event.Symbol).Msg("execution: failed to make sell order")
		return nil
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
		Stringer("qty_coins", qty).
		Stringer("buy_vol", buyVol).
		Stringer("sell_vol", sellVol).
		Msg("🚀 execution: opening position")

	go e.submitOpenLegs(ctx, pos, &buyOrder, &sellOrder)

	return &ordersToOpenClosePosition{
		buyOrderID:  buyOrder.ID,
		sellOrderID: sellOrder.ID,
	}
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
		e.logger.Info().Str("symbol", pos.Symbol).Msg("🔻 execution: closing position")
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
			e.logger.Info().Str("symbol", pos.Symbol).Str("exchange", pos.BuyExchange).Msg("✅ execution: buy leg confirmed")
			go e.markOrderFilled(ctx, event)
		} else if pos.SellOrderID == event.OrderID {
			pos.SellConfirmed = true
			e.logger.Info().Str("symbol", pos.Symbol).Str("exchange", pos.SellExchange).Msg("✅ execution: sell leg confirmed")
			go e.markOrderFilled(ctx, event)
		} else {
			continue
		}

		e.chOrdersToUpdate <- event.OrderID

		if !pos.bothConfirmed() {
			return
		}

		if pos.State == PositionStateOpeningPendingClose {
			e.logger.Info().Str("symbol", pos.Symbol).Msg("⚡ execution: both legs confirmed, closing immediately (spread already closed)")
			pos.State = PositionStateClosing
			go e.submitCloseLegs(ctx, pos)
		} else {
			pos.State = PositionStateOpen
			e.logger.Info().Str("symbol", pos.Symbol).Msg("✅ execution: position open")
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
		// Если была ошибка при открытии сделки на какой-нибудь ноге - символ в блэклист
		e.blacklistedSyms[pos.Symbol] = struct{}{}
		e.mu.Unlock()

		e.logger.Warn().Str("symbol", pos.Symbol).Msg("execution: symbol blacklisted after failed open")

		if buyErr == nil {
			go func() {
				closeOrder := *buyOrder
				closeOrder.ID = uuid.New()
				if err := buyClient.CloseOrder(ctx, &closeOrder); err != nil {
					e.logger.Error().Err(err).Str("symbol", pos.Symbol).Msg("execution: emergency close buy leg failed")
				}
			}()
		}
		if sellErr == nil {
			go func() {
				closeOrder := *sellOrder
				closeOrder.ID = uuid.New()
				if err := sellClient.CloseOrder(ctx, &closeOrder); err != nil {
					e.logger.Error().Err(err).Str("symbol", pos.Symbol).Msg("execution: emergency close sell leg failed")
				}
			}()
		}
	}
}

func (e *Engine) submitCloseLegs(ctx context.Context, pos *OpenPosition) {
	buyClient, sellClient := e.clients[pos.BuyExchange], e.clients[pos.SellExchange]

	buyVol, err := e.qtyForExchange(pos.Qty, pos.Symbol, pos.BuyExchange)
	if err != nil {
		e.logger.Error().Err(err).Str("symbol", pos.Symbol).Msg("execution: failed to convert close buy qty")
		return
	}
	sellVol, err := e.qtyForExchange(pos.Qty, pos.Symbol, pos.SellExchange)
	if err != nil {
		e.logger.Error().Err(err).Str("symbol", pos.Symbol).Msg("execution: failed to convert close sell qty")
		return
	}

	buyOrder, err := e.makeOrder(pos.Symbol, models.OrderSideBuy, buyVol, pos.BuyExchange)
	if err != nil {
		e.logger.Error().Err(err).Str("symbol", pos.Symbol).Msg("execution: failed to make close buy order")
		return
	}
	sellOrder, err := e.makeOrder(pos.Symbol, models.OrderSideSell, sellVol, pos.SellExchange)
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

	e.logger.Info().Str("symbol", pos.Symbol).Msg("💰 execution: position closed")

	// update spread, set close order IDs for profit calculation
	go func() {
		e.saveOrder(&buyOrder)
		e.saveOrder(&sellOrder)

		spread, err := e.spreadRepo.FindOne(ctx, FindFilter{
			Symbol: pos.Symbol,
			BuyEx:  pos.BuyExchange,
			SellEx: pos.SellExchange,
		})
		if err != nil {
			e.logger.Warn().Err(err).Msgf("failed to find arbitrage spread for update after close, symbol: %s, buy exchange: %s, sell exchange: %s",
				pos.Symbol, pos.BuyExchange, pos.SellExchange)
			return
		}

		if spread.CloseBuyOrderID != uuid.Nil || spread.CloseSellOrderID != uuid.Nil {
			e.logger.Warn().Msgf("arbitrage spread already has close order IDs, skipping update, symbol: %s, buy exchange: %s, sell exchange: %s",
				pos.Symbol, pos.BuyExchange, pos.SellExchange)
			return
		}

		err = e.spreadRepo.Update(ctx, &models.ArbitrageSpread{
			CloseBuyOrderID:  buyOrder.ID,
			CloseSellOrderID: sellOrder.ID,
			UpdatedAt:        time.Now(),
		}, FindFilter{
			ID: spread.ID,
		})
		if err != nil {
			e.logger.Warn().Err(err).Msgf("failed to update arbitrage spread, symbol: %s, buy exchange: %s, sell exchange: %s",
				pos.Symbol, pos.BuyExchange, pos.SellExchange)
			return
		}

		e.chOrdersToUpdate <- buyOrder.ID
		e.chOrdersToUpdate <- sellOrder.ID
	}()
}

// --- helpers ---

// coinStep returns the common step size in coins that satisfies both exchanges.
// For exchanges with contractSize > 1 (e.g. MEXC), the effective step in coins = volStep * contractSize.
func (e *Engine) coinStep(symbol, exchange1, exchange2 string) (decimal.Decimal, error) {
	inst1, err := e.instrumentFor(symbol, exchange1)
	if err != nil {
		return decimal.Zero, err
	}
	inst2, err := e.instrumentFor(symbol, exchange2)
	if err != nil {
		return decimal.Zero, err
	}
	step1 := inst1.VolStep.Mul(inst1.ContractSize)
	step2 := inst2.VolStep.Mul(inst2.ContractSize)
	return lcmDecimal(step1, step2), nil
}

func (e *Engine) instrumentFor(symbol, exchangeName string) (exchange.Instrument, error) {
	instruments, ok := e.instruments[exchangeName]
	if !ok {
		return exchange.Instrument{}, fmt.Errorf("no instrument data for exchange %s", exchangeName)
	}
	instrument, ok := instruments[symbol]
	if !ok {
		return exchange.Instrument{}, fmt.Errorf("no instrument %s on %s", symbol, exchangeName)
	}
	return instrument, nil
}

// CommonSymbols возвращает символы, которые есть в instrument cache всех активных бирж.
func (e *Engine) CommonSymbols() map[string]struct{} {
	result := map[string]struct{}{}
	first := true

	for exchangeName := range e.clients {
		instruments := e.instruments[exchangeName]
		if first {
			for symbol := range instruments {
				result[symbol] = struct{}{}
			}
			first = false
			continue
		}
		for s := range result {
			if _, ok := instruments[s]; !ok {
				delete(result, s)
			}
		}
	}

	return result
}

// qtyForExchange converts qty in coins to the exchange-specific vol (qty / contractSize).
func (e *Engine) qtyForExchange(qty decimal.Decimal, symbol, exchangeName string) (decimal.Decimal, error) {
	inst, err := e.instrumentFor(symbol, exchangeName)
	if err != nil {
		return decimal.Zero, err
	}
	return qty.Div(inst.ContractSize), nil
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

// gcdDecimal вычисляет наибольший общий делитель двух десятичных чисел (алгоритм Евклида).
func gcdDecimal(a, b decimal.Decimal) decimal.Decimal {
	for !b.IsZero() {
		a, b = b, a.Mod(b)
	}
	return a
}

// lcmDecimal вычисляет наименьшее общее кратное двух десятичных чисел.
// Используется для нахождения минимального шага в монетах, кратного шагам обеих бирж.
func lcmDecimal(a, b decimal.Decimal) decimal.Decimal {
	return a.Mul(b).Div(gcdDecimal(a, b))
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

func (e *Engine) isSilentMode() bool {
	return e.cfg.Exchange.ArbitrageBot.SilentMode
}
