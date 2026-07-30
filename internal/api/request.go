package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"reflect"
	"strings"
)

type RequestOptions struct {
	ExpectedStatus []int
}

func Request(
	ctx context.Context,
	client *Client,
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

	response, err := client.do(ctx, method, path, query, payload)
	if err != nil {
		return err
	}
	defer func() { _ = response.Body.Close() }()

	rawBody, err := io.ReadAll(response.Body)
	if err != nil {
		return &TransportError{Err: err}
	}
	if !containsStatus(options.ExpectedStatus, response.StatusCode) {
		return NewAPIError(response.StatusCode, rawBody)
	}
	if target == nil || len(rawBody) == 0 {
		return nil
	}
	if err := validateEnvelope(rawBody, target); err != nil {
		return &UnexpectedResponseError{
			StatusCode: response.StatusCode, RawBody: append([]byte(nil), rawBody...), Err: err,
		}
	}
	if err := json.Unmarshal(rawBody, target); err != nil {
		return &UnexpectedResponseError{
			StatusCode: response.StatusCode, RawBody: append([]byte(nil), rawBody...), Err: err,
		}
	}

	return nil
}

func validateEnvelope(rawBody []byte, target any) error {
	targetType := reflect.TypeOf(target)
	if targetType == nil || targetType.Kind() != reflect.Pointer {
		return nil
	}
	targetType = targetType.Elem()
	if targetType.Kind() != reflect.Struct || targetType.NumField() != 1 {
		return nil
	}
	field := targetType.Field(0)
	key := strings.Split(field.Tag.Get("json"), ",")[0]
	if key == "" || key == "-" {
		return nil
	}
	var object map[string]json.RawMessage
	if err := json.Unmarshal(rawBody, &object); err != nil {
		return nil
	}
	value, ok := object[key]
	if !ok || string(value) == "null" {
		return fmt.Errorf("response envelope must contain non-null %q", key)
	}
	if field.Type.Kind() == reflect.Struct && string(value) == "{}" {
		return fmt.Errorf("response envelope %q must contain a resource", key)
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
