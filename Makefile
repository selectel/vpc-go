.PHONY: test cover lint fix check

# Coverage is measured across the whole module so helpers exercised only from
# other packages' tests are counted; test-only helpers in internal/testutil are left out.
COVER_PKGS = ./internal/api,./internal/utils,./pkg/...

test:
	go test ./...

cover:
	go test ./... -coverpkg=$(COVER_PKGS) -coverprofile=coverage.out
	go tool cover -func=coverage.out | tail -1

lint:
	golangci-lint run

fix:
	go fix ./...
	golangci-lint run --fix

check: test lint
