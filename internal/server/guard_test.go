package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Au1rxx/free-vpn-subscriptions/pkg/node"
)

func TestFilterNodes_DropsPrivateIPs(t *testing.T) {
	nodes := []*node.Node{
		{Name: "public", Protocol: node.ProtoTrojan, Server: "203.0.113.10", Port: 443},
		{Name: "rfc1918", Protocol: node.ProtoTrojan, Server: "192.168.1.1", Port: 443},
		{Name: "loopback", Protocol: node.ProtoTrojan, Server: "127.0.0.1", Port: 443},
		{Name: "metadata", Protocol: node.ProtoTrojan, Server: "169.254.169.254", Port: 80},
		{Name: "unspec", Protocol: node.ProtoTrojan, Server: "0.0.0.0", Port: 443},
		{Name: "hostname", Protocol: node.ProtoTrojan, Server: "example.com", Port: 443},
	}
	kept, dropped := filterNodes(nodes)
	keptNames := map[string]bool{}
	for _, n := range kept {
		keptNames[n.Name] = true
	}
	if !keptNames["public"] || !keptNames["hostname"] {
		t.Errorf("should keep public IP + hostname; kept=%v", keptNames)
	}
	for _, bad := range []string{"rfc1918", "loopback", "metadata", "unspec"} {
		if keptNames[bad] {
			t.Errorf("leaked %q past SSRF filter", bad)
		}
	}
	if len(dropped) != 4 {
		t.Errorf("expected 4 drop reasons, got %d: %v", len(dropped), dropped)
	}
}

func TestGuard_Authorize(t *testing.T) {
	g := &Guard{AuthToken: "secret"}

	noHeader := httptest.NewRequest("POST", "/", nil)
	if g.authorize(noHeader) {
		t.Error("unauth request passed")
	}

	wrong := httptest.NewRequest("POST", "/", nil)
	wrong.Header.Set("Authorization", "Bearer wrong")
	if g.authorize(wrong) {
		t.Error("wrong token passed")
	}

	right := httptest.NewRequest("POST", "/", nil)
	right.Header.Set("Authorization", "Bearer secret")
	if !g.authorize(right) {
		t.Error("correct token rejected")
	}

	basic := httptest.NewRequest("POST", "/", nil)
	basic.Header.Set("Authorization", "Basic xxx")
	if g.authorize(basic) {
		t.Error("Basic auth leaked past Bearer check")
	}

	// Unconfigured guard accepts everything.
	open := &Guard{}
	if !open.authorize(noHeader) {
		t.Error("open guard should accept")
	}
}

func TestRequireAuth_Rejects401(t *testing.T) {
	g := &Guard{AuthToken: "tok"}
	called := false
	h := g.requireAuth(func(w http.ResponseWriter, r *http.Request) { called = true })
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/x", nil)
	h(w, r)
	if w.Code != 401 {
		t.Errorf("code=%d (want 401)", w.Code)
	}
	if called {
		t.Error("handler should not be reached")
	}
	if !strings.Contains(w.Header().Get("WWW-Authenticate"), "Bearer") {
		t.Errorf("missing WWW-Authenticate header: %v", w.Header())
	}
}

func TestGuardedEndpoints_Unauth(t *testing.T) {
	g := &Guard{AuthToken: "k"}
	h := New("v", g)
	w := do(t, h, "POST", "/api/test", `proxies: []`)
	if w.Code != 401 {
		t.Errorf("/api/test unauth code=%d (want 401)", w.Code)
	}
	w = do(t, h, "POST", "/api/unlock", `proxies: []`)
	if w.Code != 401 {
		t.Errorf("/api/unlock unauth code=%d (want 401)", w.Code)
	}
}

func TestHandleTest_SSRFOnly_AllRejected(t *testing.T) {
	g := &Guard{}
	h := New("v", g)
	// All-private input → 400 "all blocked".
	body := `proxies:
  - name: priv
    type: trojan
    server: 10.0.0.1
    port: 443
    password: p
    sni: s
`
	w := do(t, h, "POST", "/api/test", body)
	if w.Code != 400 || !strings.Contains(w.Body.String(), "SSRF") {
		t.Errorf("expected 400 SSRF rejection, got code=%d body=%q", w.Code, w.Body.String())
	}
}

func TestNoGuard_HidesProbeEndpoints(t *testing.T) {
	// With guard=nil we never register POST /api/test or
	// POST /api/unlock. Go 1.22 mux falls through to the `GET /`
	// pattern whose path matches any "/…" — but method mismatch,
	// so it emits 405 (not 404). Either way the handler is not
	// reachable, which is the invariant we care about.
	h := New("v", nil)
	for _, path := range []string{"/api/test", "/api/unlock"} {
		w := do(t, h, "POST", path, "x")
		if w.Code != 404 && w.Code != 405 {
			t.Errorf("guard=nil %s should 404/405, got %d", path, w.Code)
		}
	}
}
