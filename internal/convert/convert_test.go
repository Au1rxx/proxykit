package convert

import (
	"encoding/json"
	"strings"
	"testing"
)

const sampleURIList = `trojan://pw@trojan.example.com:443?sni=trojan.example.com#t1
ss://YWVzLTI1Ni1nY206cHc@ss.example.com:8388#ss1
`

const sampleClash = `proxies:
  - name: demo-trojan
    type: trojan
    server: trojan.example.com
    port: 443
    password: pw
    sni: trojan.example.com
  - name: demo-ss
    type: ss
    server: ss.example.com
    port: 8388
    cipher: aes-256-gcm
    password: pw2
`

func TestDetect(t *testing.T) {
	cases := map[string]string{
		"uri-list": sampleURIList,
		"clash":    sampleClash,
	}
	for want, body := range cases {
		if got := Detect([]byte(body)); got != want {
			t.Errorf("Detect(%s): got %q, want %q", want, got, want)
		}
	}
}

func TestConvert_ClashToSingbox(t *testing.T) {
	nodes, err := Decode([]byte(sampleClash), "auto")
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if len(nodes) != 2 {
		t.Fatalf("got %d nodes, want 2", len(nodes))
	}

	out, err := Encode(nodes, "singbox")
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	var cfg map[string]any
	if err := json.Unmarshal([]byte(out), &cfg); err != nil {
		t.Fatalf("sing-box output is not valid JSON: %v", err)
	}
	if !strings.Contains(out, "trojan.example.com") {
		t.Error("sing-box output missing trojan server")
	}
}

func TestConvert_URIListToClash(t *testing.T) {
	nodes, err := Decode([]byte(sampleURIList), "auto")
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if len(nodes) == 0 {
		t.Fatal("no nodes parsed from URI list")
	}
	out, err := Encode(nodes, "clash")
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	if !strings.Contains(out, "proxies:") {
		t.Error("Clash output missing proxies block")
	}
}

func TestEncode_RejectsUnknownFormat(t *testing.T) {
	if _, err := Encode(nil, "nope"); err == nil {
		t.Error("expected error for unknown output format, got nil")
	}
}

func TestConvert_ClashToPartialFormats(t *testing.T) {
	nodes, err := Decode([]byte(sampleClash), "auto")
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	cases := []struct {
		format string
		wants  []string
	}{
		{"surge", []string{"[Proxy]", "= trojan, trojan.example.com, 443", "= ss, ss.example.com, 8388"}},
		{"quanx", []string{"[server_local]", "trojan=trojan.example.com:443", "shadowsocks=ss.example.com:8388"}},
		{"loon", []string{"[Proxy]", "= trojan,trojan.example.com,443", "= Shadowsocks,ss.example.com,8388"}},
	}
	for _, c := range cases {
		out, err := Encode(nodes, c.format)
		if err != nil {
			t.Errorf("Encode(%s): %v", c.format, err)
			continue
		}
		for _, w := range c.wants {
			if !strings.Contains(out, w) {
				t.Errorf("Encode(%s) missing %q:\n%s", c.format, w, out)
			}
		}
	}
}
