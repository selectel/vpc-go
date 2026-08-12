package api

import (
	"context"
	"net/http"
	"net/url"
)

type tagsEnvelope struct {
	Tags []string `json:"tags"`
}

type TagOperations struct {
	client     *Client
	collection string
	resourceID string
}

func NewTagOperations(client *Client, collection, resourceID string) TagOperations {
	return TagOperations{client: client, collection: collection, resourceID: resourceID}
}

func (operations TagOperations) Get(ctx context.Context) ([]string, error) {
	var envelope tagsEnvelope
	err := Request(ctx, operations.client, http.MethodGet, operations.path(), nil, nil,
		&envelope, http.StatusOK)
	if err != nil {
		return nil, err
	}

	return envelope.Tags, nil
}

func (operations TagOperations) Has(ctx context.Context, tag string) (bool, error) {
	err := Request(ctx, operations.client, http.MethodGet, operations.tagPath(tag), nil, nil,
		nil, http.StatusNoContent)
	if err == nil {
		return true, nil
	}
	if IsErrorClass(err, ErrorClassNotFound) {
		return false, nil
	}

	return false, err
}

func (operations TagOperations) Add(ctx context.Context, tag string) error {
	return Request(ctx, operations.client, http.MethodPut, operations.tagPath(tag), nil, nil,
		nil, http.StatusCreated, http.StatusNoContent)
}

func (operations TagOperations) Delete(ctx context.Context, tag string) error {
	return Request(ctx, operations.client, http.MethodDelete, operations.tagPath(tag), nil, nil,
		nil, http.StatusNoContent)
}

func (operations TagOperations) Replace(ctx context.Context, tags []string) ([]string, error) {
	var envelope tagsEnvelope
	err := Request(ctx, operations.client, http.MethodPut, operations.path(), nil,
		tagsEnvelope{Tags: tags}, &envelope, http.StatusOK)
	if err != nil {
		return nil, err
	}

	return envelope.Tags, nil
}

func (operations TagOperations) DeleteAll(ctx context.Context) error {
	return Request(ctx, operations.client, http.MethodDelete, operations.path(), nil, nil,
		nil, http.StatusNoContent)
}

func (operations TagOperations) path() string {
	return "/v2.0/" + url.PathEscape(operations.collection) + "/" +
		url.PathEscape(operations.resourceID) + "/tags"
}

func (operations TagOperations) tagPath(tag string) string {
	return operations.path() + "/" + url.PathEscape(tag)
}
