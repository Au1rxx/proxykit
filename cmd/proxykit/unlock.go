package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/Au1rxx/free-vpn-subscriptions/pkg/node"
	"github.com/Au1rxx/free-vpn-subscriptions/pkg/unlock"

	"github.com/Au1rxx/proxykit/internal/convert"
	"github.com/Au1rxx/proxykit/internal/singbox"
)

func newUnlockCmd() *cobra.Command {
	var (
		outPath   string
		format    string
		targets   string
		timeoutMS int
		direct    bool
		via       string
		sub       string
		from      string
	)
	cmd := &cobra.Command{
		Use:   "unlock",
		Short: "Check streaming / service unlock status (Netflix, Disney+, YouTube Premium, ChatGPT)",
		Long: "Run the unlock probe suite against an HTTP client and report per-target " +
			"Status (blocked|partial|unlocked) + Region when known.\n\n" +
			"Three modes (exactly one required):\n" +
			"  --direct              probe from this machine (no proxy)\n" +
			"  --via <proxy-uri>     spin up a one-shot sing-box SOCKS5 inbound\n" +
			"                        bound to the given vless/vmess/trojan/ss/hy2\n" +
			"                        URI and route probes through it\n" +
			"  --sub <file>          parse a subscription file (clash/v2ray/uri-list,\n" +
			"                        auto-detected) and run unlock against every\n" +
			"                        node in sequence — one sing-box subprocess\n" +
			"                        per node, matrix output\n\n" +
			"`--via` and `--sub` require a `sing-box` binary on PATH (https://sing-box.sagernet.org/).",
		Example: "  proxykit unlock --direct\n" +
			"  proxykit unlock --direct --target netflix,chatgpt --format json\n" +
			"  proxykit unlock --via 'trojan://pw@host:443?sni=host#t1'\n" +
			"  proxykit unlock --sub nodes.yaml --format json -o matrix.json",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := checkMutex(direct, via, sub); err != nil {
				return err
			}
			selected, err := selectTargets(targets)
			if err != nil {
				return err
			}
			ctx := cmd.Context()
			perTarget := time.Duration(timeoutMS) * time.Millisecond

			if sub != "" {
				return runSubMatrix(ctx, sub, from, selected, perTarget, outPath, format)
			}

			client, modeLabel, cleanup, err := buildUnlockClient(ctx, direct, via, timeoutMS)
			if err != nil {
				return err
			}
			defer cleanup()

			fmt.Fprintf(os.Stderr, "probing %d targets (%s, per-target %dms)...\n", len(selected), modeLabel, timeoutMS)
			results := unlock.Run(ctx, client, selected, perTarget)

			return writeUnlockReport(outPath, format, results)
		},
	}
	cmd.Flags().StringVarP(&outPath, "out", "o", "-", "output file, or '-' for stdout")
	cmd.Flags().StringVar(&format, "format", "table", "report format: table|json")
	cmd.Flags().StringVar(&targets, "target", "", "comma-separated subset; default = all (netflix,disney,youtube-premium,chatgpt)")
	cmd.Flags().IntVar(&timeoutMS, "timeout-ms", 8000, "per-target timeout in milliseconds")
	cmd.Flags().BoolVar(&direct, "direct", false, "probe from this machine (no proxy)")
	cmd.Flags().StringVar(&via, "via", "", "route probes through a single proxy URI (vless/vmess/trojan/ss/hy2); requires sing-box on PATH")
	cmd.Flags().StringVar(&sub, "sub", "", "subscription file (clash/v2ray/uri-list); runs unlock per node; requires sing-box on PATH")
	cmd.Flags().StringVar(&from, "from", "auto", "input format for --sub: auto|clash|v2ray|uri-list|base64")
	return cmd
}

func checkMutex(direct bool, via, sub string) error {
	n := 0
	if direct {
		n++
	}
	if via != "" {
		n++
	}
	if sub != "" {
		n++
	}
	if n != 1 {
		return fmt.Errorf("exactly one of --direct, --via <uri>, or --sub <file> is required")
	}
	return nil
}

// buildUnlockClient returns an *http.Client either talking directly
// from this host or routed through a one-shot sing-box SOCKS5 inbound
// bound to the given URI. cleanup is always non-nil and safe to defer.
func buildUnlockClient(ctx context.Context, direct bool, via string, timeoutMS int) (*http.Client, string, func(), error) {
	if direct {
		return &http.Client{Timeout: time.Duration(timeoutMS) * time.Millisecond}, "direct", func() {}, nil
	}
	proc, err := singbox.Launch(ctx, via, singbox.Config{})
	if err != nil {
		return nil, "", nil, fmt.Errorf("launch sing-box: %w", err)
	}
	return nodeClient(proc, timeoutMS), "via " + proc.SocksAddr, proc.Stop, nil
}

