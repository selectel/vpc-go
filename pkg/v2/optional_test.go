package v2

import (
	"encoding/json"
	"testing"
)

func TestOptionalThreeStates(t *testing.T) {
	type request struct {
		Gateway *Optional[string] `json:"gateway_ip,omitempty"`
	}

	tests := []struct {
		name string
		dto  request
		want string
	}{
		{"unset", request{}, `{}`},
		{"value", request{Gateway: Value("192.0.2.1")}, `{"gateway_ip":"192.0.2.1"}`},
		{"null", request{Gateway: Null[string]()}, `{"gateway_ip":null}`},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			data, err := json.Marshal(test.dto)
			if err != nil {
				t.Fatalf("json.Marshal() error = %v", err)
			}
			if string(data) != test.want {
				t.Fatalf("JSON = %s, want %s", data, test.want)
			}
		})
	}
}
