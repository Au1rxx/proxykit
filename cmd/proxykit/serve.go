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
	var addr string
	cmd := &cobra.Command{
		Use:   "serve",
		Short: "HTTP server exposing /api/convert + a browser page",
		Long: "Starts an HTTP server wrapping `proxykit convert` and serving " +
			"a single-page browser tool at /. Endpoints:\n" +
			"  POST /api/convert?from=...&to=...   body = subscription\n" +
			"  GET  /health                        200 \"ok\"\n" +
			"  GET  /version                       200 \"proxykit <ver>\"\n" +
			"  GET  /                              embedded HTML tool\n\n" +
			"No sing-box dependency — convert-only for now; test/unlock " +
			"endpoints are a later slice.",
		Example: "  proxykit serve --addr 127.0.0.1:8080\n" +
			"  proxykit serve --addr 0.0.0.0:8080  # bind all interfaces",
		RunE: func(cmd *cobra.Command, args []string) error {
			h := server.New(Version)
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
	return cmd
}
