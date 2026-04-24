#!/bin/bash

# load api key and api secret from .env
export $(grep -v '^#' .env | xargs)

RECV_WINDOW="5000"
TIMESTAMP=$(python3 -c "import time; print(int(time.time() * 1000))")

BODY='{"category":"linear","symbol":"TONUSDT","side":"Buy","orderType":"Market","qty":"1","orderLinkId":"test-order-123"}'

SIGN_STR="${TIMESTAMP}${BYBIT_API_KEY}${RECV_WINDOW}${BODY}"
SIGNATURE=$(echo -n "$SIGN_STR" | openssl dgst -sha256 -hmac "$BYBIT_API_SECRET" | awk '{print $2}')

# Try to create order with curl

curl -v -X POST "${BYBIT_BASE_URL}/v5/order/create" \
  -H "Content-Type: application/json" \
  -H "X-BAPI-API-KEY: ${BYBIT_API_KEY}" \
  -H "X-BAPI-TIMESTAMP: ${TIMESTAMP}" \
  -H "X-BAPI-RECV-WINDOW: ${RECV_WINDOW}" \
  -H "X-BAPI-SIGN: ${SIGNATURE}" \
  -d "$BODY"
