// Package engine contains the bot engine.
package engine

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/shopspring/decimal"

	"github.com/lucrumx/bot/internal/utils"

	"github.com/lucrumx/bot/internal/exchange"
)

// Константы порогов (Thresholds) ---
const (
	// MinTrades1s - минимальное кол-во сделок за 1 сек для сигнала
	MinTrades1s = 10
	// MinTrades3s - минимальное кол-во сделок за 3 сек
	MinTrades3s = 20

	// MinVolume1s - абсолютный минимум объема за 1 сек (USDT)
	MinVolume1s = 20_000
	// MinVolume3s - абсолютный минимум объема за 3 сек (USDT)
	MinVolume3s = 50_000

	// PriceDelta1s - минимальный рост цены за 1 сек (в процентах, 0.4 = 0.4%)
	PriceDelta1s = 0.4
	// PriceDelta3s - минимальный рост цены за 3 сек (в процентах, 1.0 = 1%)
	PriceDelta3s = 1.0

	// StartUpDelay - время накопления статистики перед началом сигналов
	StartUpDelay = 10 * time.Second
)

// Bot represents a bot engine.
type Bot struct {
	provider exchange.Provider

	mutex   sync.Mutex
	windows map[string]*Window

	kFactor      decimal.Decimal
	absMinVolume decimal.Decimal
	startTime    time.Time
}

// NewBot creates a new Bot (constructor).
func NewBot(provider exchange.Provider) *Bot {
	kFactor, err := decimal.NewFromString(utils.GetEnv("K_FACTOR", ""))
	if err != nil {
		log.Fatal().Err(err).Msg("failed to parse K_FACTOR evn")
	}

	absMinVolume, err := decimal.NewFromString(utils.GetEnv("ABS_MIN_VOLUME", ""))
	if err != nil {
		log.Fatal().Err(err).Msg("failed to parse ABS_MIN_VOLUME evn")
	}

	return &Bot{
		provider:     provider,
		mutex:        sync.Mutex{},
		windows:      map[string]*Window{},
		kFactor:      kFactor,
		absMinVolume: absMinVolume,
	}
}

// StartBot starts the bot engine and returns a channel of trades.
func (b *Bot) StartBot(ctx context.Context) (<-chan exchange.Trade, error) {
	b.startTime = time.Now()
	log.Print("bot engine: starting bot")

	log.Print("bot engine: getting tickers")
	tickers, err := b.provider.GetTickers(ctx, nil, exchange.CategoryLinear)
	if err != nil {
		return nil, fmt.Errorf("bot engine: failed to get tickers")
	}
	cntTickers := len(*tickers)
	if cntTickers == 0 {
		return nil, fmt.Errorf("bot engine: no tickers found")
	}

	filteredTickers := filterTickers(*tickers)

	sourceChan, err := b.provider.SubscribeTrades(ctx, filteredTickers)
	if err != nil {
		return nil, err
	}

	log.Printf("bot engine: starting trade processor and collection statistics for %d seconds", windowSize)

	outChan := make(chan exchange.Trade, 10000)

	go func() {
		defer close(outChan)
		for {
			select {
			case <-ctx.Done():
				return
			case trade, ok := <-sourceChan:
				if !ok {
					return
				}
				// Анализируем
				b.processTrade(trade)

				// Пробрасываем наружу (неблокирующе или с буфером)
				select {
				case outChan <- trade:
				default:
					// Если получатель (main) тормозит, мы не блокируем работу бота,
					// но данные в main могут пропадать. Это допустимо для логов.
				}
			}
		}
	}()

	return outChan, nil
}

func filterTickers(tickers []exchange.Ticker) []string {
	log.Printf("bot engine: got %d tickers", len(tickers))

	var filteredTickers []string
	for _, ticker := range tickers {
		if !strings.HasSuffix(ticker.Symbol, "USDT") {
			continue
		}
		// Фильтр по Turnover24h (оборот в деньгах), а не OpenInterest
		minTurnover := decimal.NewFromInt(800_000)    // $800k
		maxTurnover := decimal.NewFromInt(10_000_000) // $10m

		if ticker.Turnover24h.LessThan(minTurnover) || ticker.Turnover24h.GreaterThan(maxTurnover) {
			continue
		}

		filteredTickers = append(filteredTickers, ticker.Symbol)
	}

	log.Printf("bot engine: %d tickers left after filtering", len(filteredTickers))
	return filteredTickers
}

func (b *Bot) processTrade(trade exchange.Trade) {
	b.mutex.Lock()
	window, ok := b.windows[trade.Symbol]
	if !ok {
		window = NewWindow()
		b.windows[trade.Symbol] = window
	}
	b.mutex.Unlock()

	window.AddTrade(trade)
	b.checkPump(trade.Symbol, window)
}

