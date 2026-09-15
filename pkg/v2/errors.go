package v2

import "github.com/selectel/vpc-go/internal/api"

type ErrorClass = api.ErrorClass

const (
	ErrorClassBadRequest         = api.ErrorClassBadRequest
	ErrorClassNotAuthenticated   = api.ErrorClassNotAuthenticated
	ErrorClassForbidden          = api.ErrorClassForbidden
	ErrorClassNotFound           = api.ErrorClassNotFound
	ErrorClassConflict           = api.ErrorClassConflict
	ErrorClassServer             = api.ErrorClassServer
	ErrorClassUnexpectedResponse = api.ErrorClassUnexpectedResponse
	ErrorClassUnclassified       = api.ErrorClassUnclassified
)

type ClientError = api.ClientError

type TransportError = api.TransportError

type APIError = api.Error

type UnexpectedResponseError = api.UnexpectedResponseError

func IsErrorClass(err error, class ErrorClass) bool {
	return api.IsErrorClass(err, class)
}
