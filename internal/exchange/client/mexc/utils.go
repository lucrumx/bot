package mexc

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// mexcAliases maps MEXC-specific normalized names to canonical names used by other exchanges.
var mexcAliases = map[string]string{
	"TONCOINUSDT": "TONUSDT",
}

// mexcAliasesReverse is the reverse of mexcAliases: canonical name → MEXC API name.
var mexcAliasesReverse = map[string]string{
	"TONUSDT": "TONCOIN_USDT",
}

func normalizeTickerName(symbol string) string {
	if strings.HasSuffix(symbol, "_USDT") {
		symbol = strings.TrimSuffix(symbol, "_USDT") + "USDT"
	}

	if canonical, ok := mexcAliases[symbol]; ok {
		return canonical
	}

	return symbol
}

// setSignedHeaders sets MEXC private API authentication headers on the request.
// For GET requests with no params, pass parameterString = "".
// For POST requests, pass the raw JSON body string as parameterString.
func setSignedHeaders(req *http.Request, apiKey, apiSecret, parameterString string) {
	timestamp := strconv.FormatInt(time.Now().UnixMilli(), 10)

	toSign := apiKey + timestamp + parameterString
	h := hmac.New(sha256.New, []byte(apiSecret))
	h.Write([]byte(toSign))
	signature := hex.EncodeToString(h.Sum(nil))

	req.Header.Set("ApiKey", apiKey)
	req.Header.Set("Request-Time", timestamp)
	req.Header.Set("Signature", signature)
	req.Header.Set("Content-Type", "application/json")
}

func denormalizeTickerName(symbol string) string {
	if apiName, ok := mexcAliasesReverse[symbol]; ok {
		return apiName
	}

	if strings.HasSuffix(symbol, "USDT") && !strings.Contains(symbol, "_") {
		symbol = strings.TrimSuffix(symbol, "USDT") + "_USDT"
	}

	return symbol
}
