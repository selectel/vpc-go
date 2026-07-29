// Package firewallpolicy provides operations for firewall policy resources.
//
// Neutron resets a policy's audited value on every update that omits the
// audited attribute. Callers that need to preserve it must pass Audited
// explicitly; this package never reads or restores the previous value.
package firewallpolicy
