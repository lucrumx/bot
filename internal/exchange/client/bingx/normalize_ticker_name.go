package bingx

import "strings"

func normalizeTickerName(symbol string) string {
	if strings.Contains(symbol, "-USDT") {
		symbol = strings.TrimSuffix(symbol, "-USDT") + "USDT"
	}

	return symbol
}

func denormalizeTickerName(symbol string) string {
	if !strings.Contains(symbol, "-USDT") {
		symbol = strings.TrimSuffix(symbol, "USDT") + "-USDT"
	}

	return symbol
}
