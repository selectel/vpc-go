package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"slices"
)

func Request(
	ctx context.Context,
	client *Client,
	method string,
	path string,
	query url.Values,
	body any,
	target any,
	expectedStatuses ...int,
) error {
	var payload io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return &ClientError{Err: err}
		}
		payload = bytes.NewReader(encoded)
	}

	response, err := client.do(ctx, method, path, query, payload)
	if err != nil {
		return err
	}
	defer func() { _ = response.Body.Close() }()

	rawBody, err := io.ReadAll(response.Body)
	if err != nil {
		return &TransportError{Err: err}
	}
	if !containsStatus(expectedStatuses, response.StatusCode) {
		return NewAPIError(response.StatusCode, rawBody)
	}
	if target == nil || len(rawBody) == 0 {
		return nil
	}
	if err := json.Unmarshal(rawBody, target); err != nil {
		return &UnexpectedResponseError{
			StatusCode: response.StatusCode, RawBody: append([]byte(nil), rawBody...), Err: err,
		}
	}

	return nil
}

func containsStatus(expected []int, actual int) bool {
	return slices.Contains(expected, actual)
}

func newRequestError(err error) error {
	if err == nil {
		return nil
	}

	return &ClientError{Err: fmt.Errorf("build request: %w", err)}
}
