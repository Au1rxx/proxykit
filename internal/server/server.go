package server

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Au1rxx/free-vpn-subscriptions/pkg/node"
	"github.com/Au1rxx/free-vpn-subscriptions/pkg/probe"
	"github.com/Au1rxx/free-vpn-subscriptions/pkg/unlock"

	"github.com/Au1rxx/proxykit/internal/convert"
	"github.com/Au1rxx/proxykit/internal/singbox"
)

// MaxBodyBytes caps request bodies at 2 MiB. Plenty for a Clash YAML
// with thousands of proxies; stops an accidental curl | cat from OOM'ing
// the host.
const MaxBodyBytes = 2 << 20

//go:embed index.html
var indexHTML []byte

// New returns an http.Handler wiring all MVP endpoints. version is
// what /version reports and what the embedded page shows in its
// footer. guard is optional — nil means "convert-only server, no
// outbound probes".
func New(version string, guard *Guard) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write([]byte("ok"))
	})
	mux.HandleFunc("GET /version", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		fmt.Fprintf(w, "proxykit %s", version)
	})
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(indexHTML)
	})
	mux.HandleFunc("POST /api/convert", handleConvert)

	if guard != nil {
		mux.HandleFunc("POST /api/test", guard.requireAuth(guard.handleTest))
		mux.HandleFunc("POST /api/unlock", guard.requireAuth(guard.handleUnlock))
	}
	return mux
}

func handleConvert(w http.ResponseWriter, r *http.Request) {
	from := strings.TrimSpace(r.URL.Query().Get("from"))
	if from == "" {
		from = "auto"
	}
	to := strings.TrimSpace(r.URL.Query().Get("to"))
	if to == "" {
		httpError(w, http.StatusBadRequest, "query param `to` is required (clash|singbox|v2ray|surge|quanx|loon)")
		return
	}

	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, MaxBodyBytes))
	if err != nil {
		httpError(w, http.StatusBadRequest, "read body: %v", err)
		return
	}
	defer r.Body.Close()

	nodes, err := convert.Decode(body, from)
	if err != nil {
		httpError(w, http.StatusBadRequest, "decode: %v", err)
		return
	}
	if len(nodes) == 0 {
		httpError(w, http.StatusBadRequest, "decode: no nodes parsed (check --from / payload)")
		return
	}
	out, err := convert.Encode(nodes, to)
	if err != nil {
		httpError(w, http.StatusBadRequest, "encode: %v", err)
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = w.Write([]byte(out))
}

// TestReport is what /api/test returns.
type TestReport struct {
	Total   int         `json:"total"`
	Dropped []string    `json:"dropped,omitempty"`
	Alive   []AliveNode `json:"alive"`
}

// AliveNode is one alive row in /api/test output.
type AliveNode struct {
	Name     string `json:"name"`
	Protocol string `json:"protocol"`
	Server   string `json:"server"`
	Port     int    `json:"port"`
	TLS      bool   `json:"tls"`
}

func (g *Guard) handleTest(w http.ResponseWriter, r *http.Request) {
	g.init()

	nodes, dropped, err := g.parseAndFilter(r, g.MaxNodesTest)
	if err != nil {
		httpError(w, http.StatusBadRequest, "%v", err)
		return
	}

	release, err := g.acquire(r.Context())
	if err != nil {
		httpError(w, http.StatusServiceUnavailable, "server busy: %v", err)
		return
	}
	defer release()

	timeout := 3 * time.Second
	tcpAlive := probe.TCP(r.Context(), nodes, timeout, 20)
	tlsAlive := probe.TLS(r.Context(), tcpAlive, timeout, 10)
	tlsOK := map[string]bool{}
	for _, n := range tlsAlive {
		tlsOK[n.Key()] = true
	}

	rep := TestReport{Total: len(nodes), Dropped: dropped}
	for _, n := range tcpAlive {
		rep.Alive = append(rep.Alive, AliveNode{
			Name: n.Name, Protocol: string(n.Protocol),
			Server: n.Server, Port: n.Port,
			TLS: tlsOK[n.Key()],
		})
	}
	writeJSON(w, rep)
}

// UnlockReport is what /api/unlock returns.
type UnlockReport struct {
	Total   int           `json:"total"`
	Dropped []string      `json:"dropped,omitempty"`
	Rows    []UnlockRow   `json:"rows"`
	Targets []string      `json:"targets"`
}

