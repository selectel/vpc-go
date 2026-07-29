package v2

import (
	"encoding/json"
	"reflect"
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

func TestOptionalSliceUnsetAndClear(t *testing.T) {
	type request struct {
		Routes *[]string `json:"routes,omitempty"`
	}
	empty := []string{}

	unset, err := json.Marshal(request{})
	if err != nil {
		t.Fatalf("json.Marshal(unset) error = %v", err)
	}
	cleared, err := json.Marshal(request{Routes: &empty})
	if err != nil {
		t.Fatalf("json.Marshal(cleared) error = %v", err)
	}

	if string(unset) != `{}` {
		t.Fatalf("unset JSON = %s", unset)
	}
	if string(cleared) != `{"routes":[]}` {
		t.Fatalf("cleared JSON = %s", cleared)
	}
}

func TestDecodeIgnoresUnknownFields(t *testing.T) {
	type model struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}

	var got model
	err := json.Unmarshal(
		[]byte(`{"id":"id","name":"name","future_attribute":{"nested":true}}`),
		&got,
	)
	if err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if !reflect.DeepEqual(got, model{ID: "id", Name: "name"}) {
		t.Fatalf("model = %+v", got)
	}
}
