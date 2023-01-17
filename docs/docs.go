// Package docs GENERATED BY SWAG; DO NOT EDIT
// This file was generated by swaggo/swag
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
        "/": {
            "get": {
                "produces": [
                    "application/json"
                ],
                "summary": "API greeting message",
                "responses": {
                    "200": {
                        "description": "OK"
                    }
                }
            }
        },
        "/photo": {
            "post": {
                "consumes": [
                    "application/json"
                ],
                "produces": [
                    "application/json"
                ],
                "summary": "Upload a new Photo",
                "parameters": [
                    {
                        "description": "name of the photo",
                        "name": "name",
                        "in": "body",
                        "required": true,
                        "schema": {
                            "type": "string"
                        }
                    }
                ],
                "responses": {
                    "202": {
                        "description": "Accepted",
                        "schema": {
                            "$ref": "#/definitions/main.Photo"
                        }
                    }
                }
            }
        },
        "/photo/{cid}": {
            "get": {
                "produces": [
                    "application/json"
                ],
                "summary": "Download existing photo",
                "parameters": [
                    {
                        "type": "string",
                        "description": "first name",
                        "name": "cid",
                        "in": "query",
                        "required": true
                    }
                ],
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "type": "string"
                        }
                    }
                }
            }
        }
    },
    "definitions": {
        "main.Photo": {
            "type": "object",
            "properties": {
                "cid": {
                    "type": "string"
                },
                "id": {
                    "type": "integer"
                },
                "name": {
                    "type": "string"
                },
                "owner": {
                    "$ref": "#/definitions/main.User"
                }
            }
        },
        "main.User": {
            "type": "object",
            "properties": {
                "apikey": {
                    "type": "string"
                },
                "email": {
                    "type": "string"
                },
                "handle": {
                    "type": "string"
                },
                "id": {
                    "type": "integer"
                }
            }
        }
    }
}`

// SwaggerInfo holds exported Swagger Info so clients can modify it
var SwaggerInfo = &swag.Spec{
	Version:          "2.0",
	Host:             "",
	BasePath:         "/",
	Schemes:          []string{"http"},
	Title:            "People API",
	Description:      "",
	InfoInstanceName: "swagger",
	SwaggerTemplate:  docTemplate,
}

func init() {
	swag.Register(SwaggerInfo.InstanceName(), SwaggerInfo)
}
