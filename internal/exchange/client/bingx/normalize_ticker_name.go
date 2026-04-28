package bingx

import "strings"

// bingxAliases maps BingX displayName (normalized) to canonical name used internally.
var bingxAliases = map[string]string{
	"TONCOINUSDT": "TONUSDT",
}

// bingxAliasesReverse maps canonical name to BingX API symbol (not displayName).
var bingxAliasesReverse = map[string]string{
	"TONUSDT": "TONCOIN-USDT",
}

func normalizeTickerName(symbol string) string {
	if strings.Contains(symbol, "-USDT") {
		symbol = strings.TrimSuffix(symbol, "-USDT") + "USDT"
	}

	if canonical, ok := bingxAliases[symbol]; ok {
		return canonical
	}

	return symbol
}

func denormalizeTickerName(symbol string) string {
	if apiName, ok := bingxAliasesReverse[symbol]; ok {
		return apiName
	}

	if !strings.HasSuffix(symbol, "-USDT") {
		symbol = strings.TrimSuffix(symbol, "USDT") + "-USDT"
	}

	return symbol
}
