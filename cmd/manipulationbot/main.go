// Package main is the main entry point for the manipulationbot.
package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/lucrumx/bot/internal/config"
	"github.com/lucrumx/bot/internal/exchange/client/bybit"
	"github.com/lucrumx/bot/internal/exchange/manipulationbot"
	"github.com/lucrumx/bot/internal/notifier"
)

func main() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	logger := log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	cfg, err := config.Load(logger)
	if err != nil {
		logger.Fatal().Err(err).Msg("Error loading config")
	}

	provider := bybit.NewByBitClient(cfg)
	notif := notifier.NewTelegramNotifier(cfg)
	botCfg := loadBotConfig(cfg)
	bot := manipulationbot.NewBot(provider, notif, botCfg, logger)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err = bot.Run(ctx); err != nil {
		logger.Fatal().Err(err).Msg("Failed to run manipulation bot")
	}
}

func loadBotConfig(cfg *config.Config) manipulationbot.Config {
	defaults := manipulationbot.DefaultConfig()
	raw := cfg.Exchange.ManipulationBot

	if len(raw.Symbols) > 0 {
		defaults.Symbols = raw.Symbols
	}
	if raw.WindowSize > 0 {
		defaults.WindowSize = raw.WindowSize
	}
	if raw.CheckInterval > 0 {
		defaults.CheckInterval = raw.CheckInterval
	}
	if raw.StartupDelay > 0 {
		defaults.StartupDelay = raw.StartupDelay
	}
	if raw.AlertCooldown > 0 {
		defaults.AlertCooldown = raw.AlertCooldown
	}
	if raw.MinSpotATRPct > 0 {
		defaults.MinSpotATRPct = raw.MinSpotATRPct
	}
	if raw.MinATRRatio > 0 {
		defaults.MinATRRatio = raw.MinATRRatio
	}
	if raw.MinPerpTurnover24h > 0 {
		defaults.MinPerpTurnover24h = raw.MinPerpTurnover24h
	}
	if raw.MaxSpotTurnover24h > 0 {
		defaults.MaxSpotTurnover24h = raw.MaxSpotTurnover24h
	}
	if raw.RPSTimerInterval > 0 {
		defaults.RPSTimerInterval = raw.RPSTimerInterval
	}

	return defaults
}
