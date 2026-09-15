package api

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

func newTagTestOperations(
	t *testing.T,
	status int,
	body string,
) (TagOperations, *recordingHTTPClient) {
	t.Helper()
	httpClient := &recordingHTTPClient{do: func(*http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: status,
			Body:       io.NopCloser(strings.NewReader(body)),
		}, nil
	}}
	client, err := NewClient(Config{
		Endpoint:   "https://network.example.test",
		Token:      "token",
		HTTPClient: httpClient,
	})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	return NewTagOperations(client, "security-groups", "resource-id"), httpClient
}

func TestTagsGetPath(t *testing.T) {
	operations, transport := newTagTestOperations(t, http.StatusOK, `{"tags":["one"]}`)
	if _, err := operations.Get(context.Background()); err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	assertTagRequest(t, transport, http.MethodGet, "/v2.0/security-groups/resource-id/tags")
}

func TestTagsHasEscapesPath(t *testing.T) {
	operations, transport := newTagTestOperations(t, http.StatusNoContent, "")
	if _, err := operations.Has(context.Background(), "one/tag"); err != nil {
		t.Fatalf("Has() error = %v", err)
	}
	assertTagRequest(t, transport, http.MethodGet, "/v2.0/security-groups/resource-id/tags/one%2Ftag")
}

func TestTagsAddPath(t *testing.T) {
	operations, transport := newTagTestOperations(t, http.StatusCreated, "")
	if err := operations.Add(context.Background(), "one/tag"); err != nil {
		t.Fatalf("Add() error = %v", err)
	}
	assertTagRequest(t, transport, http.MethodPut, "/v2.0/security-groups/resource-id/tags/one%2Ftag")
}

func TestTagsDeletePath(t *testing.T) {
	operations, transport := newTagTestOperations(t, http.StatusNoContent, "")
	if err := operations.Delete(context.Background(), "one/tag"); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	assertTagRequest(t, transport, http.MethodDelete, "/v2.0/security-groups/resource-id/tags/one%2Ftag")
}

func TestTagsDeleteIsIdempotent(t *testing.T) {
	operations, transport := newTagTestOperations(t, http.StatusNotFound, "")
	if err := operations.Delete(context.Background(), "missing"); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	assertTagRequest(t, transport, http.MethodDelete,
		"/v2.0/security-groups/resource-id/tags/missing")
}

func TestTagsReplacePath(t *testing.T) {
	operations, transport := newTagTestOperations(t, http.StatusOK, `{"tags":["two"]}`)
	replaced, err := operations.Replace(context.Background(), []string{"two"})
	if err != nil {
		t.Fatalf("Replace() error = %v", err)
	}
	if len(replaced) != 1 || replaced[0] != "two" {
		t.Fatalf("Replace() = %v, want [two]", replaced)
	}
	assertTagRequest(t, transport, http.MethodPut, "/v2.0/security-groups/resource-id/tags")
}

func TestTagsDeleteAllPath(t *testing.T) {
	operations, transport := newTagTestOperations(t, http.StatusNoContent, "")
	if err := operations.DeleteAll(context.Background()); err != nil {
		t.Fatalf("DeleteAll() error = %v", err)
	}
	assertTagRequest(t, transport, http.MethodDelete, "/v2.0/security-groups/resource-id/tags")
}

func assertTagRequest(t *testing.T, transport *recordingHTTPClient, method, escapedPath string) {
	t.Helper()
	if len(transport.requests) != 1 {
		t.Fatalf("request count = %d, want 1", len(transport.requests))
	}
	request := transport.requests[0]
	if request.Method != method || request.URL.EscapedPath() != escapedPath {
		t.Fatalf("request = %s %s, want %s %s",
			request.Method, request.URL.EscapedPath(), method, escapedPath)
	}
}

func TestTagsHasReportsMissingTag(t *testing.T) {
	operations, _ := newTagTestOperations(t, http.StatusNotFound, "")
	has, err := operations.Has(context.Background(), "missing")
	if err != nil {
		t.Fatalf("Has() error = %v", err)
	}
	if has {
		t.Fatal("Has() = true, want false")
	}
}

func TestTagsHasPropagatesOtherErrors(t *testing.T) {
	operations, _ := newTagTestOperations(t, http.StatusInternalServerError,
		`{"NeutronError":{"type":"ServiceUnavailable","message":"down"}}`)
	has, err := operations.Has(context.Background(), "one")
	if !IsErrorClass(err, ErrorClassServer) {
		t.Fatalf("Has() error = %v, want server class", err)
	}
	if has {
		t.Fatal("Has() = true on error, want false")
	}
}
