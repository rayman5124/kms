package res

type AccountRes struct {
	KeyID   string `json:"keyID" example:"f50a9229-e7c7-45ba-b06c-8036b894424e"`
	Address string `json:"address" example:"0x216690cD286d8a9c8D39d9714263bB6AB97046F3"`
}

type AccountListRes struct {
	Accounts []AccountRes `json:"accounts"`
	Marker   string       `json:"marker"`
}

// ec6f85ba43b74000825208945f90d10443b03f46a6c3513fe62f60733e7bcea7880de0b6b3a764000080808080
// e96f85ba43b74000825208945f90d10443b03f46a6c3513fe62f60733e7bcea7880de0b6b3a764000080
// 0xf27085ba43b74000825208945f90d10443b03f46a6c3513fe62f60733e7bcea7880de0b6b3a76400008086059407ad8e8b8080
// 0xe97085ba43b74000825208945f90d10443b03f46a6c3513fe62f60733e7bcea7880de0b6b3a764000080
