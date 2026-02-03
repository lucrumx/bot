package engine

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/lucrumx/bot/internal/exchange"
)

type worker struct {
	id      int
	bot     *Bot
	inChan  chan exchange.Trade
	windows map[string]*Window
}

func (w *worker) workerStart(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case trade, ok := <-w.inChan:
			if !ok {
				return
			}
			w.processTrade(trade)
		}
	}
}

func (w *worker) processTrade(trade exchange.Trade) {
	window, ok := w.windows[trade.Symbol]
	if !ok {
		window = NewWindow(w.bot.pumpInterval)
		w.windows[trade.Symbol] = window
	}

	window.AddTrade(trade)

	atomic.AddUint64(&w.bot.tradeCounter, 1)

	w.checkPump(trade.Symbol, window)
}

func (w *worker) checkPump(symbol string, win *Window) {
	if time.Since(w.bot.startTime) < w.bot.startupDelay {
		return
	}

	// Throttling
	if !win.CanCheck(w.bot.checkInterval) {
		return
	}

	change, isGrow := win.CheckGrow(w.bot.pumpInterval, w.bot.targetPriceChange)
	if !isGrow {
		return
	}

	lastAlertTime, lastAlertLevel := win.GetAlertState()

	// Новый это памп или продолжение старого
	// Если с прошлого алерта прошло времени больше, чем длина окна,
	// значит старый памп закончился, поймали новый.
	isNewPump := time.Since(lastAlertTime) > time.Duration(w.bot.pumpInterval)*time.Second

	needAlert := false

	if isNewPump {
		needAlert = true
	} else {
		// Памп продолжается. Проверяем, выросли ли мы на "шаг" (например, +5%)
		// Текущий рост >= Прошлый уровень + Шаг
		// Пример: 22% >= 15% + 5% -> True
		nextThreshold := lastAlertLevel.Add(w.bot.alertStep)
		if change.GreaterThanOrEqual(nextThreshold) {
			needAlert = true
		}
	}

	if needAlert {
		win.UpdateAlertState(change)

		priceChangePct := change.StringFixed(2) + "%"

		w.bot.logger.Warn().
			Str("pair", symbol).
			Str("change", priceChangePct).
			Msg("🔥 PUMP DETECTED")

		msg := fmt.Sprintf(
			"<b>🚀 PUMP DETECTED: <a href=\"https://www.bybit.com/trade/usdt/%s\">%s</a></b>\n"+
				"Price Change: <b>+%s%%</b>",
			symbol,
			symbol,
			priceChangePct,
		)

		err := w.bot.notifier.Send(msg)
		if err != nil {
			w.bot.logger.Error().Err(err).Msg("failed to send telegram notification")
		}
	}
}
