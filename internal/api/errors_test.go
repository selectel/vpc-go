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

	var gotClient *ClientError
	var gotTransport *TransportError
	var gotAPI *Error
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
		{"external address", 400, "ExternalIpAddressExhausted", ErrorClassAddressUnavailable},
		{"server", 503, "ServiceUnavailable", ErrorClassServer},
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
		RequestOptions{ExpectedStatus: []int{http.StatusOK}},
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
		RequestOptions{ExpectedStatus: []int{http.StatusOK}},
	)
	if !IsErrorClass(err, ErrorClassUnexpectedResponse) {
		t.Fatalf("Request() error = %v, want unexpected response class", err)
	}
}

func TestErrorUnexpectedResourceEnvelope(t *testing.T) {
	for _, body := range []string{
		`{}`,
		`null`,
		`{"network":null}`,
		`{"network":{}}`,
	} {
		t.Run(body, func(t *testing.T) {
			httpClient := &recordingHTTPClient{
				do: func(*http.Request) (*http.Response, error) {
					return &http.Response{
						StatusCode: http.StatusOK,
						Body:       io.NopCloser(strings.NewReader(body)),
					}, nil
				},
			}
			client, err := NewClient(Config{
				Endpoint: "https://network.example.test", Token: "token", HTTPClient: httpClient,
			})
			if err != nil {
				t.Fatal(err)
			}
			var target struct {
				Network struct {
					ID string `json:"id"`
				} `json:"network"`
			}
			err = Request(
				context.Background(), client, http.MethodGet, "/v2.0/networks/id",
				nil, nil, &target, RequestOptions{ExpectedStatus: []int{http.StatusOK}},
			)
			if !IsErrorClass(err, ErrorClassUnexpectedResponse) {
				t.Fatalf("Request() error = %v, want unexpected response", err)
			}
		})
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
		RequestOptions{ExpectedStatus: []int{http.StatusCreated}},
	)
	var transportErr *TransportError
	if !errors.As(err, &transportErr) || !errors.Is(err, connectionErr) {
		t.Fatalf("Request() error = %v, want transport error", err)
	}
}
