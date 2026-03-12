package arbitragebot

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"time"

	"github.com/rs/zerolog"
	"github.com/shopspring/decimal"

	"github.com/lucrumx/bot/internal/models"

	"github.com/lucrumx/bot/internal/notifier"
)

type signalHandler struct {
	notif    notifier.Notifier
	logger   zerolog.Logger
	repo     ArbitrageSpreadRepository
	signalCh chan *SpreadEvent
}

func newSignalHandler(notif notifier.Notifier, logger zerolog.Logger, repo ArbitrageSpreadRepository) *signalHandler {
	return &signalHandler{
		notif:    notif,
		logger:   logger,
		repo:     repo,
		signalCh: make(chan *SpreadEvent, 1000),
	}
}

func (s *signalHandler) run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case event := <-s.signalCh:
			switch event.Status {
			case models.ArbitrageSpreadOpened:
				s.handleNewSpreadEvent(ctx, event)
			case models.ArbitrageSpreadUpdated:
				s.handleNewSpreadUpdate(ctx, event)
			case models.ArbitrageSpreadClosed:
				s.handleNewSpreadClose(ctx, event)
			}
		}
	}
}

func (s *signalHandler) handleSignal(se []*SpreadEvent) {
	for _, e := range se {
		s.signalCh <- e
	}
}

func (s *signalHandler) handleNewSpreadEvent(ctx context.Context, e *SpreadEvent) {
	spreadStr := strconv.FormatFloat(e.FromSpreadPercent, 'f', 2, 64)

	s.logger.Warn().
		Str("pair", e.Symbol).
		Str("spread", spreadStr).
		Str("buy on", e.BuyOnExchange).
		Str("sell on", e.SellOnExchange).
		Msg("🔥 SPREAD DETECTED")

	msg := fmt.Sprintf(
		"<b>🔔 ARBITRAGE: Ticker - %s</b>\n\n"+
			"Spread: <code>%s%%</code>\n\n"+
			"🟢 Buy:  %s - <b>%s</b>\n"+
			"🔴 Sell: %s - <b>%s</b>",
		e.Symbol,
		spreadStr,
		e.BuyOnExchange, formatPrice(e.BuyPrice),
		e.SellOnExchange, formatPrice(e.SellPrice),
	)

	if err := s.notif.Send(msg); err != nil {
		s.logger.Warn().Err(err).Msg("failed to send telegram notification")
	}

	err := s.repo.Create(ctx, &models.ArbitrageSpread{
		Symbol:           e.Symbol,
		BuyOnExchange:    e.BuyOnExchange,
		SellOnExchange:   e.SellOnExchange,
		BuyPrice:         decimal.NewFromFloat(e.BuyPrice),
		SellPrice:        decimal.NewFromFloat(e.SellPrice),
		SpreadPercent:    decimal.NewFromFloat(e.FromSpreadPercent),
		MaxSpreadPercent: decimal.NewFromFloat(e.MaxSpreadPercent),
		Status:           e.Status,
	})
	if err != nil {
		s.logger.Warn().Err(err).Msgf("failed to create arbitrage spread, symbol: %s, buy on: %s, sell on: %s",
			e.Symbol, e.BuyOnExchange, e.SellOnExchange)
	}

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

func (s *signalHandler) handleNewSpreadUpdate(ctx context.Context, e *SpreadEvent) {
	spreadStr := strconv.FormatFloat(e.MaxSpreadPercent, 'f', 2, 64)

	s.logger.Warn().
		Str("pair", e.Symbol).
		Str("spread", spreadStr).
		Str("buy on", e.BuyOnExchange).
		Str("sell on", e.SellOnExchange).
		Msg("🔥 SPREAD GROWING")

	msg := fmt.Sprintf(
		"<b>🔔 ARBITRAGE: Ticker - %s</b>\n\n"+
			"Spread is growing, new spread: <code>%s%%</code>",
		e.Symbol,
		spreadStr,
	)

	if err := s.notif.Send(msg); err != nil {
		s.logger.Warn().Err(err).Msg("failed to send telegram notification")
	}

	err := s.repo.Update(ctx, &models.ArbitrageSpread{
		Status:           models.ArbitrageSpreadUpdated,
		MaxSpreadPercent: decimal.NewFromFloat(e.MaxSpreadPercent),
		UpdatedAt:        time.Now(),
	}, FindFilter{
		Symbol: e.Symbol,
		BuyEx:  e.BuyOnExchange,
		SellEx: e.SellOnExchange,
		Status: []models.ArbitrageSpreadStatus{models.ArbitrageSpreadOpened, models.ArbitrageSpreadUpdated},
	})
	if err != nil {
		s.logger.Warn().Err(err).Msgf("failed to update arbitrage spread, symbol: %s, buy exchange: %s, sell exchange: %s",
			e.Symbol, e.BuyOnExchange, e.SellOnExchange)
	}

}

func (s *signalHandler) handleNewSpreadClose(ctx context.Context, e *SpreadEvent) {
	s.logger.Warn().
		Str("pair", e.Symbol).
		Msg("🔥 SPREAD CLOSED")

	msg := fmt.Sprintf(
		"<b>🔔 ARBITRAGE: Ticker - %s</b>\n\n"+
			"Spread closed",
		e.Symbol,
	)

	if err := s.notif.Send(msg); err != nil {
		s.logger.Warn().Err(err).Msg("failed to send telegram notification")
	}

	err := s.repo.Update(ctx, &models.ArbitrageSpread{
		Status:    models.ArbitrageSpreadClosed,
		UpdatedAt: time.Now(),
	}, FindFilter{
		Symbol: e.Symbol,
		BuyEx:  e.BuyOnExchange,
		SellEx: e.SellOnExchange,
		Status: []models.ArbitrageSpreadStatus{models.ArbitrageSpreadOpened, models.ArbitrageSpreadUpdated},
	})
	if err != nil {
		s.logger.Warn().Err(err).Msgf("failed to update arbitrage spread, symbol: %s, buy exchange: %s, sell exchange: %s",
			e.Symbol, e.BuyOnExchange, e.SellOnExchange)
	}
}
