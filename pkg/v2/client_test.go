package v2

import (
	"net/http"
	"testing"
)

func TestNewClientValidatesConfig(t *testing.T) {
	if _, err := NewClient(Config{Endpoint: "https://network.example.test", Token: "token"}); err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	if _, err := NewClient(Config{Token: "token"}); err == nil {
		t.Fatal("NewClient() without endpoint returned nil error")
	}
}

func TestIsErrorClassRecognisesAPIError(t *testing.T) {
	err := &APIError{StatusCode: http.StatusNotFound, Class: ErrorClassNotFound}
	if !IsErrorClass(err, ErrorClassNotFound) {
		t.Fatal("IsErrorClass() = false for matching class")
	}
	if IsErrorClass(err, ErrorClassConflict) {
		t.Fatal("IsErrorClass() = true for different class")
	}
}
