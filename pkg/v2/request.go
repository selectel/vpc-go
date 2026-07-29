package v2

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
)

// RequestOptions controls response handling for a JSON API operation.
type RequestOptions struct {
	ExpectedStatus []int
	// ReadBlocked performs one diagnostic read after a 403 response. It returns
	// true only when the affected resource is readable and currently blocked.
	ReadBlocked func(context.Context) (bool, error)
}

// Request sends one JSON request and decodes one JSON response.
func (client *Client) Request(
	ctx context.Context,
	method string,
	path string,
	query url.Values,
	body any,
	target any,
	options RequestOptions,
) error {
	var payload io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return &ClientError{Err: err}
		}
		payload = bytes.NewReader(encoded)
	}

	response, err := client.Do(ctx, method, path, query, payload)
	if err != nil {
		return err
	}
	defer func() {
		_ = response.Body.Close()
	}()

	rawBody, err := io.ReadAll(response.Body)
	if err != nil {
		return &TransportError{Err: err}
	}

	if !containsStatus(options.ExpectedStatus, response.StatusCode) {
		apiErr := newAPIError(response.StatusCode, rawBody)
		if apiErr.Class == ErrorClassForbidden && options.ReadBlocked != nil {
			blocked, _ := options.ReadBlocked(ctx)
			if blocked {
				apiErr.Class = ErrorClassResourceBlocked
			}
		}
		return apiErr
	}

	if target == nil || len(rawBody) == 0 {
		return nil
	}
	if err := json.Unmarshal(rawBody, target); err != nil {
		return &UnexpectedResponseError{
			StatusCode: response.StatusCode,
			RawBody:    append([]byte(nil), rawBody...),
			Err:        err,
		}
	}

	return nil
}

func containsStatus(expected []int, actual int) bool {
	for _, status := range expected {
		if status == actual {
			return true
		}
	}
	return false
}

func newRequestError(err error) error {
	if err == nil {
		return nil
	}
	return &ClientError{Err: fmt.Errorf("build request: %w", err)}
}
