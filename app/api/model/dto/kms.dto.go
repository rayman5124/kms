package dto

type KeyIdDTO struct {
	KeyID string `json:"keyID" validate:"required,ascii" example:"f50a9229-e7c7-45ba-b06c-8036b894424e"`
}

type PkDTO struct {
	PK string `json:"pk" validate:"required,sha256" example:"637081577126ff5c2d327f992bd66548e00262578fd558d7fe272ef21b8bf825"`
}

type AccountListDTO struct {
	Limit  *int32  `json:"limit" validate:"omitempty,numeric,lte=1000" example:"100"`
	Marker *string `json:"marker" validate:"omitempty,marker"`
}
