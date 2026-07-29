package v2

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

type recordingHTTPClient struct {
	requests []*http.Request
	do       func(*http.Request) (*http.Response, error)
}

func (client *recordingHTTPClient) Do(request *http.Request) (*http.Response, error) {
	client.requests = append(client.requests, request)
	if client.do != nil {
		return client.do(request)
	}

	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader("{}")),
	}, nil
}

func TestClientRequestUsesConfiguredScope(t *testing.T) {
	httpClient := &recordingHTTPClient{}
	client, err := NewClient(Config{
		Endpoint:   "https://network.example.test/base",
		Token:      "project-token",
		UserAgent:  "consumer/1.0",
		HTTPClient: httpClient,
	})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	response, err := client.Do(
		context.Background(),
		http.MethodPost,
		"/v2.0/networks",
		url.Values{"fields": {"id", "name"}},
		strings.NewReader(`{"network":{"name":"test"}}`),
	)
	if err != nil {
		t.Fatalf("Do() error = %v", err)
	}
	_ = response.Body.Close()

	if len(httpClient.requests) != 1 {
		t.Fatalf("request count = %d, want 1", len(httpClient.requests))
	}

	request := httpClient.requests[0]
	if request.URL.String() != "https://network.example.test/base/v2.0/networks?fields=id&fields=name" {
		t.Fatalf("request URL = %q", request.URL.String())
	}
	if request.Header.Get(authTokenHeader) != "project-token" {
		t.Fatalf("%s header = %q", authTokenHeader, request.Header.Get(authTokenHeader))
	}
	if request.Header.Get(userAgentHeader) != "consumer/1.0 "+moduleUserAgent {
		t.Fatalf("%s header = %q", userAgentHeader, request.Header.Get(userAgentHeader))
	}
	if request.Header.Get("Content-Type") != "application/json" {
		t.Fatalf("Content-Type header = %q", request.Header.Get("Content-Type"))
	}
}

func TestClientCancellationStopsSingleRequest(t *testing.T) {
	httpClient := &recordingHTTPClient{
		do: func(request *http.Request) (*http.Response, error) {
			<-request.Context().Done()
			return nil, request.Context().Err()
		},
	}
	client, err := NewClient(Config{
		Endpoint:   "https://network.example.test",
		Token:      "project-token",
		HTTPClient: httpClient,
	})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	response, err := client.Do(ctx, http.MethodGet, "/v2.0/networks", nil, nil)
	if response != nil {
		t.Fatal("Do() returned a response for a canceled request")
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Do() error = %v, want context.Canceled", err)
	}
	if len(httpClient.requests) != 1 {
		t.Fatalf("request count = %d, want 1", len(httpClient.requests))
	}
}

func TestClientValidation(t *testing.T) {
	tests := []Config{
		{Token: "token"},
		{Endpoint: "https://network.example.test"},
		{Endpoint: "/relative", Token: "token"},
	}

	for _, config := range tests {
		if _, err := NewClient(config); err == nil {
			t.Fatalf("NewClient(%+v) returned nil error", config)
		}
	}
}
