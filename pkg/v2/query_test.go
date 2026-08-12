package v2

import (
	"reflect"
	"testing"
)

func TestQueryValues(t *testing.T) {
	options := ListOptions{
		SelectionOptions: SelectionOptions{
			Filters: map[string][]string{
				"status": {"ACTIVE", "DOWN"},
			},
			Fields:  []string{"id", "name"},
			SortKey: "name",
			SortDir: "asc",
		},
		Limit:  50,
		Marker: "last-id",
	}

	values := options.Values()
	if got := values["status"]; !reflect.DeepEqual(got, []string{"ACTIVE", "DOWN"}) {
		t.Fatalf("status values = %#v", got)
	}
	if got := values["fields"]; !reflect.DeepEqual(got, []string{"id", "name"}) {
		t.Fatalf("fields values = %#v", got)
	}
	if values.Get("sort_key") != "name" ||
		values.Get("sort_dir") != "asc" ||
		values.Get("limit") != "50" ||
		values.Get("marker") != "last-id" {
		t.Fatalf("encoded values = %v", values)
	}
}
