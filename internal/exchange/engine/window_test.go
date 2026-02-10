package engine

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/lucrumx/bot/internal/exchange"
)

func TestWindow_AddTrade(t *testing.T) {
	w := NewWindow(100)
	ts := time.Now().Unix()
	price := 150.0

	w.AddTrade(exchange.Trade{
		Price: price,
		Ts:    ts * 1000,
	})

	assert.Equal(t, price, w.lastPrice)
	assert.Equal(t, ts, w.lastTs)

	idx := int(ts % w.windowSize)
	assert.Equal(t, price, w.prices[idx])
	assert.Equal(t, ts, w.timestamps[idx])
}

func TestWindow_GapFilling(t *testing.T) {
	w := NewWindow(100)
	ts := time.Now().Unix()

	// Первый трейд в T-10 секунд
	w.AddTrade(exchange.Trade{
		Price: 100,
		Ts:    (ts - 10) * 1000,
	})

	// Второй трейд через 5 секунд (в T-5)
	w.AddTrade(exchange.Trade{
		Price: 110.0,
		Ts:    (ts - 5) * 1000,
	})

	// Проверка, что секунды T-9, T-8, T-7, T-6 заполнились ценой 100 (последняя)
	for i := int64(1); i < 5; i++ {
		checkTs := ts - 10 + i
		idx := int(checkTs % w.windowSize)
		assert.Equal(t, 100.0, w.prices[idx], "Price at ts %d should be filled with 100", checkTs)
		assert.Equal(t, checkTs, w.timestamps[idx])
	}
}

func TestWindow_CheckGrow(t *testing.T) {
	w := NewWindow(1000)
	now := time.Now().Unix()

	// цена 100 ровно 900 секунд назад
	w.AddTrade(exchange.Trade{
		Price: 100.0,
		Ts:    (now - 900) * 1000,
	})

	// цена 120 сейчас (20 процентов рост)
	w.AddTrade(exchange.Trade{
		Price: 120.0,
		Ts:    now * 1000,
	})

	// если росто больше порога
	change, isGrow := w.CheckGrow(900, 15.0)
	assert.True(t, isGrow)
	assert.Equal(t, 20.0, change)

	// если рост меньше порога
	_, isGrow = w.CheckGrow(900, 25.0)
	assert.False(t, isGrow)
}

func TestWindow_AlertState(t *testing.T) {
	w := NewWindow(100)
	level := 15.5

	w.UpdateAlertState(level)
	alertTime, alertLevel := w.GetAlertState()

	assert.WithinDuration(t, time.Now(), alertTime, time.Second)
	assert.Equal(t, level, alertLevel)
}
