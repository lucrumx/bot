package manipulationbot

import "time"

// Config controls the detector sensitivity.
type Config struct {
	// Если список задан, бот мониторит только эти символы.
	// Если пустой, символы выбираются автоматически по фильтрам ликвидности.
	Symbols []string

	// Размер окна, на котором считается ATR.
	WindowSize time.Duration

	// Минимальный интервал между повторными проверками одного символа.
	CheckInterval time.Duration

	// Задержка после старта бота, чтобы успела накопиться история по окну.
	StartupDelay time.Duration

	// Минимальная пауза между двумя алертами по одному символу.
	AlertCooldown time.Duration

	// Минимальный нормализованный ATR спота в процентах.
	MinSpotATRPct float64

	// Минимальное отношение ATR% spot/perp.
	MinATRRatio float64

	// Минимальный 24h оборот perp-рынка для автоподбора символов.
	MinPerpTurnover24h float64

	// Максимальный 24h оборот spot-рынка для автоподбора символов.
	// Идея в том, чтобы исключить слишком ликвидные пары, где такой паттерн менее показателен.
	MaxSpotTurnover24h float64

	// Интервал логирования throughput бота.
	RPSTimerInterval time.Duration
}

// DefaultConfig returns a conservative baseline tuned for short-term pump detection.
func DefaultConfig() Config {
	return Config{
		WindowSize:         90 * time.Second,
		CheckInterval:      5 * time.Second,
		StartupDelay:       2 * time.Minute,
		AlertCooldown:      10 * time.Minute,
		MinSpotATRPct:      0.20,
		MinATRRatio:        1.80,
		MinPerpTurnover24h: 250_000,
		MaxSpotTurnover24h: 5_000_000,
		RPSTimerInterval:   30 * time.Second,
	}
}
