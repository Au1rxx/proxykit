package singbox

import (
	"context"
	"encoding/json"
	"net"
	"os"
	"os/exec"
	"strconv"
	"testing"
	"time"
)

func TestWriteSingleProxyConfig_StructureAndCleanup(t *testing.T) {
	ob := map[string]any{
		"tag": "proxy", "type": "shadowsocks",
		"server": "1.2.3.4", "server_port": 8388,
		"method": "aes-256-gcm", "password": "x",
	}
	path, err := writeSingleProxyConfig(ob, 12345)
	if err != nil {
		t.Fatalf("write: %v", err)
	}
	defer os.Remove(path)

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read back: %v", err)
	}
	var cfg struct {
		Inbounds  []map[string]any `json:"inbounds"`
		Outbounds []map[string]any `json:"outbounds"`
		Route     struct {
			Rules []map[string]any `json:"rules"`
			Final string           `json:"final"`
		} `json:"route"`
	}
	if err := json.Unmarshal(raw, &cfg); err != nil {
		t.Fatalf("unmarshal: %v\n%s", err, raw)
	}
	if len(cfg.Inbounds) != 1 || cfg.Inbounds[0]["listen_port"].(float64) != 12345 {
		t.Errorf("inbound port wrong: %+v", cfg.Inbounds)
	}
	if len(cfg.Outbounds) != 2 {
		t.Fatalf("outbounds=%d, want 2 (proxy + direct)", len(cfg.Outbounds))
	}
	if cfg.Route.Final != "direct" {
		t.Errorf("final=%q, want direct", cfg.Route.Final)
	}
	if len(cfg.Route.Rules) != 1 || cfg.Route.Rules[0]["outbound"] != "proxy" {
		t.Errorf("route rule wrong: %+v", cfg.Route.Rules)
	}
}

func TestPickFreePort_Returns(t *testing.T) {
	p, err := pickFreePort()
	if err != nil {
		t.Fatalf("pickFreePort: %v", err)
	}
	if p <= 1024 || p > 65535 {
		t.Errorf("port %d out of ephemeral range", p)
	}
}

func TestWaitPort_OpensAfterListen(t *testing.T) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer l.Close()
	addr := l.Addr().String()

	if !waitPort(addr, 500*time.Millisecond) {
		t.Error("waitPort should return true for an open listener")
	}

	// A port nothing's listening on (use pickFreePort to grab one then
	// close it; next-millisecond nothing listens).
	p, _ := pickFreePort()
	if waitPort(net.JoinHostPort("127.0.0.1", strconv.Itoa(p)), 300*time.Millisecond) {
		t.Error("waitPort should time out on a closed port")
	}
}

func TestLaunch_RejectsBadURI(t *testing.T) {
	_, err := Launch(context.Background(), "not-a-uri", Config{})
	if err == nil {
		t.Fatal("expected parse error")
	}
}

// TestLaunch_EndToEnd is a lightweight integration test that starts
// sing-box against a deliberately unreachable SS node and asserts
// Launch + Stop succeed at the process-orchestration level (config
// accepted, port opened, Stop kills cleanly). The proxy target is
// TEST-NET-1 so no real traffic leaves.
//
// Skipped when sing-box is absent so CI environments without it still
// go green.
func TestLaunch_EndToEnd(t *testing.T) {
	if _, err := exec.LookPath("sing-box"); err != nil {
		t.Skip("sing-box not on PATH")
	}
	uri := "ss://YWVzLTI1Ni1nY206cHc=@192.0.2.1:8388#test"
	p, err := Launch(context.Background(), uri, Config{StartupTimeout: 3 * time.Second})
	if err != nil {
		t.Fatalf("Launch: %v", err)
	}
	if p.SocksAddr == "" {
		t.Fatal("SocksAddr empty")
	}
	// The inbound should be accepting — connect + close proves it.
	c, err := net.DialTimeout("tcp", p.SocksAddr, 1*time.Second)
	if err != nil {
		p.Stop()
		t.Fatalf("dial inbound: %v", err)
	}
	_ = c.Close()
	p.Stop()

	// After Stop the port should not reopen.
	if waitPort(p.SocksAddr, 500*time.Millisecond) {
		t.Error("port still open after Stop")
	}
}
