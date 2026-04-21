// Package singbox spawns a minimal sing-box subprocess that exposes a
// single proxy node as a local SOCKS5 inbound. It is intentionally
// scoped to on-demand probe workflows (proxykit unlock --via) — not a
// persistent proxy server, and not a batch verifier.
//
// sing-box itself is a runtime dependency: callers must have the
// `sing-box` binary on $PATH (or pass an absolute path via BinPath).
// proxykit's go.mod stays zero-runtime-deps; the launcher just exec's
// whatever is installed.
package singbox

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"strconv"
	"syscall"
	"time"

	upstream "github.com/Au1rxx/free-vpn-subscriptions/pkg/emit"
	"github.com/Au1rxx/free-vpn-subscriptions/pkg/node"
)

// Config controls a single launcher instance.
type Config struct {
	// BinPath is the sing-box binary. Empty → "sing-box" on $PATH.
	BinPath string
	// ListenPort is the SOCKS5 listen port on 127.0.0.1. Zero picks a
	// free ephemeral port via net.Listen + close.
	ListenPort int
	// StartupTimeout caps the wait for the inbound to become reachable.
	// Zero → 4 seconds.
	StartupTimeout time.Duration
}

// Proc is a running sing-box subprocess.
type Proc struct {
	// SocksAddr is the "127.0.0.1:port" SOCKS5 address clients should
	// use. Populated before Start returns.
	SocksAddr string

	cmd     *exec.Cmd
	cfgPath string
	cancel  context.CancelFunc
}

// Launch parses uri into a node, writes a minimal sing-box config
// (mixed inbound on ListenPort → proxy outbound), starts the binary,
// and waits for the inbound to accept TCP. Callers MUST call Stop to
// terminate the subprocess and delete the config file.
func Launch(ctx context.Context, uri string, cfg Config) (*Proc, error) {
	n, err := node.ParseURI(uri)
	if err != nil {
		return nil, fmt.Errorf("parse uri: %w", err)
	}
	if !n.Valid() {
		return nil, fmt.Errorf("parsed node is invalid: %s %s:%d", n.Protocol, n.Server, n.Port)
	}

	bin := cfg.BinPath
	if bin == "" {
		bin = "sing-box"
	}
	if _, err := exec.LookPath(bin); err != nil {
		return nil, fmt.Errorf("sing-box binary not found — install from https://sing-box.sagernet.org/ and ensure it is on PATH (looked up %q)", bin)
	}

	port := cfg.ListenPort
	if port == 0 {
		port, err = pickFreePort()
		if err != nil {
			return nil, fmt.Errorf("pick free port: %w", err)
		}
	}

	ob := upstream.SingboxOutbound(n, "proxy")
	if ob == nil {
		return nil, fmt.Errorf("sing-box has no mapping for protocol %q", n.Protocol)
	}

	cfgPath, err := writeSingleProxyConfig(ob, port)
	if err != nil {
		return nil, err
	}

	// Pre-flight: catch schema/field errors before we commit to a full boot.
	if out, err := exec.Command(bin, "check", "-c", cfgPath).CombinedOutput(); err != nil {
		_ = os.Remove(cfgPath)
		return nil, fmt.Errorf("sing-box config check failed: %w (%s)", err, truncate(string(out), 400))
	}

	startupTO := cfg.StartupTimeout
	if startupTO == 0 {
		startupTO = 4 * time.Second
	}

	runCtx, cancel := context.WithCancel(ctx)
	cmd := exec.CommandContext(runCtx, bin, "run", "-c", cfgPath)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := cmd.Start(); err != nil {
		cancel()
		_ = os.Remove(cfgPath)
		return nil, fmt.Errorf("start sing-box: %w", err)
	}

	addr := net.JoinHostPort("127.0.0.1", strconv.Itoa(port))
	if !waitPort(addr, startupTO) {
		_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		_, _ = cmd.Process.Wait()
		cancel()
		_ = os.Remove(cfgPath)
		return nil, fmt.Errorf("sing-box did not open %s within %s (config: %s)", addr, startupTO, cfgPath)
	}

	return &Proc{
		SocksAddr: addr,
		cmd:       cmd,
		cfgPath:   cfgPath,
		cancel:    cancel,
	}, nil
}

// Stop terminates the sing-box subprocess (SIGKILLs the whole process
// group) and removes the temp config file. Safe to call multiple times.
func (p *Proc) Stop() {
	if p == nil {
		return
	}
	if p.cmd != nil && p.cmd.Process != nil {
		_ = syscall.Kill(-p.cmd.Process.Pid, syscall.SIGKILL)
		_, _ = p.cmd.Process.Wait()
		p.cmd = nil
	}
	if p.cancel != nil {
		p.cancel()
		p.cancel = nil
	}
	if p.cfgPath != "" {
		_ = os.Remove(p.cfgPath)
		p.cfgPath = ""
	}
}

func writeSingleProxyConfig(proxyOutbound map[string]any, port int) (string, error) {
	cfg := map[string]any{
		"log": map[string]any{"disabled": true},
		"inbounds": []map[string]any{
			{"type": "mixed", "tag": "in", "listen": "127.0.0.1", "listen_port": port},
		},
		"outbounds": []map[string]any{
			proxyOutbound,
			{"type": "direct", "tag": "direct"},
		},
		"route": map[string]any{
			"rules": []map[string]any{{"inbound": "in", "outbound": "proxy"}},
			"final": "direct",
		},
	}
	raw, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return "", err
	}
	f, err := os.CreateTemp("", "proxykit-singbox-*.json")
	if err != nil {
		return "", err
	}
	if _, err := f.Write(raw); err != nil {
		_ = f.Close()
		_ = os.Remove(f.Name())
		return "", err
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(f.Name())
		return "", err
	}
	return f.Name(), nil
}

func pickFreePort() (int, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer l.Close()
	addr, ok := l.Addr().(*net.TCPAddr)
	if !ok {
		return 0, errors.New("listener returned non-TCP addr")
	}
	return addr.Port, nil
}

func waitPort(addr string, within time.Duration) bool {
	deadline := time.Now().Add(within)
	for time.Now().Before(deadline) {
		c, err := net.DialTimeout("tcp", addr, 200*time.Millisecond)
		if err == nil {
			_ = c.Close()
			return true
		}
		time.Sleep(100 * time.Millisecond)
	}
	return false
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
