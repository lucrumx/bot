// Package main implements the entry point for the Bot.
package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/joho/godotenv"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	notifier2 "github.com/lucrumx/bot/internal/notifier"

	"github.com/lucrumx/bot/internal/exchange/client/bybit"
	"github.com/lucrumx/bot/internal/exchange/engine"
)

func main() {
	// 1. Настройка красивого логирования в консоль (вместо JSON)
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	if err := godotenv.Load(); err != nil {
		log.Warn().Msg("No .env file found, hope environment variables are set")
	}

	notifier := notifier2.NewTelegramNotifier()

	client := bybit.NewByBitClient()
	bot := engine.NewBot(client, notifier)

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
