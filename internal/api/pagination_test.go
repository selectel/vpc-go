package api

import (
	"context"
	"errors"
	"net/url"
	"reflect"
	"testing"
)

func TestWalkPages(t *testing.T) {
	queries := make([]url.Values, 0, 2)
	pages := []Page[string]{
		{
			Items:    []string{"one"},
			NextLink: "https://ignored.example/other/path?marker=one&limit=1",
		},
		{Items: []string{"two"}},
	}

	items, err := WalkPages(
		context.Background(),
		url.Values{"limit": {"1"}},
		func(_ context.Context, query url.Values) (Page[string], error) {
			queries = append(queries, cloneValues(query))
			return pages[len(queries)-1], nil
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(items, []string{"one", "two"}) {
		t.Fatalf("items=%v", items)
	}
	if queries[1].Get("marker") != "one" || queries[1].Get("limit") != "1" {
		t.Fatalf("query=%v", queries[1])
	}
}

func TestWalkPagesReturnsFetchError(t *testing.T) {
	wantErr := errors.New("connection lost")
	_, err := WalkPages(
		context.Background(),
		nil,
		func(context.Context, url.Values) (Page[string], error) {
			return Page[string]{}, wantErr
		},
	)
	if !errors.Is(err, wantErr) {
		t.Fatalf("error=%v, want %v", err, wantErr)
	}
}
