package server

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func do(t *testing.T, h http.Handler, method, target, body string) *httptest.ResponseRecorder {
	t.Helper()
	var r *http.Request
	if body == "" {
		r = httptest.NewRequest(method, target, nil)
	} else {
		r = httptest.NewRequest(method, target, strings.NewReader(body))
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}

func TestHealthAndVersion(t *testing.T) {
	h := New("t-test")
	if w := do(t, h, "GET", "/health", ""); w.Code != 200 || w.Body.String() != "ok" {
		t.Errorf("/health code=%d body=%q", w.Code, w.Body.String())
	}
	if w := do(t, h, "GET", "/version", ""); w.Code != 200 || !strings.Contains(w.Body.String(), "t-test") {
		t.Errorf("/version code=%d body=%q", w.Code, w.Body.String())
	}
}

func TestIndex(t *testing.T) {
	h := New("v0")
	w := do(t, h, "GET", "/", "")
	if w.Code != 200 {
		t.Fatalf("code=%d", w.Code)
	}
	b := w.Body.String()
	if !strings.Contains(b, "<title>proxykit") || !strings.Contains(b, "/api/convert") {
		t.Errorf("index HTML missing sentinel strings: %q", b[:min(200, len(b))])
	}
}

func TestConvert_ClashToSingbox(t *testing.T) {
	h := New("v0")
	clashYAML := `proxies:
  - name: "n1"
    type: trojan
    server: t.example.com
    port: 443
    password: "pw"
    sni: "t.example.com"
`
	w := do(t, h, "POST", "/api/convert?to=singbox", clashYAML)
	if w.Code != 200 {
		t.Fatalf("code=%d body=%q", w.Code, w.Body.String())
	}
	out := w.Body.String()
	if !strings.Contains(out, `"type": "trojan"`) || !strings.Contains(out, `"t.example.com"`) {
		t.Errorf("output missing expected singbox fields: %s", out)
	}
}

func TestConvert_MissingTo(t *testing.T) {
	h := New("v0")
	w := do(t, h, "POST", "/api/convert", "x")
	if w.Code != 400 {
		t.Errorf("code=%d (want 400)", w.Code)
	}
}

func TestConvert_EmptyBody(t *testing.T) {
	h := New("v0")
	w := do(t, h, "POST", "/api/convert?to=clash", "")
	if w.Code != 400 {
		t.Errorf("code=%d (want 400)", w.Code)
	}
}

func TestConvert_TooBig(t *testing.T) {
	h := New("v0")
	big := strings.Repeat("x", MaxBodyBytes+1)
	w := do(t, h, "POST", "/api/convert?to=clash", big)
	if w.Code != 400 {
		t.Errorf("code=%d (want 400 on oversize)", w.Code)
	}
}

// Sanity: ensure embed is actually pulling the HTML in (non-empty).
func TestIndexEmbedded(t *testing.T) {
	if len(indexHTML) < 100 {
		t.Fatalf("indexHTML seems empty (%d bytes) — embed not working?", len(indexHTML))
	}
}

// Make `io` usage explicit for future expansion (streamed bodies).
var _ = io.EOF
