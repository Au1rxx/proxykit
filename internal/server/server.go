package server

import (
	_ "embed"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/Au1rxx/proxykit/internal/convert"
)

// MaxBodyBytes caps request bodies at 2 MiB. Plenty for a Clash YAML
// with thousands of proxies; stops an accidental curl | cat from OOM'ing
// the host.
const MaxBodyBytes = 2 << 20

//go:embed index.html
var indexHTML []byte

// New returns an http.Handler wiring the MVP endpoints. version is
// what /version reports and what the embedded page shows in its
// footer.
func New(version string) http.Handler {
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

func httpError(w http.ResponseWriter, code int, format string, args ...any) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(code)
	fmt.Fprintf(w, format, args...)
}
