package bingx

import (
	"bytes"
	"compress/gzip"
	"io"
)

func decodeGzip(data []byte) (string, error) {
	reader, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return "", err
	}
	defer func() {
		_ = reader.Close()
	}()

	var decodedMsg []byte
	decodedMsg, err = io.ReadAll(reader)
	if err != nil {
		return "", err
	}

	return string(decodedMsg), nil
}
