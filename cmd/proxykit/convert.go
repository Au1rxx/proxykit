package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/Au1rxx/proxykit/internal/convert"
)

func newConvertCmd() *cobra.Command {
	var (
		inPath  string
		outPath string
		from    string
		to      string
	)
	cmd := &cobra.Command{
		Use:   "convert",
		Short: "Convert a subscription between clash / singbox / v2ray / surge / quanx / loon",
		Long: "Read a subscription file (or stdin if -i -) in one format, " +
			"decode it into the normalized node model, and re-emit it in " +
			"the chosen output format.\n\n" +
			"Input formats (--from): auto (default), clash, uri-list, base64\n" +
			"Output formats (--to):  clash, singbox, v2ray, surge, quanx, loon\n\n" +
			"surge / quanx / loon are partial-coverage — VLESS and Hysteria2 " +
			"nodes are dropped silently (no native mapping in those clients).",
		Example: "  proxykit convert -i sub.yaml --to singbox -o out.json\n" +
			"  cat sub.txt | proxykit convert -i - --from base64 --to clash\n" +
			"  proxykit convert -i sub.yaml --to surge -o surge.conf",
		RunE: func(cmd *cobra.Command, args []string) error {
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
			out, err := convert.Encode(nodes, to)
			if err != nil {
				return fmt.Errorf("encode: %w", err)
			}
			if err := writeOutput(outPath, out); err != nil {
				return err
			}
			fmt.Fprintf(os.Stderr, "converted %d nodes -> %s\n", len(nodes), to)
			return nil
		},
	}
	cmd.Flags().StringVarP(&inPath, "in", "i", "-", "input file, or '-' for stdin")
	cmd.Flags().StringVarP(&outPath, "out", "o", "-", "output file, or '-' for stdout")
	cmd.Flags().StringVar(&from, "from", "auto", "input format: auto|clash|uri-list|base64")
	cmd.Flags().StringVar(&to, "to", "", "output format: clash|singbox|v2ray|surge|quanx|loon (required)")
	_ = cmd.MarkFlagRequired("to")
	return cmd
}

func readInput(path string) ([]byte, error) {
	if path == "-" {
		var buf bytes.Buffer
		if _, err := io.Copy(&buf, os.Stdin); err != nil {
			return nil, fmt.Errorf("read stdin: %w", err)
		}
		return buf.Bytes(), nil
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	return b, nil
}

func writeOutput(path, content string) error {
	if path == "-" {
		_, err := io.WriteString(os.Stdout, ensureTrailingNewline(content))
		return err
	}
	return os.WriteFile(path, []byte(ensureTrailingNewline(content)), 0o644)
}

func ensureTrailingNewline(s string) string {
	if strings.HasSuffix(s, "\n") {
		return s
	}
	return s + "\n"
}
