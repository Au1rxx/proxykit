package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/Au1rxx/proxykit/internal/server"
)

func newServeCmd() *cobra.Command {
	var (
		addr          string
		authToken     string
		enableProbes  bool
		maxTestNodes  int
		maxUnlockNs   int
		parallelReqs  int
	)
	cmd := &cobra.Command{
		Use:   "serve",
		Short: "HTTP server exposing /api/convert (+ optional /api/test, /api/unlock) and a browser page",
		Long: "Starts an HTTP server wrapping proxykit's CLI pipelines. Endpoints:\n" +
			"  POST /api/convert?from=...&to=...   body = subscription\n" +
			"  POST /api/test?from=...             body = subscription (if --enable-probes)\n" +
			"  POST /api/unlock?from=...&target=.. body = subscription (if --enable-probes)\n" +
			"  GET  /health                        200 \"ok\"\n" +
			"  GET  /version                       200 \"proxykit <ver>\"\n" +
			"  GET  /                              embedded HTML tool\n\n" +
			"convert is always on (pure compute, minimal threat surface).\n" +
			"test/unlock are gated behind --enable-probes because they:\n" +
			"  * spawn sing-box subprocesses and probe the public internet\n" +
			"  * can be abused for SSRF if the server sits behind a permissive egress\n" +
			"The built-in SSRF filter drops RFC1918/loopback/link-local/cloud-metadata\n" +
			"IPs, but a hostname that resolves privately is NOT caught — run behind a\n" +
			"network policy that blocks outbound private ranges for untrusted users.\n" +
			"--auth-token <string> adds Authorization: Bearer <token> on test/unlock.",
		Example: "  proxykit serve --addr 127.0.0.1:8080\n" +
			"  proxykit serve --addr 0.0.0.0:8080 --enable-probes --auth-token $SECRET\n" +
			"  proxykit serve --enable-probes --parallel 4 --max-test-nodes 100",
		RunE: func(cmd *cobra.Command, args []string) error {
			var guard *server.Guard
			if enableProbes {
				guard = &server.Guard{
					AuthToken:      authToken,
					MaxNodesTest:   maxTestNodes,
					MaxNodesUnlock: maxUnlockNs,
					Parallel:       parallelReqs,
				}
			}
			h := server.New(Version, guard)
			srv := &http.Server{
				Addr:              addr,
				Handler:           h,
				ReadHeaderTimeout: 5 * time.Second,
			}

			ctx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
			defer stop()

			errCh := make(chan error, 1)
			go func() { errCh <- srv.ListenAndServe() }()

			fmt.Fprintf(os.Stderr, "proxykit %s listening on http://%s (Ctrl-C to stop)\n", Version, addr)

			select {
			case err := <-errCh:
				if errors.Is(err, http.ErrServerClosed) {
					return nil
				}
				return err
			case <-ctx.Done():
				shutdownCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
				defer cancel()
				return srv.Shutdown(shutdownCtx)
			}
		},
	}
	cmd.Flags().StringVar(&addr, "addr", "127.0.0.1:8080", "listen address")
	cmd.Flags().BoolVar(&enableProbes, "enable-probes", false, "enable /api/test and /api/unlock (they spawn sing-box and hit the public internet)")
	cmd.Flags().StringVar(&authToken, "auth-token", "", "if set, test/unlock require Authorization: Bearer <token>")
	cmd.Flags().IntVar(&maxTestNodes, "max-test-nodes", 50, "per-request cap for /api/test")
	cmd.Flags().IntVar(&maxUnlockNs, "max-unlock-nodes", 10, "per-request cap for /api/unlock (each spawns sing-box)")
	cmd.Flags().IntVar(&parallelReqs, "parallel", 2, "global concurrent in-flight test/unlock requests")
	return cmd
}
