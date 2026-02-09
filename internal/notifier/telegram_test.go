package notifier

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/lucrumx/bot/internal/config"
)

func setConfig() *config.Config {
	token := "some-token"
	chatID := "some-chat-id"

	cfg := config.Config{
		Notifications: config.NotificationsConfig{
			Telegram: config.TelegramConfig{
				BotToken: token,
				ChatID:   chatID,
			},
		},
	}

	return &cfg
}

func TestTelegramNotifier_Send_Mock(t *testing.T) {
	cfg := setConfig()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)

		err := r.ParseForm()
		assert.NoError(t, err)
		assert.Equal(t, cfg.Notifications.Telegram.ChatID, r.FormValue("chat_id"))
		assert.Equal(t, "test message", r.FormValue("text"))
		assert.Equal(t, "HTML", r.FormValue("parse_mode"))

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok": true}`))
	}))
	defer server.Close()

	notifier := NewTelegramNotifier(cfg)
	notifier.setTestOptions(server.URL, server.Client())

	err := notifier.Send("test message")

	assert.NoError(t, err)
}

func TestTelegramNotifier_Send_Integration(t *testing.T) {
	t.Skip("Skipping integration test")

	cfg := setConfig()

	n := NewTelegramNotifier(cfg)

	symbol := "BTCUSDT"

	msg := fmt.Sprintf(
		"<b>🚀 PUMP DETECTED: <a href=\"https://www.bybit.com/trade/usdt/%s\">%s</a></b>\n"+
			"Price Change: <b>+%s%%</b>",
		symbol,
		symbol,
		"45",
	)

	err := n.Send(msg)
	assert.NoError(t, err)
}
