package bingx

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/url"
	"sort"
	"strings"
)

func getSortedQuery(query map[string]string, timestamp int64, urlEncode bool) string {
	queryKeys := make([]string, 0, len(query))

	for k := range query {
		queryKeys = append(queryKeys, k)
	}

	sort.Strings(queryKeys)

	var queryStr string
	for _, paramName := range queryKeys {
		if queryStr != "" {
			queryStr += "&"
		}
		value := query[paramName]
		if urlEncode {
			value = url.QueryEscape(value)
			value = strings.ReplaceAll(value, "+", "%20")
		}

		queryStr += fmt.Sprintf("%s=%s", paramName, value)
	}
	queryStr += fmt.Sprintf("&timestamp=%d", timestamp)

	return queryStr
}

func (c *Client) computeHmac256(sortedQueryStr string) string {
	key := []byte(c.cfg.Exchange.BingX.APISecret)
	h := hmac.New(sha256.New, key)
	h.Write([]byte(sortedQueryStr))
	return hex.EncodeToString(h.Sum(nil))
}
