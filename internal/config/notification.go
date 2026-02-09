package config

// TelegramConfig represents the Telegram configuration.
type TelegramConfig struct {
	BotToken string `yaml:"bot_token"`
	ChatID   string `yaml:"chat_id"`
}

// NotificationsConfig represents the Notifications configuration.
type NotificationsConfig struct {
	Telegram TelegramConfig `yaml:"telegram"`
}
