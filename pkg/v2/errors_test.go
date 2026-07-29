package v2

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
	apiErr := newAPIError(http.StatusNotFound, []byte(
		`{"NeutronError":{"type":"NetworkNotFound","message":"missing","detail":"network id"}}`,
	))

	var gotClient *ClientError
	var gotTransport *TransportError
	var gotAPI *APIError
	if !errors.As(clientErr, &gotClient) {
		t.Fatal("ClientError is not distinguishable")
	}
	if !errors.As(transportErr, &gotTransport) {
		t.Fatal("TransportError is not distinguishable")
	}
	if !errors.As(apiErr, &gotAPI) {
		t.Fatal("APIError is not distinguishable")
	}
	if apiErr.StatusCode != http.StatusNotFound ||
		apiErr.Type != "NetworkNotFound" ||
		apiErr.Message != "missing" ||
		apiErr.Detail != "network id" {
		t.Fatalf("APIError fields = %+v", apiErr)
	}
	if string(apiErr.Raw()) == "" {
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
		{"quota", 409, "OverQuota", ErrorClassQuotaExceeded},
		{"address", 409, "IpAddressGenerationFailure", ErrorClassAddressUnavailable},
		{"server", 503, "ServiceUnavailable", ErrorClassServer},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			body := []byte(`{"NeutronError":{"type":"` + test.errorType + `","message":"failure"}}`)
			err := newAPIError(test.statusCode, body)
			if err.Class != test.want {
				t.Fatalf("class = %q, want %q", err.Class, test.want)
			}
		})
	}
}

func TestErrorBlockedDiagnostic(t *testing.T) {
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

	diagnosticReads := 0
	err = client.Request(
		context.Background(),
		http.MethodPut,
		"/v2.0/networks/id",
		nil,
		map[string]any{"network": map[string]string{"name": "new"}},
		nil,
		RequestOptions{
			ExpectedStatus: []int{http.StatusOK},
			ReadBlocked: func(context.Context) (bool, error) {
				diagnosticReads++
				return true, nil
			},
		},
	)
	if !IsErrorClass(err, ErrorClassResourceBlocked) {
		t.Fatalf("Request() error = %v, want blocked class", err)
	}
	if diagnosticReads != 1 {
		t.Fatalf("diagnostic read count = %d, want 1", diagnosticReads)
	}

	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusForbidden {
		t.Fatalf("Request() did not preserve API error: %v", err)
	}
}

func TestErrorBlockedDiagnosticFallback(t *testing.T) {
	for _, diagnostic := range []func(context.Context) (bool, error){
		func(context.Context) (bool, error) { return false, nil },
		func(context.Context) (bool, error) { return false, errors.New("not readable") },
	} {
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

		err = client.Request(
			context.Background(),
			http.MethodDelete,
			"/v2.0/networks/id",
			nil,
			nil,
			nil,
			RequestOptions{ExpectedStatus: []int{http.StatusNoContent}, ReadBlocked: diagnostic},
		)
		if !IsErrorClass(err, ErrorClassForbidden) {
			t.Fatalf("Request() error = %v, want forbidden class", err)
		}
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
	err = client.Request(
		context.Background(),
		http.MethodGet,
		"/v2.0/networks/id",
		nil,
		nil,
		&target,
		RequestOptions{ExpectedStatus: []int{http.StatusOK}},
	)
	if !IsErrorClass(err, ErrorClassUnexpectedResponse) {
		t.Fatalf("Request() error = %v, want unexpected response class", err)
	}
}

func TestErrorTransportDoesNotRetry(t *testing.T) {
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

	err = client.Request(
		context.Background(),
		http.MethodPost,
		"/v2.0/networks",
		nil,
		map[string]any{"network": map[string]string{"name": "new"}},
		nil,
		RequestOptions{ExpectedStatus: []int{http.StatusCreated}},
	)
	var transportErr *TransportError
	if !errors.As(err, &transportErr) || !errors.Is(err, connectionErr) {
		t.Fatalf("Request() error = %v, want transport error", err)
	}
	if len(httpClient.requests) != 1 {
		t.Fatalf("request count = %d, want 1", len(httpClient.requests))
	}
}
