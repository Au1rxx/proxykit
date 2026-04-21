package main

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"time"

	"github.com/spf13/cobra"

	upstream "github.com/Au1rxx/free-vpn-subscriptions/pkg/node"
	"github.com/Au1rxx/free-vpn-subscriptions/pkg/probe"

	"github.com/Au1rxx/proxykit/internal/convert"
)

func newTestCmd() *cobra.Command {
	var (
		inPath      string
		outPath     string
		from        string
		format      string
		fast        bool
		timeoutMS   int
		concurrency int
		limit       int
	)
	cmd := &cobra.Command{
		Use:   "test",
		Short: "Probe a subscription and report which nodes are alive",
		Long: "Parse an input subscription and run handshake probes (TCP, " +
			"plus TLS for nodes that advertise it). --fast is the only " +
			"supported mode today; --full (HTTP-over-proxy) is planned.\n\n" +
			"Input formats mirror `proxykit convert`. Output can be a " +
			"human-readable table (default), JSON, or CSV.",
		Example: "  proxykit test -i sub.yaml\n" +
			"  proxykit test -i sub.yaml --format json -o report.json\n" +
			"  proxykit test -i - --from base64 --format csv",
		RunE: func(cmd *cobra.Command, args []string) error {
			if !fast {
				return fmt.Errorf("--fast is currently the only supported mode (default); --full will arrive in a later release")
			}

			data, err := readInput(inPath)
			if err != nil {
				return err
			}
			nodes, err := convert.Decode(data, from)
			if err != nil {
				return fmt.Errorf("decode: %w", err)
			}
			if len(nodes) == 0 {
				return fmt.Errorf("no valid nodes parsed from input")
			}
			if limit > 0 && limit < len(nodes) {
				nodes = nodes[:limit]
			}

			ctx, cancel := context.WithCancel(cmd.Context())
			defer cancel()

			timeout := time.Duration(timeoutMS) * time.Millisecond
			fmt.Fprintf(os.Stderr, "probing %d nodes (TCP timeout %s, concurrency %d)...\n",
				len(nodes), timeout, concurrency)
			tcpAlive := probe.TCP(ctx, nodes, timeout, concurrency)
			fmt.Fprintf(os.Stderr, "tcp alive: %d / %d\n", len(tcpAlive), len(nodes))

			tlsAlive := probe.TLS(ctx, tcpAlive, timeout, concurrency)
			fmt.Fprintf(os.Stderr, "tls alive: %d / %d\n", len(tlsAlive), len(tcpAlive))

			sort.SliceStable(tlsAlive, func(i, j int) bool {
				return tlsAlive[i].LatencyMS < tlsAlive[j].LatencyMS
			})

			return writeReport(outPath, format, tlsAlive, len(nodes), len(tcpAlive))
		},
	}
	cmd.Flags().StringVarP(&inPath, "in", "i", "-", "input file, or '-' for stdin")
	cmd.Flags().StringVarP(&outPath, "out", "o", "-", "output file, or '-' for stdout")
	cmd.Flags().StringVar(&from, "from", "auto", "input format: auto|clash|uri-list|base64")
	cmd.Flags().StringVar(&format, "format", "table", "report format: table|json|csv")
	cmd.Flags().BoolVar(&fast, "fast", true, "handshake-only probe (TCP + TLS)")
	cmd.Flags().IntVar(&timeoutMS, "timeout-ms", 4000, "per-handshake timeout in milliseconds")
	cmd.Flags().IntVar(&concurrency, "concurrency", 50, "max in-flight probes")
	cmd.Flags().IntVar(&limit, "limit", 0, "only probe the first N nodes (0 = no limit)")
	return cmd
}

type report struct {
	TotalInput int          `json:"total_input"`
	TCPAlive   int          `json:"tcp_alive"`
	TLSAlive   int          `json:"tls_alive"`
	Nodes      []reportNode `json:"nodes"`
}

type reportNode struct {
	Name      string `json:"name"`
	Protocol  string `json:"protocol"`
	Server    string `json:"server"`
	Port      int    `json:"port"`
	LatencyMS int    `json:"latency_ms"`
}

func writeReport(path, format string, alive []*upstream.Node, total, tcpAlive int) error {
	r := report{
		TotalInput: total,
		TCPAlive:   tcpAlive,
		TLSAlive:   len(alive),
		Nodes:      make([]reportNode, 0, len(alive)),
	}
	for _, n := range alive {
		r.Nodes = append(r.Nodes, reportNode{
			Name:      n.Name,
			Protocol:  n.Protocol,
			Server:    n.Server,
			Port:      n.Port,
			LatencyMS: n.LatencyMS,
		})
	}

	w, closeFn, err := openOutput(path)
	if err != nil {
		return err
	}
	defer closeFn()

	switch format {
	case "json":
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(r)
	case "csv":
		cw := csv.NewWriter(w)
		defer cw.Flush()
		if err := cw.Write([]string{"name", "protocol", "server", "port", "latency_ms"}); err != nil {
			return err
		}
		for _, n := range r.Nodes {
			if err := cw.Write([]string{
				n.Name, n.Protocol, n.Server, strconv.Itoa(n.Port), strconv.Itoa(n.LatencyMS),
			}); err != nil {
				return err
			}
		}
		return nil
	case "table":
		fmt.Fprintf(w, "%-4s  %-14s  %-5s  %s\n", "RTT", "PROTO", "PORT", "SERVER")
		for _, n := range r.Nodes {
			fmt.Fprintf(w, "%4d  %-14s  %5d  %s\n", n.LatencyMS, n.Protocol, n.Port, n.Server)
		}
		fmt.Fprintf(w, "\n%d / %d nodes alive (tcp %d, tls %d)\n",
			r.TLSAlive, r.TotalInput, r.TCPAlive, r.TLSAlive)
		return nil
	default:
		return fmt.Errorf("unknown --format %q (want table|json|csv)", format)
	}
}

func openOutput(path string) (io.Writer, func(), error) {
	if path == "-" {
		return os.Stdout, func() {}, nil
	}
	f, err := os.Create(path)
	if err != nil {
		return nil, nil, fmt.Errorf("create %s: %w", path, err)
	}
	return f, func() { _ = f.Close() }, nil
}
