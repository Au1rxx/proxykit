package nodeio

import "testing"

func TestParseURI_Trojan(t *testing.T) {
	uri := "trojan://hunter2@example.com:443?sni=example.com#demo"
	n, err := ParseURI(uri)
	if err != nil {
		t.Fatalf("ParseURI returned error: %v", err)
	}
	if n == nil {
		t.Fatal("ParseURI returned nil node without error")
	}
	if n.Protocol != ProtoTrojan {
		t.Errorf("Protocol = %q, want %q", n.Protocol, ProtoTrojan)
	}
	if n.Server != "example.com" || n.Port != 443 {
		t.Errorf("Server:Port = %s:%d, want example.com:443", n.Server, n.Port)
	}
	if n.Password != "hunter2" {
		t.Errorf("Password = %q, want hunter2", n.Password)
	}
	if !n.Valid() {
		t.Error("Node.Valid() = false, want true")
	}
	if n.Key() == "" {
		t.Error("Node.Key() returned empty string")
	}
}

func TestParseURI_Unknown(t *testing.T) {
	if _, err := ParseURI("not-a-real-scheme://whatever"); err == nil {
		t.Error("expected error for unknown scheme, got nil")
	}
}
