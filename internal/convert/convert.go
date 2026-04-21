// Package convert is the format-dispatch layer behind `proxykit convert`.
// It wraps pkg/parse + pkg/emit from the aggregator module and adds a
// format auto-detector so callers don't have to specify --from for common
// inputs.
package convert

import (
	"fmt"
	"strings"

	"github.com/Au1rxx/free-vpn-subscriptions/pkg/emit"
	"github.com/Au1rxx/free-vpn-subscriptions/pkg/node"
	"github.com/Au1rxx/free-vpn-subscriptions/pkg/parse"
)

// Decode parses an input body in the given format into normalized Nodes.
// format may be "auto" for heuristic detection.
func Decode(body []byte, format string) ([]*node.Node, error) {
	if format == "" || format == "auto" {
		format = Detect(body)
	}
	switch format {
	case "clash":
		return parse.Clash(body)
	case "base64":
		return parse.Base64List(body)
	case "uri-list":
		return parse.URIList(string(body)), nil
	default:
		return nil, fmt.Errorf("unknown input format %q (want auto|clash|uri-list|base64)", format)
	}
}

// Encode renders Nodes into the requested output format.
// surge/quanx/loon are partial-coverage formats: VLESS and Hysteria2 nodes
// are dropped silently because those clients have no native mapping.
func Encode(nodes []*node.Node, format string) (string, error) {
	switch format {
	case "clash":
		return emit.Clash(nodes)
	case "singbox":
		return emit.Singbox(nodes)
	case "v2ray":
		return emit.V2RayBase64(nodes), nil
	case "surge":
		return emit.Surge(nodes)
	case "quanx":
		return emit.QuantumultX(nodes)
	case "loon":
		return emit.Loon(nodes)
	default:
		return "", fmt.Errorf("unknown output format %q (want clash|singbox|v2ray|surge|quanx|loon)", format)
	}
}

// Detect picks an input format from a raw body. The heuristic is
// intentionally simple: YAML takes priority, then URI scheme, then base64.
func Detect(body []byte) string {
	trimmed := strings.TrimSpace(string(body))
	if trimmed == "" {
		return "uri-list"
	}
	// Clash YAML almost always has a top-level `proxies:` key.
	if strings.Contains(trimmed, "\nproxies:") || strings.HasPrefix(trimmed, "proxies:") {
		return "clash"
	}
	// A body that starts with a known URI scheme is a URI list.
	for _, scheme := range []string{"vless://", "vmess://", "trojan://", "ss://", "hy2://", "hysteria2://"} {
		if strings.HasPrefix(trimmed, scheme) {
			return "uri-list"
		}
	}
	// Pure base64 blob — no spaces, no newlines except trailing, charset fits.
	if looksBase64(trimmed) {
		return "base64"
	}
	return "uri-list"
}

func looksBase64(s string) bool {
	if len(s) < 16 {
		return false
	}
	for _, r := range s {
		if (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') ||
			(r >= '0' && r <= '9') || r == '+' || r == '/' || r == '=' ||
			r == '-' || r == '_' || r == '\n' || r == '\r' {
			continue
		}
		return false
	}
	return true
}
