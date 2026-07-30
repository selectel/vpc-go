// Package v2 provides the versioned public client contract for the Selectel
// Neutron network API.
//
// # Access scope
//
// Callers obtain a project-scoped Keystone token and a regional network
// endpoint and pass them to NewClient. Client does not authenticate, refresh
// credentials, select a region, keep mutable process-global authentication
// state, or retry unsafe requests. Every resource operation accepts a
// context.Context, and callers may provide their own HTTP transport.
//
// # Resource packages
//
// The v2 package owns the client, selection and pagination options, optional
// request values, common errors, and tag primitives. Packages network, subnet,
// subnetpool, port, router, floatingip, addressscope, rbacpolicy,
// securitygroup, firewallgroup, firewallpolicy, and firewallrule are divided
// by Neutron resource collection. Nested actions that only mutate a parent,
// such as router interfaces, remain in that parent's package.
//
// Selectel restricts addressscope and rbacpolicy operations to the project
// owner. The RBAC-policy collection does not support pagination. Extension
// availability can differ by region; the SDK sends the request and preserves
// the resulting API error rather than maintaining a local availability table.
//
// # Errors
//
// ClientError, TransportError, UnexpectedResponseError, IncompleteListError,
// and APIError distinguish failure sources. IsErrorClass classifies API
// failures, including quota and address allocation failures that share an HTTP
// status with ordinary conflicts. PolicyNotAuthorized remains a general policy
// failure; the SDK does not issue a diagnostic read to infer whether a
// resource is blocked.
//
// A not-found class returned by update or delete may be a Neutron policy
// failure masked as absence. It does not prove the resource is absent. APIError
// preserves the actual status, type, message, and raw response.
//
// # Versioning and boundaries
//
// Public imports always include a version segment such as pkg/v2. An
// incompatible public Go API change is released in a new versioned package
// pkg/vN instead of breaking an existing package.
//
// This SDK intentionally excludes administrative operations, the bmnet device
// API, trunk resources, and atomic router-route actions. It covers Neutron,
// not the Selectel Resell API.
package v2
