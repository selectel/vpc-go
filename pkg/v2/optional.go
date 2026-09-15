package v2

import "encoding/json"

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

// MarshalJSON encodes the explicit value or null.
func (optional Optional[T]) MarshalJSON() ([]byte, error) {
	if optional.null {
		return []byte("null"), nil
	}

	return json.Marshal(optional.value)
}
