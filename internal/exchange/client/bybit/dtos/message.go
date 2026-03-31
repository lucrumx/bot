package dtos

import "encoding/json"

// MessageDTO represent message in private stream
type MessageDTO struct {
	Topic        string          `json:"topic"`
	ID           string          `json:"id"`
	CreationTime int64           `json:"creationTime"`
	Data         json.RawMessage `json:"data"`
}
