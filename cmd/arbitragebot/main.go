// Package main contains the main entry point for the arbitrage bot application.
package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/lucrumx/bot/internal/config"
	"github.com/lucrumx/bot/internal/exchange"
	"github.com/lucrumx/bot/internal/exchange/client/bingx"
	"github.com/lucrumx/bot/internal/exchange/client/bybit"
	"github.com/lucrumx/bot/internal/notifier"

	"github.com/lucrumx/bot/internal/exchange/arbitragebot"
)

func main() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	logger := log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	cfg, err := config.Load(logger)
	if err != nil {
		logger.Fatal().Err(err).Msg("Error loading config")
	}

	byBitClient := bybit.NewByBitClient(cfg)
	bingXClient := bingx.NewClient(cfg)

	clients := []exchange.Provider{
		byBitClient,
		bingXClient,
	}

	notif := notifier.NewTelegramNotifier(cfg)

	bot := arbitragebot.NewBot(clients, notif, logger, cfg)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err = bot.Run(ctx); err != nil {
		logger.Fatal().Err(err).Msg("Failed to run bot")
	}

}
