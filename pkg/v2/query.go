package v2

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strconv"
)

// SelectionOptions contains query parameters shared by all list operations.
type SelectionOptions struct {
	Filters map[string][]string
	Fields  []string
	SortKey string
	SortDir string
}

// Values encodes selection parameters using repeated keys for repeated values.
func (options SelectionOptions) Values() url.Values {
	values := make(url.Values)
	for key, items := range options.Filters {
		for _, item := range items {
			values.Add(key, item)
		}
	}
	for _, field := range options.Fields {
		values.Add("fields", field)
	}
	if options.SortKey != "" {
		values.Set("sort_key", options.SortKey)
	}
	if options.SortDir != "" {
		values.Set("sort_dir", options.SortDir)
	}

	return values
}

// ListOptions adds pagination parameters to SelectionOptions.
type ListOptions struct {
	SelectionOptions
	Limit  int
	Marker string
}

// Values encodes selection and pagination parameters.
func (options ListOptions) Values() url.Values {
	values := options.SelectionOptions.Values()
	if options.Limit > 0 {
		values.Set("limit", strconv.Itoa(options.Limit))
	}
	if options.Marker != "" {
		values.Set("marker", options.Marker)
	}

	return values
}

// Page is one decoded API collection response.
type Page[T any] struct {
	Items    []T
	NextLink string
}

// PageFetcher reads one page using query parameters for a fixed resource path.
type PageFetcher[T any] func(context.Context, url.Values) (Page[T], error)

// IncompleteListError reports that at least one page was read but the complete
// collection could not be obtained.
type IncompleteListError struct {
	Err error
}

func (err *IncompleteListError) Error() string {
	return "incomplete vpc API list: " + err.Err.Error()
}

func (err *IncompleteListError) Unwrap() error {
	return err.Err
}

// WalkPages follows next links until the complete collection is available.
//
// Only query parameters are taken from a next link. Its scheme, host, and path
// are deliberately ignored, so PageFetcher continues to use its fixed endpoint
// and resource path.
func WalkPages[T any](
	ctx context.Context,
	initial url.Values,
	fetch PageFetcher[T],
) ([]T, error) {
	query := cloneValues(initial)
	var result []T

	for pageNumber := 0; ; pageNumber++ {
		page, err := fetch(ctx, query)
		if err != nil {
			if pageNumber == 0 {
				return nil, err
			}
			return nil, &IncompleteListError{Err: err}
		}
		result = append(result, page.Items...)

		if page.NextLink == "" {
			return result, nil
		}

		nextURL, err := url.Parse(page.NextLink)
		if err != nil {
			return nil, &IncompleteListError{
				Err: fmt.Errorf("parse next page link: %w", err),
			}
		}
		query = nextURL.Query()
	}
}

// IsIncompleteList reports whether list traversal stopped after a partial read.
func IsIncompleteList(err error) bool {
	var incompleteErr *IncompleteListError
	return errors.As(err, &incompleteErr)
}

func cloneValues(values url.Values) url.Values {
	cloned := make(url.Values, len(values))
	for key, items := range values {
		cloned[key] = append([]string(nil), items...)
	}
	return cloned
}