func nodeClient(proc *singbox.Proc, timeoutMS int) *http.Client {
	proxyURL, _ := url.Parse("socks5://" + proc.SocksAddr)
	return &http.Client{
		Transport: &http.Transport{
			Proxy:                 http.ProxyURL(proxyURL),
			DisableKeepAlives:     true,
			ResponseHeaderTimeout: time.Duration(timeoutMS) * time.Millisecond,
		},
		Timeout: time.Duration(timeoutMS) * time.Millisecond,
	}
}

// NodeRow is one row in the matrix — one subscription node with its
// per-target unlock verdicts (or an Error describing why the node
// was skipped before probes could run).
type NodeRow struct {
	Node    string          `json:"node"`
	Server  string          `json:"server"`
	Error   string          `json:"error,omitempty"`
	Results []unlock.Result `json:"results,omitempty"`
}

func runSubMatrix(ctx context.Context, path, from string, selected []unlock.Target, perTarget time.Duration, outPath, format string) error {
	body, err := readInput(path)
	if err != nil {
		return fmt.Errorf("read subscription: %w", err)
	}
	nodes, err := convert.Decode(body, from)
	if err != nil {
		return fmt.Errorf("parse subscription (--from %s): %w", from, err)
	}
	if len(nodes) == 0 {
		return fmt.Errorf("no nodes in subscription")
	}

	timeoutMS := int(perTarget / time.Millisecond)
	fmt.Fprintf(os.Stderr, "probing %d nodes × %d targets (per-target %dms, sequential)...\n", len(nodes), len(selected), timeoutMS)

	rows := make([]NodeRow, 0, len(nodes))
	for i, n := range nodes {
		row := NodeRow{Node: nodeLabel(n, i), Server: fmt.Sprintf("%s:%d", n.Server, n.Port)}
		fmt.Fprintf(os.Stderr, "[%d/%d] %s\n", i+1, len(nodes), row.Node)

		proc, err := singbox.LaunchNode(ctx, n, singbox.Config{})
		if err != nil {
			row.Error = err.Error()
			rows = append(rows, row)
			continue
		}
		client := nodeClient(proc, timeoutMS)
		row.Results = unlock.Run(ctx, client, selected, perTarget)
		proc.Stop()
		rows = append(rows, row)
	}

	return writeMatrixReport(outPath, format, selected, rows)
}

func nodeLabel(n *node.Node, idx int) string {
	if n.Name != "" {
		return n.Name
	}
	return fmt.Sprintf("%02d-%s", idx+1, n.Protocol)
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
	known := map[string]bool{}
	for _, t := range all {
		known[t.Name] = true
		if want[t.Name] {
			out = append(out, t)
			delete(want, t.Name)
		}
	}
	for unknown := range want {
		return nil, fmt.Errorf("unknown target %q (known: %s)", unknown, joinTargetNames(all))
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("no targets selected")
	}
	return out, nil
}

func joinTargetNames(targets []unlock.Target) string {
	names := make([]string, 0, len(targets))
	for _, t := range targets {
		names = append(names, t.Name)
	}
	return strings.Join(names, ",")
}

func writeUnlockReport(path, format string, results []unlock.Result) error {
	w, closeFn, err := openOutput(path)
	if err != nil {
		return err
	}
	defer closeFn()

	switch format {
	case "json":
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(results)
	case "table":
		fmt.Fprintf(w, "%-18s  %-9s  %-6s  %s\n", "TARGET", "STATUS", "REGION", "DETAIL")
		for _, r := range results {
			fmt.Fprintf(w, "%-18s  %-9s  %-6s  %s\n", r.Target, r.StatusText, r.Region, r.Detail)
		}
		return nil
	default:
		return fmt.Errorf("unknown --format %q (want table|json)", format)
	}
}

func writeMatrixReport(path, format string, selected []unlock.Target, rows []NodeRow) error {
	w, closeFn, err := openOutput(path)
	if err != nil {
		return err
	}
	defer closeFn()

	switch format {
	case "json":
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(rows)
	case "table":
		cols := make([]string, 0, len(selected))
		for _, t := range selected {
			cols = append(cols, t.Name)
		}
		fmt.Fprintf(w, "%-28s", "NODE")
		for _, c := range cols {
			fmt.Fprintf(w, "%-17s", c)
		}
		fmt.Fprintf(w, "%s\n", "NOTE")
		for _, r := range rows {
			fmt.Fprintf(w, "%-28s", truncCol(r.Node, 27))
			if r.Error != "" {
				for range cols {
					fmt.Fprintf(w, "%-17s", "-")
				}
				fmt.Fprintf(w, "error: %s\n", truncCol(r.Error, 80))
				continue
			}
			statusByTarget := map[string]string{}
			for _, res := range r.Results {
				statusByTarget[res.Target] = res.StatusText
			}
			for _, c := range cols {
				s := statusByTarget[c]
				if s == "" {
					s = "-"
				}
				fmt.Fprintf(w, "%-17s", s)
			}
			fmt.Fprintf(w, "%s\n", r.Server)
		}
		return nil
	default:
		return fmt.Errorf("unknown --format %q (want table|json)", format)
	}
}

func truncCol(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-1] + "…"
}
