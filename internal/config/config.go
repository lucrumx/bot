// Package config contains configuration structs
package config

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/rs/zerolog"
	"gopkg.in/yaml.v3"

	"github.com/lucrumx/bot/internal/utils"
)

const defaultConfigFile = "config.yaml"

// Config represents the application configuration.
type Config struct {
	HTTP          HTTPConfig          `yaml:"http"`
	Database      DatabaseConfig      `yaml:"database"`
	Exchange      ExchangeConfig      `yaml:"exchange"`
	Notifications NotificationsConfig `yaml:"notifications"`
}

// Load loads the application configuration from a YAML file or environment variables.
func Load(logger zerolog.Logger) (*Config, error) {
	cfg := &Config{}

	configFilePath := flag.String("config", "", "path to config file")
	flag.Parse()

	if *configFilePath == "" {
		*configFilePath = defaultConfigFile
	}

	data, err := os.ReadFile(*configFilePath)
	if err != nil {
		logger.Info().Msgf("YAML config file %s not found, loading configs from env", *configFilePath)
		if err := loadFromEnv(cfg, logger); err != nil {
			return nil, err
		}
	} else {
		logger.Info().Msgf("YAML config found, loading from file %s", *configFilePath)

		if err := yaml.Unmarshal(data, cfg); err != nil {
			return nil, err
		}

		if err := validateConfig(cfg); err != nil {
			return nil, err
		}
	}

	return cfg, nil
}

