package dto

type DeviceTokenRequest struct {
	Token    string `json:"token"`
	Platform string `json:"platform"`
}
