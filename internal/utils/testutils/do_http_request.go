package testutils

import (
	"bytes"
	"net/http"
	"net/http/httptest"
)

// DoHTTPRequest performs an HTTP request to the provided router and returns the response recorder.
func DoHTTPRequest(router http.Handler, method, path string, body []byte, headers map[string]string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(method, path, bytes.NewBuffer(body))

	for key, value := range headers {
		req.Header.Set(key, value)
	}

	router.ServeHTTP(w, req)

	return w
}
