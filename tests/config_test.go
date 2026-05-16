package tests

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/trainking/lulu"
)

func TestReadConfigYaml(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.yaml")

	yamlContent := `
Version: "1.0.0"
Address: "0.0.0.0:9000"
Network: "tcp"
ConnMax: 500
HeartLimit: 50
`
	if err := os.WriteFile(path, []byte(yamlContent), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := lulu.ReadConfigYaml(path)
	if err != nil {
		t.Fatalf("ReadConfigYaml failed: %v", err)
	}

	if cfg.Version != "1.0.0" {
		t.Errorf("Version = %q, want %q", cfg.Version, "1.0.0")
	}
	if cfg.Address != "0.0.0.0:9000" {
		t.Errorf("Address = %q, want %q", cfg.Address, "0.0.0.0:9000")
	}
	if cfg.NetWork != "tcp" {
		t.Errorf("Network = %q, want %q", cfg.NetWork, "tcp")
	}
	if cfg.ConnMax != 500 {
		t.Errorf("ConnMax = %d, want %d", cfg.ConnMax, 500)
	}
	if cfg.HeartLimit != 50 {
		t.Errorf("HeartLimit = %d, want %d", cfg.HeartLimit, 50)
	}
}

func TestConfigDefaults(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "minimal.yaml")

	// Only set required fields, omit the rest to test defaults
	yamlContent := `
Version: "1.0.0"
Address: "127.0.0.1:8080"
Network: "kcp"
`
	if err := os.WriteFile(path, []byte(yamlContent), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := lulu.ReadConfigYaml(path)
	if err != nil {
		t.Fatalf("ReadConfigYaml failed: %v", err)
	}

	if cfg.WebsocketPath != "/ws" {
		t.Errorf("WebsocketPath default = %q, want %q", cfg.WebsocketPath, "/ws")
	}
	if cfg.KcpMode != "fast" {
		t.Errorf("KcpMode default = %q, want %q", cfg.KcpMode, "fast")
	}
	if cfg.ConnReadTimeout != 10 {
		t.Errorf("ConnReadTimeout default = %d, want %d", cfg.ConnReadTimeout, 10)
	}
	if cfg.ConnWriteTimeout != 5 {
		t.Errorf("ConnWriteTimeout default = %d, want %d", cfg.ConnWriteTimeout, 5)
	}
	if cfg.ConnMax != 10000 {
		t.Errorf("ConnMax default = %d, want %d", cfg.ConnMax, 10000)
	}
	if cfg.ValidTimeout != 10 {
		t.Errorf("ValidTimeout default = %d, want %d", cfg.ValidTimeout, 10)
	}
	if cfg.HeartLimit != 100 {
		t.Errorf("HeartLimit default = %d, want %d", cfg.HeartLimit, 100)
	}
}

func TestConfigDefaultsOverwriteZero(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "zero.yaml")

	// Note: defaultValue() overwrites zero values since it can't distinguish
	// "unset" from "explicitly set to 0". HeartLimit=0 becomes default 100.
	yamlContent := `
Version: "1.0.0"
Address: "127.0.0.1:8080"
Network: "tcp"
HeartLimit: 0
`
	if err := os.WriteFile(path, []byte(yamlContent), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := lulu.ReadConfigYaml(path)
	if err != nil {
		t.Fatalf("ReadConfigYaml failed: %v", err)
	}

	// Zero values are overwritten by defaults
	if cfg.HeartLimit != 100 {
		t.Errorf("HeartLimit=0 is overwritten to default, got %d", cfg.HeartLimit)
	}
}

func TestConfigTLS(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tls.yaml")

	yamlContent := `
Version: "1.0.0"
Address: "127.0.0.1:8443"
Network: "websocket"
TLS:
  CertFile: "/path/to/cert.pem"
  KeyFile: "/path/to/key.pem"
`
	if err := os.WriteFile(path, []byte(yamlContent), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := lulu.ReadConfigYaml(path)
	if err != nil {
		t.Fatalf("ReadConfigYaml failed: %v", err)
	}

	if cfg.TLS == nil {
		t.Fatal("TLS config should not be nil")
	}
	if cfg.TLS.CertFile != "/path/to/cert.pem" {
		t.Errorf("CertFile = %q", cfg.TLS.CertFile)
	}
	if cfg.TLS.KeyFile != "/path/to/key.pem" {
		t.Errorf("KeyFile = %q", cfg.TLS.KeyFile)
	}
}

func TestReadConfigYamlFileNotFound(t *testing.T) {
	_, err := lulu.ReadConfigYaml("/nonexistent/path/config.yaml")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestLoadDefaultAppConfigPanic(t *testing.T) {
	// Since configs/lulu.yaml likely doesn't exist in tests/,
	// LoadDefaultAppConfig should panic
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic from LoadDefaultAppConfig when file missing")
		}
	}()
	lulu.LoadDefaultAppConfig()
}
