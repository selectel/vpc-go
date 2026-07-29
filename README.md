# vpc-go

`vpc-go` is a typed Go SDK for the Selectel Neutron network API. Its public
API is versioned by the network API generation and is imported from
`github.com/selectel/vpc-go/pkg/v2`.

## Access scope and authentication

The caller obtains a project-scoped Keystone token and the network endpoint
for the required region, then passes both to `v2.NewClient`. The client uses
only that endpoint and token. It does not authenticate with Keystone, select a
region, refresh credentials, keep process-global authentication state, or
silently retry unsafe requests. Every operation accepts `context.Context`.

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
operations live in packages split by Neutron collection:

- `network`, `subnet`, `subnetpool`, `port`, `router`, and `floatingip`;
- `addressscope` and `rbacpolicy`;
- `securitygroup`, including security-group rules;
- `firewallgroup`, `firewallpolicy`, and `firewallrule`.

A separate package is introduced when a resource has its own collection and
public model. Nested actions that only mutate their parent remain with the
parent package, such as router interfaces and firewall-policy rule actions.

`addressscope` and `rbacpolicy` operations are restricted by Selectel policy to
the project owner. The RBAC-policy collection does not provide pagination, so
its `List` operation performs one request rather than page traversal.

## Errors

Errors distinguish client-side construction or decoding failures,
transport failures, unexpected successful responses, incomplete list
traversal, and API failures. `vpc.IsErrorClass` classifies API failures such as
bad requests, authentication and authorization failures, not found, conflict,
quota exceeded, address unavailable, and server errors.

For resources exposing Selectel's `blocked` attribute, update and delete
operations can classify a readable blocked resource separately from an
ordinary 403 authorization failure. This diagnostic may require one read after
the failed write.

Important: a not-found class returned by update or delete may be a policy
failure masked by Neutron and does not prove that the resource is absent.
Inspect the preserved HTTP status, API error type, message, and raw body when
deciding whether a retry or reconciliation is safe.

## Versioning

Every public import path includes an explicit version segment such as
`pkg/v2`. An incompatible change to the public Go API is released as a new
versioned package `pkg/vN`; it is not made by breaking the existing package.
Module releases use SemVer tags.

## Regional extensions and intentional boundaries

Neutron extensions can differ between Selectel regions. The SDK sends the
requested operation and returns the region's API error; it does not maintain a
local availability table or reject an extension call in advance.

The SDK intentionally excludes:

- administrative operations;
- the bmnet device API;
- trunk resources;
- atomic router-route operations.

The module covers Neutron only. Selectel Resell API operations remain in
`go-selvpcclient`, which can be used alongside this module.
