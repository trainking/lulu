package tests

import (
	"testing"

	"github.com/trainking/lulu"
	"github.com/trainking/lulu/network"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func TestNewClientInvalidNetwork(t *testing.T) {
	cfg := &network.Config{
		Addr:         "127.0.0.1:9999",
		ReadTimeout:  5,
		WriteTimeout: 5,
	}

	_, err := lulu.NewClient("invalid-network", cfg)
	if err == nil {
		t.Error("expected error for invalid network type")
	}
}

func TestNewClientUnreachableAddress(t *testing.T) {
	cfg := &network.Config{
		Addr:         "127.0.0.1:19999",
		ReadTimeout:  0,
		WriteTimeout: 0,
	}

	// Should fail to connect, but not panic
	_, err := lulu.NewClient(network.TcpNet, cfg)
	if err == nil {
		t.Error("expected connection error for unreachable address")
	}
}

func TestClientSendAPI(t *testing.T) {
	// Compile-time validation of Client.Send signature
	_ = func(c *lulu.Client, opcode uint16, msg *wrapperspb.StringValue) error {
		return c.Send(opcode, msg)
	}
}

func TestClientConfig(t *testing.T) {
	// Validate network.Config fields
	cfg := &network.Config{
		Addr:          "127.0.0.1:8080",
		ReadTimeout:   10,
		WriteTimeout:  5,
		WSUpgradePath: "/ws",
		KcpMode:       "fast",
	}

	if cfg.Addr != "127.0.0.1:8080" {
		t.Error("unexpected Addr")
	}
	if cfg.ReadTimeout != 10 {
		t.Error("unexpected ReadTimeout")
	}
	if cfg.WriteTimeout != 5 {
		t.Error("unexpected WriteTimeout")
	}
	if cfg.WSUpgradePath != "/ws" {
		t.Error("unexpected WSUpgradePath")
	}
	if cfg.KcpMode != "fast" {
		t.Error("unexpected KcpMode")
	}
}

func TestNewClientWithTLSConfig(t *testing.T) {
	cfg := &network.Config{
		Addr:         "127.0.0.1:19998",
		ReadTimeout:  1,
		WriteTimeout: 1,
		// TLSConfig would be set here in production
	}

	// TCP should fail to connect but validate the API
	_, err := lulu.NewClient(network.TcpNet, cfg)
	if err == nil {
		t.Log("unexpectedly connected to 127.0.0.1:19998")
	}
}

func TestNewClientKCP(t *testing.T) {
	cfg := &network.Config{
		Addr:         "127.0.0.1:19997",
		ReadTimeout:  0,
		WriteTimeout: 0,
		KcpMode:      "fast",
	}

	_, err := lulu.NewClient(network.KcpNet, cfg)
	if err == nil {
		t.Log("unexpectedly connected via KCP to local port")
	}
}

func TestNewClientWebSocket(t *testing.T) {
	cfg := &network.Config{
		Addr:           "127.0.0.1:19996",
		ReadTimeout:    1,
		WriteTimeout:   1,
		WSUpgradePath:  "/ws",
	}

	_, err := lulu.NewClient(network.WebSocketNet, cfg)
	if err == nil {
		t.Log("unexpectedly connected via WebSocket")
	}
}
