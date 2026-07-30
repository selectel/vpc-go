package api

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestTagsPathsAndMethods(t *testing.T) {
	responses := []*http.Response{
		{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"tags":["one"]}`))},
		{StatusCode: http.StatusNoContent, Body: io.NopCloser(strings.NewReader(""))},
		{StatusCode: http.StatusCreated, Body: io.NopCloser(strings.NewReader(""))},
		{StatusCode: http.StatusNoContent, Body: io.NopCloser(strings.NewReader(""))},
		{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"tags":["two"]}`))},
		{StatusCode: http.StatusNoContent, Body: io.NopCloser(strings.NewReader(""))},
	}
	httpClient := &recordingHTTPClient{}
	// Bind the response index without adding request behavior to production code.
	responseIndex := 0
	httpClient.do = func(*http.Request) (*http.Response, error) {
		response := responses[responseIndex]
		responseIndex++
		return response, nil
	}
	client, err := NewClient(Config{
		Endpoint:   "https://network.example.test",
		Token:      "token",
		HTTPClient: httpClient,
	})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	operations := NewTagOperations(client, "security-groups", "resource-id")

	if _, err := operations.Get(context.Background()); err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if _, err := operations.Has(context.Background(), "one/tag"); err != nil {
		t.Fatalf("Has() error = %v", err)
	}
	if err := operations.Add(context.Background(), "one/tag"); err != nil {
		t.Fatalf("Add() error = %v", err)
	}
	if err := operations.Delete(context.Background(), "one/tag"); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	replaced, err := operations.Replace(context.Background(), []string{"two"})
	if err != nil {
		t.Fatalf("Replace() error = %v", err)
	}
	if len(replaced) != 1 || replaced[0] != "two" {
		t.Fatalf("Replace() = %v, want [two]", replaced)
	}
	if err := operations.DeleteAll(context.Background()); err != nil {
		t.Fatalf("DeleteAll() error = %v", err)
	}

	wantMethods := []string{"GET", "GET", "PUT", "DELETE", "PUT", "DELETE"}
	wantPaths := []string{
		"/v2.0/security-groups/resource-id/tags",
		"/v2.0/security-groups/resource-id/tags/one%2Ftag",
		"/v2.0/security-groups/resource-id/tags/one%2Ftag",
		"/v2.0/security-groups/resource-id/tags/one%2Ftag",
		"/v2.0/security-groups/resource-id/tags",
		"/v2.0/security-groups/resource-id/tags",
	}
	if len(httpClient.requests) != len(wantMethods) {
		t.Fatalf("request count = %d, want %d", len(httpClient.requests), len(wantMethods))
	}
	for index, request := range httpClient.requests {
		if request.Method != wantMethods[index] || request.URL.EscapedPath() != wantPaths[index] {
			t.Fatalf(
				"request %d = %s %s, want %s %s",
				index,
				request.Method,
				request.URL.EscapedPath(),
				wantMethods[index],
				wantPaths[index],
			)
		}
	}
}

func TestTagsReplaceUsesOneRequestWithoutRead(t *testing.T) {
	httpClient := &recordingHTTPClient{
		do: func(*http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"tags":[]}`)),
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

	replaced, err := NewTagOperations(client, "networks", "id").
		Replace(context.Background(), []string{})
	if err != nil {
		t.Fatalf("Replace() error = %v", err)
	}
	if replaced == nil || len(replaced) != 0 {
		t.Fatalf("Replace() = %#v, want non-nil empty slice", replaced)
	}
	if len(httpClient.requests) != 1 || httpClient.requests[0].Method != http.MethodPut {
		t.Fatalf("requests = %+v, want one PUT", httpClient.requests)
	}
}

func TestTagsForbiddenUsesOneRequest(t *testing.T) {
	httpClient := &recordingHTTPClient{
		do: func(*http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusForbidden,
				Body: io.NopCloser(strings.NewReader(
					`{"NeutronError":{"type":"PolicyNotAuthorized","message":"blocked"}}`,
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

	operations := NewTagOperations(client, "ports", "id")
	err = operations.Add(context.Background(), "tag")
	if !IsErrorClass(err, ErrorClassForbidden) {
		t.Fatalf("Add() error = %v, want forbidden class", err)
	}
	if len(httpClient.requests) != 1 {
		t.Fatalf("request count = %d, want 1", len(httpClient.requests))
	}
}
