package arbitragebot

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/shopspring/decimal"

	"github.com/lucrumx/bot/internal/models"
	"github.com/lucrumx/bot/internal/utils"
)

// handleOpen is dispatched for ArbitrageSpreadOpened events: logs synchronously, then opens the
// position. Telegram + DB writes are deferred to goroutines so they don't block the main loop,
// but the log line for "SPREAD DETECTED" is emitted synchronously so it always appears before
// the execution logs that follow.
func (e *Engine) handleOpen(ctx context.Context, event *SpreadEvent) {
	if e.pm.IsBlacklisted(event.Symbol) {
		return
	}

	e.logSpreadDetected(event)
	go e.sendOpenTelegram(event)

	if e.isSilentMode() {
		go e.saveSpread(ctx, event, nil)
		return
	}

	pos := e.openPosition(ctx, event)
	go e.saveSpread(ctx, event, pos)
}

// handleUpdate is dispatched for ArbitrageSpreadUpdated events: logs synchronously, then deferred
// Telegram + DB update.
func (e *Engine) handleUpdate(ctx context.Context, event *SpreadEvent) {
	spreadStr := strconv.FormatFloat(event.MaxSpreadPercent, 'f', 2, 64)
	e.logger.Warn().
		Str("pair", event.Symbol).
		Str("spread", spreadStr).
		Str("buy on", event.BuyOnExchange).
		Str("sell on", event.SellOnExchange).
		Msg("🔥 SPREAD GROWING")

	go func() {
		msg := fmt.Sprintf("<b>🔔 ARBITRAGE: Ticker - %s</b>\n\nSpread is growing, new spread: <code>%s%%</code>",
			event.Symbol, spreadStr)
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
			e.logger.Warn().Err(err).Msgf("failed to update spread symbol=%s", event.Symbol)
		}
	}()
}

// handleClose is dispatched for ArbitrageSpreadClosed events: logs synchronously, requests
// position close, then deferred Telegram + DB update.
func (e *Engine) handleClose(ctx context.Context, event *SpreadEvent) {
	e.logger.Warn().Str("pair", event.Symbol).Msg("🔥 SPREAD CLOSED")

	go func() {
		msg := fmt.Sprintf("<b>🔔 ARBITRAGE: Ticker - %s</b>\n\nSpread closed", event.Symbol)
		if err := e.notif.Send(msg); err != nil {
			e.logger.Warn().Err(err).Msg("failed to send telegram notification")
		}
	}()

	if !e.isSilentMode() {
		pos := e.pm.FindByKey(event.Symbol, event.BuyOnExchange, event.SellOnExchange)
		if pos != nil {
			transition := pos.RequestClose()
			e.applyTransition(ctx, pos, transition)
		}
	}

	go func() {
		now := time.Now()
		err := e.spreadRepo.Update(ctx, &models.ArbitrageSpread{
			Status:    models.ArbitrageSpreadClosed,
			ClosedAt:  &now,
			UpdatedAt: now,
		}, FindFilter{
			Symbol: event.Symbol,
			BuyEx:  event.BuyOnExchange,
			SellEx: event.SellOnExchange,
			Status: []models.ArbitrageSpreadStatus{models.ArbitrageSpreadOpened, models.ArbitrageSpreadUpdated},
		})
		if err != nil {
			e.logger.Warn().Err(err).Msgf("failed to update spread symbol=%s", event.Symbol)
		}
	}()
}

// logSpreadDetected emits the synchronous log line for a newly detected spread.
func (e *Engine) logSpreadDetected(event *SpreadEvent) {
	spreadStr := strconv.FormatFloat(event.FromSpreadPercent, 'f', 2, 64)
	e.logger.Warn().
		Str("pair", event.Symbol).
		Str("spread", spreadStr).
		Str("buy on", event.BuyOnExchange).
		Str("sell on", event.SellOnExchange).
		Msg("🔥 SPREAD DETECTED")
}

// sendOpenTelegram pushes the Telegram notification for a newly detected spread. Runs in a
// goroutine so the slow HTTP call doesn't delay execution; logs are emitted by logSpreadDetected.
func (e *Engine) sendOpenTelegram(event *SpreadEvent) {
	spreadStr := strconv.FormatFloat(event.FromSpreadPercent, 'f', 2, 64)
	msg := fmt.Sprintf(
		"<b>🔔 ARBITRAGE: Ticker - %s</b>\n\n"+
			"Spread: <code>%s%%</code>\n\n"+
			"🟢 Buy:  %s - <b>%s</b>\n"+
			"🔴 Sell: %s - <b>%s</b>",
		event.Symbol, spreadStr,
		event.BuyOnExchange, utils.FormatPrice(event.BuyPrice),
		event.SellOnExchange, utils.FormatPrice(event.SellPrice),
	)
	if err := e.notif.Send(msg); err != nil {
		e.logger.Warn().Err(err).Msg("failed to send telegram notification")
	}
}
