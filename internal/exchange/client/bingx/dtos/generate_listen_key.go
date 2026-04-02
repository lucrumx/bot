package dtos

// GenerateListenKeyDTO represents the structure for the response of a listen key generation request.
type GenerateListenKeyDTO struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data struct {
		ListenKey string `json:"listenKey"`
	} `json:"data"`
}
