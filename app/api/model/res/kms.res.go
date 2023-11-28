package res

type AccountRes struct {
	KeyID   string `json:"keyID" example:"f50a9229-e7c7-45ba-b06c-8036b894424e"`
	Address string `json:"address" example:"0x216690cD286d8a9c8D39d9714263bB6AB97046F3"`
}

type AccountListRes struct {
	Accounts []AccountRes `json:"accounts"`
	Marker   string       `json:"marker" example:"AE0AAAACAHMAAAAJYWNjb3VudElkAHMAAAAMOTg1MDk2Mzk3ODIxAHMAAAAEdGtJZABzAAAAJDQ0YTAzNWU2LTY1OTEtNDgwMC04YjcwLWM3MzNiNTI2MzljMw"`
}

type AddressRes struct {
	Address string `json:"address" example:"0x216690cD286d8a9c8D39d9714263bB6AB97046F3"`
}
