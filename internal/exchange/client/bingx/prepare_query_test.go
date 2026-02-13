package bingx

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestGetSortedQuery(t *testing.T) {
	query := map[string]string{
		"x":      "xxx",
		"a":      "aaa",
		"symbol": "BTC-USDT",
		"h":      "hello world",
	}

	timestamp := time.Now().UnixMilli()

	sortedQueryStr := getSortedQuery(query, timestamp, false)
	expected := fmt.Sprintf("a=aaa&h=hello world&symbol=BTC-USDT&x=xxx&timestamp=%d", timestamp)
	assert.Equal(t, expected, sortedQueryStr)

	sortedQueryStr = getSortedQuery(query, timestamp, true)
	expected = fmt.Sprintf("a=aaa&h=hello%sworld&symbol=BTC-USDT&x=xxx&timestamp=%d", "%20", timestamp)
	assert.Equal(t, expected, sortedQueryStr)
}
