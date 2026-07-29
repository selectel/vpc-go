package v2

import (
	"bytes"
	"encoding/json"
)

// Optional represents either an explicit value or an explicit JSON null.
//
// Use *Optional[T] in request DTOs with json:",omitempty": a nil pointer means
// that the field is unset and must not be sent.
type Optional[T any] struct {
	value T
	null  bool
}

// Value returns an explicit optional value.
func Value[T any](value T) *Optional[T] {
	return &Optional[T]{value: value}
}

// Null returns an explicit JSON null.
func Null[T any]() *Optional[T] {
	return &Optional[T]{null: true}
}

// Get returns the value and true when the optional contains a value.
func (optional Optional[T]) Get() (T, bool) {
	return optional.value, !optional.null
}

// IsNull reports whether the optional contains an explicit JSON null.
func (optional Optional[T]) IsNull() bool {
	return optional.null
}

// MarshalJSON encodes the explicit value or null.
func (optional Optional[T]) MarshalJSON() ([]byte, error) {
	if optional.null {
		return []byte("null"), nil
	}
	return json.Marshal(optional.value)
}

// UnmarshalJSON decodes an explicit value or null.
func (optional *Optional[T]) UnmarshalJSON(data []byte) error {
	if bytes.Equal(bytes.TrimSpace(data), []byte("null")) {
		var zero T
		optional.value = zero
		optional.null = true
		return nil
	}

	if err := json.Unmarshal(data, &optional.value); err != nil {
		return err
	}
	optional.null = false
	return nil
}