func loadFromEnv(cfg *Config, logger zerolog.Logger) error {
	if err := godotenv.Load(); err != nil {
		logger.Info().Msg("no .env file found, hope environment variables are set")
	}

	// HTTP
	cfg.HTTP.Auth.JwtSecret = utils.GetEnv("JWT_SECRET", "")
	cfg.HTTP.Auth.JwtExpiresIn, _ = strconv.Atoi(utils.GetEnv("JWT_EXPIRES_IN", "24"))
	cfg.HTTP.HTTPServerPort = utils.GetEnv("HTTP_SERVER_PORT", ":8080")

	// Database
	cfg.Database = DatabaseConfig{
		Host:     utils.GetEnv("DB_HOST", ""),
		User:     utils.GetEnv("DB_USER", ""),
		Password: utils.GetEnv("DB_PASSWORD", ""),
		DbName:   utils.GetEnv("DB_NAME", ""),
		Port:     ":" + utils.GetEnv("DB_PORT", "5432"),
		SslMode:  utils.GetEnv("DB_SSL_MODE", "disable"),
	}

	// Exchange
	byBit := ByBitConfig{
		BaseURL:   utils.GetEnv("BYBIT_BASE_URL", ""),
		WsBaseURL: utils.GetEnv("BYBIT_WS_BASE_URL", ""),
		APIKey:    utils.GetEnv("BYBIT_API_KEY", ""),
		APISecret: utils.GetEnv("BYBIT_API_SECRET", ""),
		RecvWindow: func() int64 {
			rcw, err := strconv.Atoi(utils.GetEnv("BYBIT_API_SECRET", "5000"))
			if err != nil {
				return 5000
			}
			return int64(rcw)
		}(),
	}

	bingX := BingXConfig{
		WSUrl:     utils.GetEnv("BINGX_WS_URL", ""),
		APIKey:    utils.GetEnv("BINGX_API_KEY", ""),
		APISecret: utils.GetEnv("BINGX_API_SECRET", ""),
	}

	wsClientBufferSize, err := strconv.Atoi(utils.GetEnv("WS_CLIENT_BUFFER_SIZE", "5000"))
	if err != nil {
		return raiseErrorEnv("WS_CLIENT_BUFFER_SIZE")
	}

	rawTurnover := strings.ReplaceAll(utils.GetEnv("FILTER_TICKERS_TURNOVER", ""), "_", "")
	filterTickersByTurnover, err := strconv.ParseFloat(rawTurnover, 64)
	if err != nil {
		return raiseErrorEnv("FILTER_TICKERS_TURNOVER")
	}

	pumpInterval, err := strconv.Atoi(utils.GetEnv("PUMP_INTERVAL", ""))
	if err != nil {
		return raiseErrorEnv("PUMP_INTERVAL")
	}

	targetPriceChange, err := strconv.ParseFloat(utils.GetEnv("TARGET_PRICE_CHANGE", ""), 64)
	if err != nil {
		return raiseErrorEnv("TARGET_PRICE_CHANGE")
	}

	startupDelay, err := strconv.ParseFloat(utils.GetEnv("STARTUP_DELAY", ""), 64)
	if err != nil {
		return raiseErrorEnv("STARTUP_DELAY")
	}

	checkIntervalRaw, err := strconv.Atoi(utils.GetEnv("CHECK_INTERVAL", ""))
	if err != nil {
		return raiseErrorEnv("CHECK_INTERVAL")
	}

	alertStep, err := strconv.ParseFloat(utils.GetEnv("ALERT_STEP", ""), 64)
	if err != nil {
		return raiseErrorEnv("ALERT_STEP")
	}

	rpsTimerIntervalInSec, err := strconv.Atoi(utils.GetEnv("RPS_TIMER_INTERVAL", "60"))
	if err != nil {
		return raiseErrorEnv("RPS_TIMER_INTERVAL")
	}

	// ArbitrageBot
	ArbitrageBotMaxAgeMs, err := strconv.ParseInt(utils.GetEnv("ARBITRATION_BOT_MAX_AGE_MS", ""), 10, 64)
	if err != nil {
		return raiseErrorEnv("ARBITRATION_BOT_MAX_AGE_MS")
	}
	ArbitrageBotMinSpreadPercent, err := strconv.ParseFloat(utils.GetEnv("ARBITRATION_BOT_MIN_SPREAD_PERCENT", ""), 64)
	if err != nil {
		return raiseErrorEnv("ARBITRATION_BOT_MIN_SPREAD_PERCENT")
	}
	ArbitrageBotPercentForCloseSpread, err := strconv.ParseFloat(utils.GetEnv("ARBITRATION_BOT_PERCENT_FOR_CLOSE_SPREAD", ""), 64)
	if err != nil {
		return raiseErrorEnv("ARBITRATION_BOT_PERCENT_FOR_CLOSE_SPREAD")
	}

	botConfig := BotConfig{
		CheckInterval:         time.Duration(checkIntervalRaw) * time.Second,
		StartupDelay:          time.Duration(startupDelay) * time.Second,
		FilterTickersTurnover: filterTickersByTurnover,
		PumpInterval:          pumpInterval,
		TargetPriceChange:     targetPriceChange,
		AlertStep:             alertStep,
		RpsTimerInterval:      rpsTimerIntervalInSec,
	}

	arbConfig := ArbitrageBotConfig{
		MaxAgeMs:              ArbitrageBotMaxAgeMs,
		MinSpreadPercent:      ArbitrageBotMinSpreadPercent,
		PercentForCloseSpread: ArbitrageBotPercentForCloseSpread,
	}

	cfg.Exchange = ExchangeConfig{
		ByBit: byBit,
		BingX: bingX,
		WsClient: WsClientConfig{
			BufferSize: wsClientBufferSize,
		},
		Bot:          botConfig,
		ArbitrageBot: arbConfig,
	}

	cfg.Notifications = NotificationsConfig{
		Telegram: TelegramConfig{
			BotToken: utils.GetEnv("TELEGRAM_BOT_TOKEN", ""),
			ChatID:   utils.GetEnv("TELEGRAM_CHAT_ID", ""),
		},
	}

	return nil
}

func raiseErrorEnv(envName string) error {
	return fmt.Errorf("invalid env value %s", envName)
}

func raiseErrorYAML(envName string) error {
	return fmt.Errorf("invalid yaml value or empty: %s", envName)
}

