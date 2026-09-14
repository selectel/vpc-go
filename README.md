# vpc-go

`vpc-go` is a typed Go SDK for the Selectel VPC API, compatible with the
OpenStack Neutron API.
Its public API is versioned by the network API generation and is imported from
`github.com/selectel/vpc-go/pkg/v2`.

## Access scope and authentication

The caller passes a project-scoped Keystone token and the VPC endpoint for the
required region to `v2.NewClient`. Every operation accepts `context.Context`.

```go
client, err := vpc.NewClient(vpc.Config{
    Endpoint:  networkEndpoint,
    Token:     projectToken,
    UserAgent: "my-service/1.0",
})
```

An application can supply its own HTTP client in `Config.HTTPClient`.

## Packages

The root `pkg/v2` package contains the client, selection and pagination
options, optional request values, tag primitives, and common errors. Resource
operations live in packages split by collection.

## Errors

Errors distinguish client-side construction or decoding failures,
transport failures, unexpected successful responses, incomplete list
traversal, and API failures. `vpc.IsErrorClass` classifies API failures such as
bad requests, authentication and authorization failures, not found, conflict,
quota exceeded, address unavailable, and server errors.

An authorization failure can be returned by the VPC API as not found. Inspect
the preserved HTTP status, API error type, message, and raw body when deciding
whether a retry or reconciliation is safe.

## Versioning

Every public import path includes an explicit version segment such as
`pkg/v2`. An incompatible change to the public Go API is released as a new
versioned package `pkg/vN`; it is not made by breaking the existing package.
Module releases use SemVer tags.

## Development

Run the tests:

```sh
make test
```

Run the tests with a coverage report (also what CI runs):

```sh
make cover
```

Run the linters configured in `.golangci.yml` with:

```sh
make lint
```

Apply automatic fixes (`go fix` and linter auto-fixes, including `modernize`) with:

```sh
make fix
```

Run all local checks with:

```sh
make check
```
