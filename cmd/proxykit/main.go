// Command proxykit is the CLI entrypoint for the ProxyKit toolbox.
//
// See https://github.com/Au1rxx/proxykit for the full story; the subcommand
// implementations live under internal/.
package main

import (
	"fmt"
	"os"
)

// Version is stamped at build time via -ldflags "-X main.Version=...".
var Version = "dev"

func main() {
	if len(os.Args) >= 2 && (os.Args[1] == "--version" || os.Args[1] == "-v") {
		fmt.Println("proxykit", Version)
		return
	}
	fmt.Fprintln(os.Stderr, "proxykit: subcommands not wired yet; scaffold only. See docs/roadmap.md.")
	os.Exit(1)
}
