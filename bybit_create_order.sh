#!/bin/bash

# load only BYBIT_ vars from .env
set -a
eval "$(grep '^BYBIT_' .env)"
set +a

RECV_WINDOW="5000"
TIMESTAMP=$(python3 -c "import time; print(int(time.time() * 1000))")

BODY='{"category":"linear","symbol":"TONUSDT","side":"Buy","orderType":"Market","qty":"5","orderLinkId":"test-order-'${TIMESTAMP}'"}'

SIGN_STR="${TIMESTAMP}${BYBIT_API_KEY}${RECV_WINDOW}${BODY}"
SIGNATURE=$(echo -n "$SIGN_STR" | openssl dgst -sha256 -hmac "$BYBIT_API_SECRET" | awk '{print $2}')

echo "=== REQUEST ==="
echo "POST ${BYBIT_BASE_URL}/v5/order/create"
echo ""
echo "Headers:"
echo "  Content-Type: application/json"
echo "  X-BAPI-API-KEY: ${BYBIT_API_KEY}"
echo "  X-BAPI-TIMESTAMP: ${TIMESTAMP}"
echo "  X-BAPI-RECV-WINDOW: ${RECV_WINDOW}"
echo "  X-BAPI-SIGN: ${SIGNATURE}"
echo ""
echo "Body:"
echo "$BODY" | python3 -m json.tool
echo ""
echo "=== RESPONSE ==="

RESPONSE=$(curl -s -D - -X POST "${BYBIT_BASE_URL}/v5/order/create" \
  -H "Content-Type: application/json" \
  -H "X-BAPI-API-KEY: ${BYBIT_API_KEY}" \
  -H "X-BAPI-TIMESTAMP: ${TIMESTAMP}" \
  -H "X-BAPI-RECV-WINDOW: ${RECV_WINDOW}" \
  -H "X-BAPI-SIGN: ${SIGNATURE}" \
  -d "$BODY")

# split headers and body
HEADERS=$(echo "$RESPONSE" | sed '/^\r$/q')
RESP_BODY=$(echo "$RESPONSE" | sed '1,/^\r$/d')

echo "Response Headers:"
echo "$HEADERS"
echo ""
echo "Response Body:"
echo "$RESP_BODY" | python3 -m json.tool 2>/dev/null || echo "$RESP_BODY"

echo ""
echo "=== Traceid ==="
echo "$HEADERS" | grep -i traceid
