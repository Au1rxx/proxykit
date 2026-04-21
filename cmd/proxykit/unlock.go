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

	"github.com/Au1rxx/free-vpn-subscriptions/pkg/unlock"

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
	)
	cmd := &cobra.Command{
		Use:   "unlock",
		Short: "Check streaming / service unlock status (Netflix, Disney+, YouTube Premium, ChatGPT)",
		Long: "Run the unlock probe suite against an HTTP client and report per-target " +
			"Status (blocked|partial|unlocked) + Region when known.\n\n" +
			"Two modes:\n" +
			"  --direct           probe from this machine (no proxy)\n" +
			"  --via <proxy-uri>  spin up a one-shot sing-box SOCKS5 inbound\n" +
			"                     bound to the given vless/vmess/trojan/ss/hy2\n" +
			"                     URI and route probes through it\n\n" +
			"`--via` requires a `sing-box` binary on PATH (https://sing-box.sagernet.org/).",
		Example: "  proxykit unlock --direct\n" +
			"  proxykit unlock --direct --target netflix,chatgpt --format json\n" +
			"  proxykit unlock --via 'trojan://pw@host:443?sni=host#t1'",
		RunE: func(cmd *cobra.Command, args []string) error {
			if (direct && via != "") || (!direct && via == "") {
				return fmt.Errorf("exactly one of --direct or --via <uri> is required")
			}

			selected, err := selectTargets(targets)
			if err != nil {
				return err
			}

			ctx := cmd.Context()
			client, modeLabel, cleanup, err := buildUnlockClient(ctx, direct, via, timeoutMS)
			if err != nil {
				return err
			}
			defer cleanup()

			fmt.Fprintf(os.Stderr, "probing %d targets (%s, per-target %dms)...\n", len(selected), modeLabel, timeoutMS)
			results := unlock.Run(ctx, client, selected, time.Duration(timeoutMS)*time.Millisecond)

			return writeUnlockReport(outPath, format, results)
		},
	}
	cmd.Flags().StringVarP(&outPath, "out", "o", "-", "output file, or '-' for stdout")
	cmd.Flags().StringVar(&format, "format", "table", "report format: table|json")
	cmd.Flags().StringVar(&targets, "target", "", "comma-separated subset; default = all (netflix,disney,youtube-premium,chatgpt)")
	cmd.Flags().IntVar(&timeoutMS, "timeout-ms", 8000, "per-target timeout in milliseconds")
	cmd.Flags().BoolVar(&direct, "direct", false, "probe from this machine (no proxy)")
	cmd.Flags().StringVar(&via, "via", "", "route probes through a single proxy URI (vless/vmess/trojan/ss/hy2); requires sing-box on PATH")
	return cmd
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
	proxyURL, _ := url.Parse("socks5://" + proc.SocksAddr)
	client := &http.Client{
		Transport: &http.Transport{
			Proxy:                 http.ProxyURL(proxyURL),
			DisableKeepAlives:     true,
			ResponseHeaderTimeout: time.Duration(timeoutMS) * time.Millisecond,
		},
		Timeout: time.Duration(timeoutMS) * time.Millisecond,
	}
	return client, "via " + proc.SocksAddr, proc.Stop, nil
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
