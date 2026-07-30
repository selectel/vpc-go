package v2

import "github.com/selectel/vpc-go/internal/api"

// ErrorClass is a stable programmatic classification of an API failure.
type ErrorClass = api.ErrorClass

const (
	ErrorClassBadRequest         = api.ErrorClassBadRequest
	ErrorClassNotAuthenticated   = api.ErrorClassNotAuthenticated
	ErrorClassForbidden          = api.ErrorClassForbidden
	ErrorClassNotFound           = api.ErrorClassNotFound
	ErrorClassConflict           = api.ErrorClassConflict
	ErrorClassQuotaExceeded      = api.ErrorClassQuotaExceeded
	ErrorClassAddressUnavailable = api.ErrorClassAddressUnavailable
	ErrorClassServer             = api.ErrorClassServer
	ErrorClassUnexpectedResponse = api.ErrorClassUnexpectedResponse
	ErrorClassUnclassified       = api.ErrorClassUnclassified
)

// ClientError reports request construction, encoding, or decoding failure.
type ClientError = api.ClientError

// TransportError reports a failure before an HTTP response was received.
type TransportError = api.TransportError

// APIError reports a non-success response from the network API.
//
// A NotFound error returned by an update or delete operation does not prove
// that the resource is absent: Neutron can mask a policy rejection as 404.
type APIError = api.Error

// UnexpectedResponseError reports a success-status response with an invalid
// representation for the requested operation.
type UnexpectedResponseError = api.UnexpectedResponseError

// IsErrorClass reports whether err belongs to class.
func IsErrorClass(err error, class ErrorClass) bool {
	return api.IsErrorClass(err, class)
}
