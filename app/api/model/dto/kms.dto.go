package dto

// req
type KeyIdReq struct {
	KeyID string `json:"keyID" validate:"required,ascii,min=1,max=2048" example:"f50a9229-e7c7-45ba-b06c-8036b894424e"`
}

type PkReq struct {
	PK string `json:"pk" validate:"required,sha256" example:"637081577126ff5c2d327f992bd66548e00262578fd558d7fe272ef21b8bf825"`
}

type AccountListReq struct {
	Limit  *int32  `json:"limit" validate:"omitempty,numeric,gte=1,lte=1000" example:"100"`
	Marker *string `json:"marker" validate:"omitempty,marker,max=1024,min=1"`
}

// res
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

type AccountDeletionRes struct {
	KeyID        string `json:"keyID" example:"f50a9229-e7c7-45ba-b06c-8036b894424e"`
	DeletionDate string `json:"deletionDate" example:"2023-12-11 03:21:18 +0000 UTC"`
}
