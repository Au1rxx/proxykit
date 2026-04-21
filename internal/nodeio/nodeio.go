// Package nodeio is the boundary between proxykit CLI commands and the
// shared node data model defined in the aggregator repo's pkg/node.
// Wrapping the upstream package here keeps import paths local and gives us
// one place to adapt if the shared surface changes.
package nodeio

import (
	upstream "github.com/Au1rxx/free-vpn-subscriptions/pkg/node"
)

// Node is the in-memory proxy endpoint representation used by proxykit.
// It is an alias for the aggregator's canonical type, so no conversion is
// needed when passing between the two codebases.
type Node = upstream.Node

// ParseURI normalizes a single scheme:// URI into a Node.
func ParseURI(uri string) (*Node, error) {
	return upstream.ParseURI(uri)
}

// Supported protocols (re-exported for CLI help text and validation).
const (
	ProtoVLESS     = upstream.ProtoVLESS
	ProtoVMess     = upstream.ProtoVMess
	ProtoTrojan    = upstream.ProtoTrojan
	ProtoSS        = upstream.ProtoSS
	ProtoHysteria2 = upstream.ProtoHysteria2
)
