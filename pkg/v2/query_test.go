package v2

import (
	"context"
	"errors"
	"net/url"
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

func TestPagingWalksAllPagesUsingOnlyNextQuery(t *testing.T) {
	queries := make([]url.Values, 0, 2)
	pages := []Page[string]{
		{
			Items:    []string{"one"},
			NextLink: "https://untrusted.example.test/other/path?marker=one&limit=1",
		},
		{Items: []string{"two"}},
	}

	items, err := WalkPages(
		context.Background(),
		url.Values{"limit": {"1"}},
		func(_ context.Context, query url.Values) (Page[string], error) {
			queries = append(queries, cloneValues(query))
			page := pages[len(queries)-1]
			return page, nil
		},
	)
	if err != nil {
		t.Fatalf("WalkPages() error = %v", err)
	}
	if !reflect.DeepEqual(items, []string{"one", "two"}) {
		t.Fatalf("items = %#v", items)
	}
	if len(queries) != 2 {
		t.Fatalf("query count = %d, want 2", len(queries))
	}
	if queries[1].Get("marker") != "one" || queries[1].Get("limit") != "1" {
		t.Fatalf("next query = %v", queries[1])
	}
}

func TestPagingFailureDoesNotReturnPartialItems(t *testing.T) {
	pageCalls := 0
	connectionErr := errors.New("connection lost")

	items, err := WalkPages(
		context.Background(),
		nil,
		func(context.Context, url.Values) (Page[string], error) {
			pageCalls++
			if pageCalls == 1 {
				return Page[string]{
					Items:    []string{"partial"},
					NextLink: "?marker=partial",
				}, nil
			}
			return Page[string]{}, connectionErr
		},
	)
	if items != nil {
		t.Fatalf("items = %#v, want nil", items)
	}
	if !IsIncompleteList(err) || !errors.Is(err, connectionErr) {
		t.Fatalf("WalkPages() error = %v, want incomplete list", err)
	}
}

func TestPagingResponseWithoutNextIsComplete(t *testing.T) {
	items, err := WalkPages(
		context.Background(),
		nil,
		func(context.Context, url.Values) (Page[string], error) {
			return Page[string]{Items: []string{"only"}}, nil
		},
	)
	if err != nil {
		t.Fatalf("WalkPages() error = %v", err)
	}
	if !reflect.DeepEqual(items, []string{"only"}) {
		t.Fatalf("items = %#v", items)
	}
}
