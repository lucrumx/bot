package notifier

import (
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/lucrumx/bot/internal/config"
)

const baseURL = "https://api.telegram.org/bot%s/sendMessage"

// TelegramNotifier is used to send notifications to a Telegram chat using the Telegram Bot API.
type TelegramNotifier struct {
	token      string
	chatID     string
	url        string
	httpClient *http.Client
	cfg        *config.Config
}

// NewTelegramNotifier constructor.
func NewTelegramNotifier(cfg *config.Config) *TelegramNotifier {
	token := cfg.Notifications.Telegram.BotToken

	sendURL := fmt.Sprintf(baseURL, token)

	return &TelegramNotifier{
		token:      token,
		chatID:     cfg.Notifications.Telegram.ChatID,
		url:        sendURL,
		httpClient: &http.Client{Timeout: 5 * time.Second},
		cfg:        cfg,
	}
}

func (t *TelegramNotifier) setTestOptions(url string, client *http.Client) {
	t.url = url
	t.httpClient = client
}

// Send sends a notification to the Telegram chat.
func (t *TelegramNotifier) Send(message string) error {
	params := url.Values{}
	params.Add("chat_id", t.chatID)
	params.Add("text", message)
	params.Add("parse_mode", "HTML")

	resp, err := t.httpClient.PostForm(t.url, params)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := resp.Body.Close(); err != nil {
			fmt.Printf("failed to close response body: %v\n", closeErr)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("telegram api unexpected status code: %d", resp.StatusCode)
	}

	return nil
}
