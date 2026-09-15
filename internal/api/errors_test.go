package api

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestErrorSourcesAreDistinct(t *testing.T) {
	clientErr := &ClientError{Err: errors.New("encode")}
	transportErr := &TransportError{Err: errors.New("connection reset")}
	apiErr := NewAPIError(http.StatusNotFound, []byte(
		`{"NeutronError":{"type":"NetworkNotFound","message":"missing","detail":"network id"}}`,
	))

	if _, ok := errors.AsType[*ClientError](clientErr); !ok {
		t.Fatal("ClientError is not distinguishable")
	}
	if _, ok := errors.AsType[*TransportError](transportErr); !ok {
		t.Fatal("TransportError is not distinguishable")
	}
	if _, ok := errors.AsType[*Error](apiErr); !ok {
		t.Fatal("APIError is not distinguishable")
	}
	if apiErr.StatusCode != http.StatusNotFound ||
		apiErr.Type != "NetworkNotFound" ||
		apiErr.Message != "missing" ||
		apiErr.Detail != "network id" {
		t.Fatalf("APIError fields = %+v", apiErr)
	}
	if len(apiErr.RawBody) == 0 {
		t.Fatal("APIError did not preserve raw body")
	}
}

func TestErrorClasses(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		errorType  string
		want       ErrorClass
	}{
		{"bad request", 400, "BadRequest", ErrorClassBadRequest},
		{"not authenticated", 401, "NotAuthenticated", ErrorClassNotAuthenticated},
		{"forbidden", 403, "PolicyNotAuthorized", ErrorClassForbidden},
		{"not found", 404, "NetworkNotFound", ErrorClassNotFound},
		{"conflict", 409, "NetworkInUse", ErrorClassConflict},
		{"server", 503, "ServiceUnavailable", ErrorClassServer},
		{"redirect is unclassified", 302, "", ErrorClassUnclassified},
		{"teapot is unclassified", 418, "Teapot", ErrorClassUnclassified},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			body := []byte(`{"NeutronError":{"type":"` + test.errorType + `","message":"failure"}}`)
			err := NewAPIError(test.statusCode, body)
			if err.Class != test.want {
				t.Fatalf("class = %q, want %q", err.Class, test.want)
			}
		})
	}
}

func TestRequestPreservesAPIError(t *testing.T) {
	httpClient := &recordingHTTPClient{
		do: func(*http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusForbidden,
				Body: io.NopCloser(strings.NewReader(
					`{"NeutronError":{"type":"PolicyNotAuthorized","message":"forbidden"}}`,
				)),
			}, nil
		},
	}
	client, err := NewClient(Config{
		Endpoint:   "https://network.example.test",
		Token:      "token",
		HTTPClient: httpClient,
	})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	err = Request(
		context.Background(), client,
		http.MethodPut,
		"/v2.0/networks/id",
		nil,
		map[string]any{"network": map[string]string{"name": "new"}},
		nil,
		http.StatusOK,
	)
	if !IsErrorClass(err, ErrorClassForbidden) {
		t.Fatalf("Request() error = %v, want forbidden class", err)
	}

	var apiErr *Error
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusForbidden {
		t.Fatalf("Request() did not preserve API error: %v", err)
	}
}

func TestErrorUnexpectedResponse(t *testing.T) {
	httpClient := &recordingHTTPClient{
		do: func(*http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"network":`)),
			}, nil
		},
	}
	client, err := NewClient(Config{
		Endpoint:   "https://network.example.test",
		Token:      "token",
		HTTPClient: httpClient,
	})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	var target map[string]any
	err = Request(
		context.Background(), client,
		http.MethodGet,
		"/v2.0/networks/id",
		nil,
		nil,
		&target,
		http.StatusOK,
	)
	if !IsErrorClass(err, ErrorClassUnexpectedResponse) {
		t.Fatalf("Request() error = %v, want unexpected response class", err)
	}
}

func TestRequestWrapsTransportError(t *testing.T) {
	connectionErr := errors.New("connection lost")
	httpClient := &recordingHTTPClient{
		do: func(*http.Request) (*http.Response, error) {
			return nil, connectionErr
		},
	}
	client, err := NewClient(Config{
		Endpoint:   "https://network.example.test",
		Token:      "token",
		HTTPClient: httpClient,
	})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	err = Request(
		context.Background(), client,
		http.MethodPost,
		"/v2.0/networks",
		nil,
		map[string]any{"network": map[string]string{"name": "new"}},
		nil,
		http.StatusCreated,
	)
	var transportErr *TransportError
	if !errors.As(err, &transportErr) || !errors.Is(err, connectionErr) {
		t.Fatalf("Request() error = %v, want transport error", err)
	}
}

func TestErrorMessages(t *testing.T) {
	cause := errors.New("connection reset")
	tests := []struct {
		name string
		err  error
		want string
	}{
		{
			"client",
			&ClientError{Err: cause},
			"vpc client error: connection reset",
		},
		{
			"transport",
			&TransportError{Err: cause},
			"vpc transport error: connection reset",
		},
		{
			"api with message",
			&Error{StatusCode: 404, Type: "NetworkNotFound", Message: "missing"},
			"vpc API error (404, NetworkNotFound): missing",
		},
		{
			"api without message",
			&Error{StatusCode: 503, Type: "ServiceUnavailable"},
			"vpc API error (503, ServiceUnavailable)",
		},
		{
			"unexpected response",
			&UnexpectedResponseError{StatusCode: 200, Err: cause},
			"unexpected vpc API response (200): connection reset",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := test.err.Error(); got != test.want {
				t.Fatalf("Error() = %q, want %q", got, test.want)
			}
		})
	}
}

func TestErrorsUnwrapToCause(t *testing.T) {
	cause := errors.New("cause")
	for _, err := range []error{
		&ClientError{Err: cause},
		&TransportError{Err: cause},
		&UnexpectedResponseError{Err: cause},
	} {
		if !errors.Is(err, cause) {
			t.Fatalf("%T does not unwrap to its cause", err)
		}
	}
}

func TestRequestRejectsUnencodableBody(t *testing.T) {
	httpClient := &recordingHTTPClient{}
	client, err := NewClient(Config{
		Endpoint:   "https://network.example.test",
		Token:      "token",
		HTTPClient: httpClient,
	})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	err = Request(context.Background(), client, http.MethodPost, "/v2.0/networks", nil,
		make(chan int), nil, http.StatusCreated)
	if _, ok := errors.AsType[*ClientError](err); !ok {
		t.Fatalf("Request() error = %v, want ClientError", err)
	}
	if len(httpClient.requests) != 0 {
		t.Fatalf("request count = %d, want 0", len(httpClient.requests))
	}
}