// UnlockRow is one node in the matrix.
type UnlockRow struct {
	Node    string          `json:"node"`
	Server  string          `json:"server"`
	Error   string          `json:"error,omitempty"`
	Results []unlock.Result `json:"results,omitempty"`
}

func (g *Guard) handleUnlock(w http.ResponseWriter, r *http.Request) {
	g.init()

	nodes, dropped, err := g.parseAndFilter(r, g.MaxNodesUnlock)
	if err != nil {
		httpError(w, http.StatusBadRequest, "%v", err)
		return
	}

	targets, err := selectTargets(r.URL.Query().Get("target"))
	if err != nil {
		httpError(w, http.StatusBadRequest, "%v", err)
		return
	}

	release, err := g.acquire(r.Context())
	if err != nil {
		httpError(w, http.StatusServiceUnavailable, "server busy: %v", err)
		return
	}
	defer release()

	perTarget := 5 * time.Second
	rep := UnlockReport{Total: len(nodes), Dropped: dropped}
	for _, t := range targets {
		rep.Targets = append(rep.Targets, t.Name)
	}

	for _, n := range nodes {
		row := UnlockRow{Node: labelFor(n), Server: fmt.Sprintf("%s:%d", n.Server, n.Port)}
		proc, err := singbox.LaunchNode(r.Context(), n, singbox.Config{})
		if err != nil {
			row.Error = err.Error()
			rep.Rows = append(rep.Rows, row)
			continue
		}
		proxyURL, _ := url.Parse("socks5://" + proc.SocksAddr)
		client := &http.Client{
			Transport: &http.Transport{
				Proxy:                 http.ProxyURL(proxyURL),
				DisableKeepAlives:     true,
				ResponseHeaderTimeout: perTarget,
			},
			Timeout: perTarget,
		}
		row.Results = unlock.Run(r.Context(), client, targets, perTarget)
		proc.Stop()
		rep.Rows = append(rep.Rows, row)
	}

	writeJSON(w, rep)
}

func (g *Guard) parseAndFilter(r *http.Request, cap int) ([]*node.Node, []string, error) {
	from := strings.TrimSpace(r.URL.Query().Get("from"))
	if from == "" {
		from = "auto"
	}
	body, err := io.ReadAll(http.MaxBytesReader(nil, r.Body, MaxBodyBytes))
	if err != nil {
		return nil, nil, fmt.Errorf("read body: %w", err)
	}
	defer r.Body.Close()

	nodes, err := convert.Decode(body, from)
	if err != nil {
		return nil, nil, fmt.Errorf("decode: %w", err)
	}
	if len(nodes) == 0 {
		return nil, nil, fmt.Errorf("decode: no nodes parsed")
	}
	kept, dropped := filterNodes(nodes)
	if len(kept) == 0 {
		return nil, dropped, fmt.Errorf("all %d nodes blocked by SSRF filter", len(nodes))
	}
	if len(kept) > cap {
		dropped = append(dropped, fmt.Sprintf("truncated from %d to %d by server cap", len(kept), cap))
		kept = kept[:cap]
	}
	return kept, dropped, nil
}

func selectTargets(spec string) ([]unlock.Target, error) {
	all := unlock.All()
	if spec == "" {
		return all, nil
	}
	want := map[string]bool{}
	for _, t := range strings.Split(spec, ",") {
		t = strings.TrimSpace(strings.ToLower(t))
		if t != "" {
			want[t] = true
		}
	}
	out := make([]unlock.Target, 0, len(want))
	for _, t := range all {
		if want[t.Name] {
			out = append(out, t)
			delete(want, t.Name)
		}
	}
	for unknown := range want {
		return nil, fmt.Errorf("unknown target %q", unknown)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("no targets selected")
	}
	return out, nil
}

func labelFor(n *node.Node) string {
	if n.Name != "" {
		return n.Name
	}
	return string(n.Protocol) + "@" + n.Server
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	_ = enc.Encode(v)
}

func httpError(w http.ResponseWriter, code int, format string, args ...any) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(code)
	fmt.Fprintf(w, format, args...)
}

// Ensure the context package is linked even if no handler uses it
// directly (reserved for future timeout wiring).
var _ = context.TODO
