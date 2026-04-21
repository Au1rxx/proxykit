package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/Au1rxx/free-vpn-subscriptions/pkg/unlock"
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
			"This first release only supports --direct mode — probes run from the " +
			"local machine. --via <proxy-uri> (route through a single proxy node) is " +
			"planned and will arrive together with an embedded sing-box launcher.",
		Example: "  proxykit unlock --direct\n" +
			"  proxykit unlock --direct --target netflix,chatgpt --format json",
		RunE: func(cmd *cobra.Command, args []string) error {
			if via != "" {
				return fmt.Errorf("--via is not yet implemented; use --direct for now")
			}
			if !direct {
				return fmt.Errorf("--direct is required (for this release it's the only supported mode)")
			}

			selected, err := selectTargets(targets)
			if err != nil {
				return err
			}

			client := &http.Client{Timeout: time.Duration(timeoutMS) * time.Millisecond}
			ctx := cmd.Context()

			fmt.Fprintf(os.Stderr, "probing %d targets (direct, per-target %dms)...\n", len(selected), timeoutMS)
			results := unlock.Run(ctx, client, selected, time.Duration(timeoutMS)*time.Millisecond)

			return writeUnlockReport(outPath, format, results)
		},
	}
	cmd.Flags().StringVarP(&outPath, "out", "o", "-", "output file, or '-' for stdout")
	cmd.Flags().StringVar(&format, "format", "table", "report format: table|json")
	cmd.Flags().StringVar(&targets, "target", "", "comma-separated subset; default = all (netflix,disney,youtube-premium,chatgpt)")
	cmd.Flags().IntVar(&timeoutMS, "timeout-ms", 8000, "per-target timeout in milliseconds")
	cmd.Flags().BoolVar(&direct, "direct", false, "probe from this machine (no proxy) — required until --via lands")
	cmd.Flags().StringVar(&via, "via", "", "route probes through a single proxy URI (NOT yet implemented)")
	return cmd
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
