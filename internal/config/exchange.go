package config

import (
	"time"
)

// WsClientConfig contains configuration for websocket client.
type WsClientConfig struct {
	BufferSize int `yaml:"buffer_size"`
}

// ByBitConfig contains configuration for ByBit exchange.
type ByBitConfig struct {
	BaseURL   string `yaml:"base_url"`
	WsBaseURL string `yaml:"ws_base_url"`
	APIKey    string `yaml:"api_key"`
	APISecret string `yaml:"api_secret"`
}

// BingXConfig contains configuration for BingX exchange.
type BingXConfig struct {
	WSUrl     string `yaml:"ws_url"`
	APIKey    string `yaml:"api_key"`
	APISecret string `yaml:"api_secret"`
}

// BotConfig contains configuration for the bot.
type BotConfig struct {
	CheckInterval         time.Duration `yaml:"check_interval"`
	StartupDelay          time.Duration `yaml:"startup_delay"`
	FilterTickersTurnover float64       `yaml:"filter_tickers_turnover"`
	PumpInterval          int           `yaml:"pump_interval"`
	TargetPriceChange     float64       `yaml:"target_price_change"`
	AlertStep             float64       `yaml:"alert_step"`
	RpsTimerInterval      int           `yaml:"rps_timer_interval"`
}

// ExchangeConfig contains a configuration for an exchange.
type ExchangeConfig struct {
	ByBit    ByBitConfig    `yaml:"bybit"`
	BingX    BingXConfig    `yaml:"bingx"`
	WsClient WsClientConfig `yaml:"ws_client"`
	Bot      BotConfig
}
