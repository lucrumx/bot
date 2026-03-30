package dtos

// AuthRespDTO present response for private ws auth request
type AuthRespDTO struct {
	Success bool   `json:"success"`
	RetMsg  string `json:"ret_msg"`
	Op      string `json:"op"`
	ConnID  string `json:"conn_id"`
}
