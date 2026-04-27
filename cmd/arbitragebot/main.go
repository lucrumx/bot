// Package main contains the main entry point for the arbitrage bot application.
package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/lucrumx/bot/internal/storage"

	"github.com/lucrumx/bot/internal/config"
	"github.com/lucrumx/bot/internal/exchange"
	"github.com/lucrumx/bot/internal/exchange/client/bingx"
	"github.com/lucrumx/bot/internal/exchange/client/bybit"
	"github.com/lucrumx/bot/internal/exchange/client/mexc"
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

	byBitClient := bybit.NewByBitClient(cfg, logger)
	bingXClient := bingx.NewClient(cfg, logger)
	mexcClient := mexc.NewClient(cfg, logger)

	clients := []exchange.Provider{
		byBitClient,
		bingXClient,
		mexcClient,
	}

	db := storage.InitDB(cfg)
	notif := notifier.NewTelegramNotifier(cfg)
	arbitrageSpreadRepo := arbitragebot.NewArbitrageSpreadRepository(db)
	orderRepo := arbitragebot.NewOrderRepository(db)

	bot := arbitragebot.NewBot(clients, logger, cfg, notif, arbitrageSpreadRepo, orderRepo)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err = bot.Run(ctx); err != nil {
		logger.Fatal().Err(err).Msg("Failed to run bot")
	}

}


/*
Фандинг
# ByBit — все символы, поле fundingRate
curl -s "https://api.bybit.com/v5/market/tickers?category=linear" | python3 -m json.tool | head -30

# BingX — все символы, поле lastFundingRate
curl -s "https://open-api.bingx.com/openApi/swap/v2/quote/premiumIndex" | python3 -m json.tool | head -30

# MEXC — все символы, поле fundingRate
curl -s "https://contract.mexc.com/api/v1/contract/funding_rate" | python3 -m json.tool | head -30
*/