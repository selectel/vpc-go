package v2

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
)

// ErrorClass is a stable programmatic classification of an API failure.
type ErrorClass string

const (
	ErrorClassBadRequest         ErrorClass = "bad_request"
	ErrorClassNotAuthenticated   ErrorClass = "not_authenticated"
	ErrorClassForbidden          ErrorClass = "forbidden"
	ErrorClassResourceBlocked    ErrorClass = "resource_blocked"
	ErrorClassNotFound           ErrorClass = "not_found"
	ErrorClassConflict           ErrorClass = "conflict"
	ErrorClassQuotaExceeded      ErrorClass = "quota_exceeded"
	ErrorClassAddressUnavailable ErrorClass = "address_unavailable"
	ErrorClassServer             ErrorClass = "server"
	ErrorClassUnexpectedResponse ErrorClass = "unexpected_response"
	ErrorClassUnclassified       ErrorClass = "unclassified"
)

// ClientError reports request construction, encoding, or decoding failure.
type ClientError struct {
	Err error
}

func (err *ClientError) Error() string {
	return "vpc client error: " + err.Err.Error()
}

func (err *ClientError) Unwrap() error {
	return err.Err
}

// TransportError reports a failure before an HTTP response was received.
type TransportError struct {
	Err error
}

func (err *TransportError) Error() string {
	return "vpc transport error: " + err.Err.Error()
}

func (err *TransportError) Unwrap() error {
	return err.Err
}

// APIError reports a non-success response from the network API.
//
// A NotFound error returned by an update or delete operation does not prove
// that the resource is absent: Neutron can mask a policy rejection as 404.
type APIError struct {
	StatusCode     int
	Type           string
	Message        string
	Detail         string
	ResourceStatus string
	Class          ErrorClass
	RawBody        []byte
}

func (err *APIError) Error() string {
	if err.Message != "" {
		return fmt.Sprintf("vpc API error (%d, %s): %s", err.StatusCode, err.Type, err.Message)
	}

	return fmt.Sprintf("vpc API error (%d, %s)", err.StatusCode, err.Type)
}

// Raw returns a copy of the original response body.
func (err *APIError) Raw() []byte {
	return append([]byte(nil), err.RawBody...)
}

// UnexpectedResponseError reports a success-status response with an invalid
// representation for the requested operation.
type UnexpectedResponseError struct {
	StatusCode int
	RawBody    []byte
	Err        error
}

func (err *UnexpectedResponseError) Error() string {
	return fmt.Sprintf("unexpected vpc API response (%d): %v", err.StatusCode, err.Err)
}

func (err *UnexpectedResponseError) Unwrap() error {
	return err.Err
}

// Raw returns a copy of the original response body.
func (err *UnexpectedResponseError) Raw() []byte {
	return append([]byte(nil), err.RawBody...)
}

// IsErrorClass reports whether err belongs to class.
func IsErrorClass(err error, class ErrorClass) bool {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.Class == class
	}

	var unexpectedErr *UnexpectedResponseError
	return class == ErrorClassUnexpectedResponse && errors.As(err, &unexpectedErr)
}

func newAPIError(statusCode int, body []byte) *APIError {
	var envelope struct {
		NeutronError struct {
			Type    string `json:"type"`
			Message string `json:"message"`
			Detail  string `json:"detail"`
			Status  string `json:"status"`
		} `json:"NeutronError"`
	}
	_ = json.Unmarshal(body, &envelope)

	apiErr := &APIError{
		StatusCode:     statusCode,
		Type:           envelope.NeutronError.Type,
		Message:        envelope.NeutronError.Message,
		Detail:         envelope.NeutronError.Detail,
		ResourceStatus: envelope.NeutronError.Status,
		RawBody:        append([]byte(nil), body...),
	}
	apiErr.Class = classifyAPIError(apiErr)

	return apiErr
}

func classifyAPIError(apiErr *APIError) ErrorClass {
	errorType := strings.ToLower(apiErr.Type)
	switch {
	case strings.Contains(errorType, "quota") || strings.Contains(errorType, "overquota"):
		return ErrorClassQuotaExceeded
	case strings.Contains(errorType, "ipaddressgenerationfailure") ||
		strings.Contains(errorType, "addressgenerationfailure"):
		return ErrorClassAddressUnavailable
	}

	switch apiErr.StatusCode {
	case http.StatusBadRequest:
		return ErrorClassBadRequest
	case http.StatusUnauthorized:
		return ErrorClassNotAuthenticated
	case http.StatusForbidden:
		return ErrorClassForbidden
	case http.StatusNotFound:
		return ErrorClassNotFound
	case http.StatusConflict:
		return ErrorClassConflict
	default:
		if apiErr.StatusCode >= http.StatusInternalServerError {
			return ErrorClassServer
		}
		return ErrorClassUnclassified
	}
}
