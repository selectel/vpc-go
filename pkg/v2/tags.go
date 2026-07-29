package v2

import (
	"context"
	"net/http"
	"net/url"
)

type tagsEnvelope struct {
	Tags []string `json:"tags"`
}

// TagOperations implements the common Neutron tags API for one resource.
type TagOperations struct {
	client      *Client
	collection  string
	resourceID  string
	readBlocked func(context.Context) (bool, error)
}

// NewTagOperations binds tag operations to one resource.
func NewTagOperations(
	client *Client,
	collection string,
	resourceID string,
	readBlocked func(context.Context) (bool, error),
) TagOperations {
	return TagOperations{
		client:      client,
		collection:  collection,
		resourceID:  resourceID,
		readBlocked: readBlocked,
	}
}

// Get returns the complete tag set.
func (operations TagOperations) Get(ctx context.Context) ([]string, error) {
	var envelope tagsEnvelope
	err := operations.client.Request(
		ctx,
		http.MethodGet,
		operations.path(),
		nil,
		nil,
		&envelope,
		RequestOptions{ExpectedStatus: []int{http.StatusOK}},
	)
	if err != nil {
		return nil, err
	}
	return envelope.Tags, nil
}

// Has reports whether one tag exists.
func (operations TagOperations) Has(ctx context.Context, tag string) (bool, error) {
	err := operations.client.Request(
		ctx,
		http.MethodGet,
		operations.tagPath(tag),
		nil,
		nil,
		nil,
		RequestOptions{ExpectedStatus: []int{http.StatusNoContent}},
	)
	if err == nil {
		return true, nil
	}
	if IsErrorClass(err, ErrorClassNotFound) {
		return false, nil
	}
	return false, err
}

// Add adds one tag.
func (operations TagOperations) Add(ctx context.Context, tag string) error {
	return operations.client.Request(
		ctx,
		http.MethodPut,
		operations.tagPath(tag),
		nil,
		nil,
		nil,
		operations.writeOptions(http.StatusCreated, http.StatusNoContent),
	)
}

// Delete deletes one tag.
func (operations TagOperations) Delete(ctx context.Context, tag string) error {
	return operations.client.Request(
		ctx,
		http.MethodDelete,
		operations.tagPath(tag),
		nil,
		nil,
		nil,
		operations.writeOptions(http.StatusNoContent),
	)
}

// Replace replaces the complete tag set with one request.
func (operations TagOperations) Replace(ctx context.Context, tags []string) error {
	var envelope tagsEnvelope
	err := operations.client.Request(
		ctx,
		http.MethodPut,
		operations.path(),
		nil,
		tagsEnvelope{Tags: tags},
		&envelope,
		operations.writeOptions(http.StatusOK),
	)
	return err
}

// DeleteAll deletes the complete tag set.
func (operations TagOperations) DeleteAll(ctx context.Context) error {
	return operations.client.Request(
		ctx,
		http.MethodDelete,
		operations.path(),
		nil,
		nil,
		nil,
		operations.writeOptions(http.StatusNoContent),
	)
}

func (operations TagOperations) path() string {
	return "/v2.0/" + url.PathEscape(operations.collection) + "/" +
		url.PathEscape(operations.resourceID) + "/tags"
}

func (operations TagOperations) tagPath(tag string) string {
	return operations.path() + "/" + url.PathEscape(tag)
}

func (operations TagOperations) writeOptions(expectedStatus ...int) RequestOptions {
	return RequestOptions{
		ExpectedStatus: expectedStatus,
		ReadBlocked:    operations.readBlocked,
	}
}

// IsTagAbsent reports the successful negative result returned by Has. It is
// retained for callers that need to distinguish that result from API errors.
func IsTagAbsent(present bool, err error) bool {
	return !present && err == nil
}
