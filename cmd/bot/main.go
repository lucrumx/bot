// Package main implements the entry point for the Bot.
package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/lucrumx/bot/internal/config"

	"github.com/lucrumx/bot/internal/notifier"

	"github.com/lucrumx/bot/internal/exchange/client/bybit"
	"github.com/lucrumx/bot/internal/exchange/engine"
)

func main() {
	log.Logger = zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr}).
		With().
		Timestamp().
		Logger()

	cfg, err := config.Load()
	if err != nil {
		log.Fatal().Err(err).Msg("Error loading config")
	}

	notif := notifier.NewTelegramNotifier(cfg)

	client := bybit.NewByBitClient(cfg)
	bot := engine.NewBot(client, notif, cfg)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	inChanTrades, err := bot.StartBot(ctx)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to start bot")
	}

	log.Info().Msg("Bot started. Waiting for pumps...")

	for trade := range inChanTrades {
		_ = trade
		/* log.Debug().Msg(fmt.Sprintf("Received trade: %-15s | ts: %-13s | price: %8s | volume: %10s | in usdt: %10s | side: %-4s",
			trade.Symbol,
			strconv.FormatInt(trade.Ts, 10),
			trade.Price.StringFixed(4),
			trade.Volume.StringFixed(4),
			trade.USDTAmount.StringFixed(4),
			trade.Side,
		))*/
	}

	log.Info().Msg("Bot stopped")
}
