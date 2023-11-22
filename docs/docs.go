// Code generated by swaggo/swag. DO NOT EDIT.

package docs

import "github.com/swaggo/swag"

const docTemplate = `{
    "schemes": {{ marshal .Schemes }},
    "swagger": "2.0",
    "info": {
        "description": "{{escape .Description}}",
        "title": "{{.Title}}",
        "contact": {},
        "version": "{{.Version}}"
    },
    "host": "{{.Host}}",
    "basePath": "{{.BasePath}}",
    "paths": {
        "/api/account": {
            "post": {
                "produces": [
                    "application/json"
                ],
                "summary": "Create new account",
                "responses": {
                    "201": {
                        "description": "Created",
                        "schema": {
                            "$ref": "#/definitions/res.AccountRes"
                        }
                    }
                }
            }
        },
        "/api/account/keyID/{keyID}": {
            "get": {
                "produces": [
                    "application/json"
                ],
                "summary": "Get account of target key id",
                "parameters": [
                    {
                        "type": "string",
                        "description": "kms key-id",
                        "name": "keyID",
                        "in": "path",
                        "required": true
                    }
                ],
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "$ref": "#/definitions/res.AccountRes"
                        }
                    }
                }
            }
        },
        "/api/account/list": {
            "get": {
                "produces": [
                    "application/json"
                ],
                "summary": "Get accounst list",
                "parameters": [
                    {
                        "type": "integer",
                        "example": 100,
                        "name": "limit",
                        "in": "query"
                    },
                    {
                        "type": "string",
                        "name": "marker",
                        "in": "query"
                    }
                ],
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "$ref": "#/definitions/res.AccountListRes"
                        }
                    }
                }
            }
        },
        "/api/import/account": {
            "post": {
                "produces": [
                    "application/json"
                ],
                "summary": "Import account to kms",
                "parameters": [
                    {
                        "description": "subject",
                        "name": "subject",
                        "in": "body",
                        "required": true,
                        "schema": {
                            "$ref": "#/definitions/dto.ImportAccountDTO"
                        }
                    }
                ],
                "responses": {
                    "201": {
                        "description": "Created",
                        "schema": {
                            "$ref": "#/definitions/res.AccountRes"
                        }
                    }
                }
            }
        },
        "/api/txn": {
            "get": {
                "produces": [
                    "application/json"
                ],
                "summary": "Get Serialized Txn",
                "parameters": [
                    {
                        "type": "string",
                        "name": "data",
                        "in": "query"
                    },
                    {
                        "type": "string",
                        "name": "from",
                        "in": "query",
                        "required": true
                    },
                    {
                        "type": "integer",
                        "name": "gas",
                        "in": "query"
                    },
                    {
                        "type": "string",
                        "name": "gasPrice",
                        "in": "query"
                    },
                    {
                        "type": "integer",
                        "name": "nonce",
                        "in": "query"
                    },
                    {
                        "type": "string",
                        "name": "to",
                        "in": "query"
                    },
                    {
                        "type": "string",
                        "name": "value",
                        "in": "query"
                    }
                ],
                "responses": {
                    "200": {
                        "description": "OK"
                    }
                }
            },
            "post": {
                "produces": [
                    "application/json"
                ],
                "summary": "Send serialized transaction.",
                "parameters": [
                    {
                        "description": "subject",
                        "name": "subject",
                        "in": "body",
                        "required": true,
                        "schema": {
                            "$ref": "#/definitions/dto.TxnDTO"
                        }
                    }
                ],
                "responses": {
                    "201": {
                        "description": "Created",
                        "schema": {
                            "$ref": "#/definitions/res.TxnRes"
                        }
                    }
                }
            }
        }
    },
    "definitions": {
        "dto.ImportAccountDTO": {
            "type": "object",
            "required": [
                "pk"
            ],
            "properties": {
                "pk": {
                    "type": "string",
                    "example": "637081577126ff5c2d327f992bd66548e00262578fd558d7fe272ef21b8bf825"
                }
            }
        },
        "dto.TxnDTO": {
            "type": "object",
            "required": [
                "keyID",
                "serializedTxn"
            ],
            "properties": {
                "keyID": {
                    "type": "string",
                    "example": "f50a9229-e7c7-45ba-b06c-8036b894424e"
                },
                "serializedTxn": {
                    "type": "string",
                    "example": "0xea5685ba43b740008252089439e243a7f209932df41e1fc0a1ada51b3a04b46d018086059407ad8e8b8080"
                }
            }
        },
        "res.AccountListRes": {
            "type": "object",
            "properties": {
                "accounts": {
                    "type": "array",
                    "items": {
                        "$ref": "#/definitions/res.AccountRes"
                    }
                },
                "marker": {
                    "type": "string"
                }
            }
        },
        "res.AccountRes": {
            "type": "object",
            "properties": {
                "address": {
                    "type": "string",
                    "example": "0x216690cD286d8a9c8D39d9714263bB6AB97046F3"
                },
                "keyID": {
                    "type": "string",
                    "example": "f50a9229-e7c7-45ba-b06c-8036b894424e"
                }
            }
        },
        "res.TxnRes": {
            "type": "object",
            "properties": {
                "transaction hash": {
                    "type": "string",
                    "example": "0x661e2747abdb7c11197f6c7d9a89d4b543248bb7dd7f08a27cacb2f2f7474c0f"
                }
            }
        }
    }
}`

// SwaggerInfo holds exported Swagger Info so clients can modify it
var SwaggerInfo = &swag.Spec{
	Version:          "",
	Host:             "",
	BasePath:         "",
	Schemes:          []string{},
	Title:            "",
	Description:      "",
	InfoInstanceName: "swagger",
	SwaggerTemplate:  docTemplate,
	LeftDelim:        "{{",
	RightDelim:       "}}",
}

func init() {
	swag.Register(SwaggerInfo.InstanceName(), SwaggerInfo)
}
