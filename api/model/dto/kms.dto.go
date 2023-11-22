package dto

type AccountDTO struct {
	KeyID string `json:"keyID" validate:"required" example:"f50a9229-e7c7-45ba-b06c-8036b894424e"`
}

type AccountListDTO struct {
	Limit  *int32  `json:"limit" validate:"omitempty,numeric" example:"100"`
	Marker *string `json:"marker" validate:"omitempty"`
}

type ImportAccountDTO struct {
	PK string `json:"pk" validate:"required,sha256" example:"637081577126ff5c2d327f992bd66548e00262578fd558d7fe272ef21b8bf825"`
}