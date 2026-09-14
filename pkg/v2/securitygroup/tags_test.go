package securitygroup

import (
	"context"
	"net/http"
	"testing"

	"github.com/selectel/vpc-go/internal/testutil"
)

func TestTagOperationsUseResourcePath(t *testing.T) {
	client, transport := testutil.NewClient(t, testutil.Response(http.StatusOK, `{"tags":["one"]}`))
	tags, err := TagOperations(client, "id").Get(context.Background())
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if len(tags) != 1 || tags[0] != "one" {
		t.Fatalf("Get() = %v, want [one]", tags)
	}
	testutil.AssertRequest(t, transport.Requests[0], http.MethodGet, "/v2.0/security-groups/id/tags")
}