func validateConfig(cfg *Config) error {
	if cfg.HTTP.Auth.JwtSecret == "" {
		return raiseErrorYAML("Http.Auth.JwtSecret")
	}

	if cfg.HTTP.Auth.JwtExpiresIn == 0 {
		return raiseErrorYAML("Http.Auth.JwtExpiresIn")
	}

	if cfg.HTTP.HTTPServerPort == "" {
		return raiseErrorYAML("Http.Auth.JwtExpiresIn")
	}

	// Database
	if cfg.Database.Host == "" {
		return raiseErrorYAML("Database.Host")
	}

	if cfg.Database.User == "" {
		return raiseErrorYAML("Database.User")
	}

	if cfg.Database.Password == "" {
		return raiseErrorYAML("Database.Password")
	}

	if cfg.Database.DbName == "" {
		return raiseErrorYAML("Database.DbName")
	}

	if cfg.Database.Port == "" {
		return raiseErrorYAML("Database.Port")
	}

	if cfg.Database.SslMode == "" {
		cfg.Database.SslMode = "false"
	}

	// Exchanges
	// ByBit
	if cfg.Exchange.ByBit.BaseURL == "" {
		return raiseErrorYAML("Exchange.ByBit.BaseUrl")
	}
	if cfg.Exchange.ByBit.WsBaseURL == "" {
		return raiseErrorYAML("Exchange.ByBit.WsBaseUrl")
	}
	if cfg.Exchange.ByBit.APIKey == "" {
		return raiseErrorYAML("Exchange.ByBit.APIKey")
	}
	if cfg.Exchange.ByBit.APISecret == "" {
		return raiseErrorYAML("Exchange.ByBit.APISecret")
	}
	if cfg.Exchange.ByBit.RecvWindow == 0 {
		cfg.Exchange.ByBit.RecvWindow = 5000
	}

	// BingX
	if cfg.Exchange.BingX.WSUrl == "" {
		return raiseErrorYAML("Exchange.BingX.WSUrl")
	}
	if cfg.Exchange.BingX.APIKey == "" {
		return raiseErrorYAML("Exchange.BingX.APIKey")
	}
	if cfg.Exchange.BingX.APISecret == "" {
		return raiseErrorYAML("Exchange.BingX.APISecret")
	}

	if cfg.Exchange.WsClient.BufferSize == 0 {
		return raiseErrorYAML("Exchange.WsClient.BufferSize")
	}

	if cfg.Exchange.Bot.CheckInterval.Seconds() == 0 {
		return raiseErrorYAML("Exchange.Bot.CheckInterval")
	}
	if cfg.Exchange.Bot.StartupDelay.Seconds() == 0 {
		return raiseErrorYAML("Exchange.Bot.StartupDelay")
	}
	// no calculation and equal check is ok
	if cfg.Exchange.Bot.FilterTickersTurnover == 0 {
		return raiseErrorYAML("Exchange.Bot.FilterTickersTurnover")
	}
	if cfg.Exchange.Bot.PumpInterval == 0 {
		return raiseErrorYAML("Exchange.Bot.PumpInterval")
	}
	if cfg.Exchange.Bot.TargetPriceChange == 0 {
		return raiseErrorYAML("Exchange.Bot.TargetPriceChange")
	}
	if cfg.Exchange.Bot.AlertStep == 0 {
		return raiseErrorYAML("Exchange.Bot.AlertStep")
	}
	if cfg.Exchange.Bot.RpsTimerInterval == 0 {
		return raiseErrorYAML("Exchange.Bot.RpsTimerInterval")
	}

	// ArbitrageBot
	if cfg.Exchange.ArbitrageBot.MaxAgeMs == 0 {
		return raiseErrorYAML("Exchange.ArbitrageBot.MaxAgeMs")
	}
	if cfg.Exchange.ArbitrageBot.MinSpreadPercent == 0 {
		return raiseErrorYAML("Exchange.ArbitrageBot.MinSpreadPercent")
	}
	if cfg.Exchange.ArbitrageBot.PercentForCloseSpread < 0 || cfg.Exchange.ArbitrageBot.PercentForCloseSpread > 0.5 {
		return raiseErrorYAML("Exchange.ArbitrageBot.PercentForCloseSpread")
	}

	if cfg.Notifications.Telegram.BotToken == "" {
		return raiseErrorYAML("Notifications.Telegram.BotToken")
	}
	if cfg.Notifications.Telegram.ChatID == "" {
		return raiseErrorYAML("Notifications.Telegram.ChatID")
	}

	return nil
}
