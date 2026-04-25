//go:build js && wasm

package main

import (
	"syscall/js"

	"github.com/Au1rxx/proxykit/internal/convert"
)

func proxykitConvert(this js.Value, args []js.Value) any {
	if len(args) < 3 {
		return errResult("expected (input, from, to)")
	}
	input := []byte(args[0].String())
	from := args[1].String()
	to := args[2].String()
	if from == "" {
		from = "auto"
	}

	nodes, err := convert.Decode(input, from)
	if err != nil {
		return errResult("decode: " + err.Error())
	}
	out, err := convert.Encode(nodes, to)
	if err != nil {
		return errResult("encode: " + err.Error())
	}
	detected := from
	if detected == "auto" {
		detected = convert.Detect(input)
	}
	return js.ValueOf(map[string]any{
		"ok":       true,
		"output":   out,
		"count":    len(nodes),
		"detected": detected,
	})
}

func errResult(msg string) any {
	return js.ValueOf(map[string]any{
		"ok":    false,
		"error": msg,
	})
}

func main() {
	js.Global().Set("proxykitConvert", js.FuncOf(proxykitConvert))
	select {} // block forever
}
