// Command proxykit is the CLI entrypoint for the ProxyKit toolbox.
//
// See https://github.com/Au1rxx/proxykit for the full story; the subcommand
// implementations live under internal/.
package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// Version is stamped at build time via -ldflags "-X main.Version=...".
var Version = "dev"

func main() {
	root := &cobra.Command{
		Use:   "proxykit",
		Short: "Subscription conversion, testing, and ranking toolkit",
		Long: "proxykit is a single-binary Swiss-army toolbox for working " +
			"with proxy subscriptions: convert between Clash / sing-box / " +
			"v2ray formats, batch-test latency and streaming unlock, serve " +
			"a local HTTP API. See `proxykit <command> --help`.",
		Version:       Version,
		SilenceErrors: true,
		SilenceUsage:  true,
	}
	root.SetVersionTemplate("proxykit {{.Version}}\n")
	root.AddCommand(newConvertCmd())
	root.AddCommand(newTestCmd())
	root.AddCommand(newUnlockCmd())
	root.AddCommand(newServeCmd())

	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
