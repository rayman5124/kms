package dto

// req
type TxnReq struct {
	KeyID         string `json:"keyID" validate:"required,ascii" example:"f50a9229-e7c7-45ba-b06c-8036b894424e"`
	SerializedTxn string `json:"serializedTxn" validate:"required,hexadecimal" example:"0xea5685ba43b740008252089439e243a7f209932df41e1fc0a1ada51b3a04b46d018086059407ad8e8b8080"`
}

// res
type SingedTxnRes struct {
	SignedTxn string `json:"signedTxn" example:"0xf86a5685ba43b740008252089439e243a7f209932df41e1fc0a1ada51b3a04b46d0180860b280f5b1d3aa00d2ea43cfd9b91151348d037a5a80293f543e1700a7019853f28063f6442c826a052a29797169740b1bc48962e197299061c8aa3314951a9c71418d19036604645"`
}
