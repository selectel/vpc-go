package testutil

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	vpc "github.com/selectel/vpc-go/pkg/v2"
)

type Transport struct {
	Requests  []*http.Request
	Responses []*http.Response
	Errors    []error
}

func (transport *Transport) Do(request *http.Request) (*http.Response, error) {
	transport.Requests = append(transport.Requests, request)
	index := len(transport.Requests) - 1
	if index < len(transport.Errors) && transport.Errors[index] != nil {
		return nil, transport.Errors[index]
	}
	if index >= len(transport.Responses) {
		return nil, errors.New("test transport has no scripted response")
	}
	return transport.Responses[index], nil
}

func NewClient(t testing.TB, responses ...*http.Response) (*vpc.Client, *Transport) {
	t.Helper()
	transport := &Transport{Responses: responses}
	client, err := vpc.NewClient(vpc.Config{
		Endpoint:   "https://network.example.test",
		Token:      "token",
		HTTPClient: transport,
	})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	return client, transport
}

func Response(status int, body string) *http.Response {
	return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader(body))}
}

func AssertJSONBody(t testing.TB, request *http.Request, want string) {
	t.Helper()
	if request.Body == nil {
		t.Fatal("request body is nil")
	}
	data, err := io.ReadAll(request.Body)
	if err != nil {
		t.Fatalf("read request body: %v", err)
	}

	var gotValue any
	var wantValue any
	if err := json.Unmarshal(data, &gotValue); err != nil {
		t.Fatalf("decode request body %q: %v", data, err)
	}
	if err := json.Unmarshal([]byte(want), &wantValue); err != nil {
		t.Fatalf("decode expected JSON %q: %v", want, err)
	}
	if !reflect.DeepEqual(gotValue, wantValue) {
		t.Errorf("request body = %s, want %s", data, want)
	}
}

func AssertRequest(t testing.TB, request *http.Request, method, path string) {
	t.Helper()
	if request.Method != method || request.URL.Path != path {
		t.Errorf("request = %s %s, want %s %s", request.Method, request.URL.Path, method, path)
	}
}
