package server

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"

	"github.com/Au1rxx/free-vpn-subscriptions/pkg/node"
)

// Guard bundles the resource + threat-surface limits shared by the
// test/unlock endpoints (the two endpoints that spawn processes or
// hit the network outwards).
type Guard struct {
	// AuthToken, if non-empty, requires Authorization: Bearer <token>
	// on test/unlock requests. Convert + health are always unauthed.
	AuthToken string
	// MaxNodesTest caps how many nodes a single /api/test request may
	// probe. Zero → 50.
	MaxNodesTest int
	// MaxNodesUnlock caps /api/unlock. Zero → 10 (each node spawns a
	// sing-box subprocess, so this is intentionally tight).
	MaxNodesUnlock int
	// Parallel is the global concurrent-requests semaphore for the
	// heavy endpoints. Zero → 2. `sem` is lazily created.
	Parallel int
	sem      chan struct{}
}

func (g *Guard) init() {
	if g.MaxNodesTest == 0 {
		g.MaxNodesTest = 50
	}
	if g.MaxNodesUnlock == 0 {
		g.MaxNodesUnlock = 10
	}
	if g.Parallel == 0 {
		g.Parallel = 2
	}
	if g.sem == nil {
		g.sem = make(chan struct{}, g.Parallel)
	}
}

// acquire blocks until a concurrency slot is free or ctx is cancelled.
// Release by calling the returned func (always non-nil).
func (g *Guard) acquire(ctx context.Context) (func(), error) {
	g.init()
	select {
	case g.sem <- struct{}{}:
		return func() { <-g.sem }, nil
	case <-ctx.Done():
		return func() {}, ctx.Err()
	}
}

// authorize returns true if the request passes the (optional) bearer
// check. When no token is configured, every request passes.
func (g *Guard) authorize(r *http.Request) bool {
	if g.AuthToken == "" {
		return true
	}
	h := r.Header.Get("Authorization")
	const prefix = "Bearer "
	if !strings.HasPrefix(h, prefix) {
		return false
	}
	// Constant-time compare would be nicer; tokens are server-op
	// secrets so a timing side-channel here is low-value.
	return strings.TrimSpace(h[len(prefix):]) == g.AuthToken
}

// filterNodes strips nodes whose declared server address is an obvious
// SSRF target (RFC 1918, loopback, link-local, IPv6 ULA). Hostnames are
// passed through — we do not do DNS resolution here to avoid adding
// latency and a DNS-rebinding side channel. Operators exposing this
// server to untrusted clients SHOULD front it with a network policy
// that blocks outbound private-range traffic.
//
// Returns (kept, dropped) where dropped is a list of human-readable
// reasons like "n1: 192.168.1.1 is RFC1918".
func filterNodes(nodes []*node.Node) ([]*node.Node, []string) {
	kept := make([]*node.Node, 0, len(nodes))
	var dropped []string
	for _, n := range nodes {
		if ip := net.ParseIP(n.Server); ip != nil {
			if reason := blockedReason(ip); reason != "" {
				dropped = append(dropped, fmt.Sprintf("%s: %s %s", n.Name, n.Server, reason))
				continue
			}
		}
		kept = append(kept, n)
	}
	return kept, dropped
}

func blockedReason(ip net.IP) string {
	if ip.IsLoopback() {
		return "loopback"
	}
	if ip.IsPrivate() {
		return "RFC1918/ULA private"
	}
	if ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
		return "link-local"
	}
	if ip.IsUnspecified() {
		return "unspecified (0.0.0.0 / ::)"
	}
	// Explicitly block the AWS/GCP/Azure metadata IP even though some
	// of these checks may already cover it.
	if ip.Equal(net.ParseIP("169.254.169.254")) {
		return "cloud metadata endpoint"
	}
	return ""
}

// requireAuth is a middleware wrapping guarded endpoints.
func (g *Guard) requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !g.authorize(r) {
			w.Header().Set("WWW-Authenticate", "Bearer realm=\"proxykit\"")
			httpError(w, http.StatusUnauthorized, "missing or bad bearer token")
			return
		}
		next(w, r)
	}
}