/*
/*
checkPump — сердце детектора аномальной активности.

Принцип работы адаптивных порогов:
Не используем жесткие цифры для всех монет, потому что $50,000 объема для BTC — это шум,
а для мелкого альткоина — начало пампа.

Принцип работы:
 1. База: Считаем средний объем и количество сделок в секунду за последние windowSize (см window.go), сейчас это
    300 сек (5 мин). Это фон или нормальное состояние конкретного тикера.
 2. Адаптивность: Умножаем фон на коэффициент K (kFactor, например 8) (из env).
    Так мы получаем порог, который в 8 раз выше обычного состояния этой монеты.
 3. Фильтрация шума: Используем абсолютные минимумы (MinVolume, MinTrades).
    Это нужно, чтобы не реагировать на случайные сделки в $10 на совсем «мертвых» парах,
    где даже одна покупка может превысить среднее значение в 100 раз.

Сигнал срабатывает, если за 1 или 3 секунды одновременно:
- Объем превысил Адаптивный Порог И Абсолютный Минимум.
- Количество сделок превысило Адаптивный Порог И Абсолютный Минимум.
- Цена выросла более чем на заданный процент.
*/
func (b *Bot) checkPump(symbol string, win *Window) {
	// Не даем сигналы первые N секунд, чтобы накопилась статистика
	if time.Since(b.startTime) < StartUpDelay {
		return
	}

	// 0. Базовые показатели за весь период окна (windowSize из window.go)
	// В Go, если константа в том же пакете, регистр должен совпадать.
	// Если в window.go она lowercase (windowSize), то и тут должна быть такой же.
	statsBase := win.GetStatistics(windowSize)

	// Средние показатели в секунду (Норма)
	avgVolPerSec := statsBase.totalVolumeUSDT.Div(decimal.NewFromInt(windowSize))
	avgTradesPerSec := decimal.NewFromInt(statsBase.tradeCount).Div(decimal.NewFromInt(windowSize))

	k := b.kFactor

	// 1: 1 секундный памп
	// ловит мгновенные "палки" вверх
	stats1s := win.GetStatistics(1)

	// Порог объема: берем максимум между жестким минимумом и (среднее * K)
	threshVol1s := decimal.Max(decimal.NewFromInt(MinVolume1s), avgVolPerSec.Mul(k))

	// Порог сделок: берем максимум между жестким минимумом и (среднее * K)
	threshTrades1s := decimal.Max(decimal.NewFromInt(MinTrades1s), avgTradesPerSec.Mul(k))

	// Порог цены: фиксированный
	threshPrice1s := decimal.NewFromFloat(PriceDelta1s)

	if stats1s.totalVolumeUSDT.GreaterThan(threshVol1s) &&
		decimal.NewFromInt(stats1s.tradeCount).GreaterThan(threshTrades1s) &&
		stats1s.priceChangePcnt.GreaterThan(threshPrice1s) {

		log.Warn().
			Str("pair", symbol).
			Str("type", "FLASH_PUMP_1S").
			Str("price_change", stats1s.priceChangePcnt.StringFixed(2)+"%").
			Str("volume", stats1s.totalVolumeUSDT.StringFixed(0)).
			Str("thresh_vol", threshVol1s.StringFixed(0)).
			Int64("trades", stats1s.tradeCount).
			Msg("🚀 PUMP DETECTED")
		return // Если сработала 1с, 3с уже не проверяем
	}

	// 2: 3 секундный памп
	// Движения мощнее, но более растянутые во времени
	stats3s := win.GetStatistics(3)

	// Порог объема: max(AbsMin3s, Среднее_за_1с * 3 секунды * K)
	threshVol3s := decimal.Max(
		decimal.NewFromInt(MinVolume3s),
		avgVolPerSec.Mul(decimal.NewFromInt(3)).Mul(k),
	)

	// Адаптивный порог сделок: max(AbsMin3s, Среднее_за_1с * 3 секунды * K)
	threshTrades3s := decimal.Max(
		decimal.NewFromInt(MinTrades3s),
		avgTradesPerSec.Mul(decimal.NewFromInt(3)).Mul(k),
	)

	// Порог цены: фиксированный (например, 1.0%)
	threshPrice3s := decimal.NewFromFloat(PriceDelta3s)

	if stats3s.totalVolumeUSDT.GreaterThan(threshVol3s) &&
		decimal.NewFromInt(stats3s.tradeCount).GreaterThan(threshTrades3s) &&
		stats3s.priceChangePcnt.GreaterThan(threshPrice3s) {

		log.Warn().
			Str("pair", symbol).
			Str("type", "MOMENTUM_PUMP_3S").
			Str("price_change", stats3s.priceChangePcnt.StringFixed(2)+"%").
			Str("volume", stats3s.totalVolumeUSDT.StringFixed(0)).
			Str("thresh_vol", threshVol3s.StringFixed(0)).
			Int64("trades", stats3s.tradeCount).
			Msg("🚀 PUMP DETECTED")
	}
}
