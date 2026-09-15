// Package v2 provides the versioned Go client for the Selectel VPC API,
// compatible with the OpenStack Neutron API.
//
// # Client
//
// NewClient accepts a regional VPC endpoint, a project-scoped Keystone token,
// and an optional HTTP client. Every resource operation accepts a
// context.Context.
//
// # Resource packages
//
// The v2 package contains the client, selection and pagination options,
// optional request values, common errors, and tag primitives. Resource
// operations are grouped into packages by resource type.
//
// # Errors
//
// ClientError, TransportError, UnexpectedResponseError, and APIError
// distinguish failure sources. IsErrorClass classifies API failures. APIError
// preserves the HTTP status, API error type, message, and raw response.
//
// # Versioning
//
// Public imports include a version segment such as pkg/v2. Incompatible public
// Go API changes are released in a new versioned package.
package v2
