package tests

import (
	"bytes"
	"encoding/binary"
	"io"
	"math"
	"strings"
	"sync"
	"testing"

	"github.com/trainking/lulu/network"
)

func TestPackingOpcode(t *testing.T) {
	body := []byte("hello world")
	opcode := uint16(0xABCD)

	p := network.PackingOpcode(opcode, body)

	if p.OpCode() != opcode {
		t.Errorf("OpCode = %d, want %d", p.OpCode(), opcode)
	}
	if p.BodyLen() != uint16(len(body)) {
		t.Errorf("BodyLen = %d, want %d", p.BodyLen(), len(body))
	}
	if !bytes.Equal(p.Body(), body) {
		t.Errorf("Body = %v, want %v", p.Body(), body)
	}

	serialized := p.Serialize()
	expectedLen := 4 + len(body)
	if len(serialized) != expectedLen {
		t.Errorf("Serialize length = %d, want %d", len(serialized), expectedLen)
	}

	// Verify big-endian header
	gotBodyLen := binary.BigEndian.Uint16(serialized[0:2])
	gotOpCode := binary.BigEndian.Uint16(serialized[2:4])
	if gotBodyLen != uint16(len(body)) {
		t.Errorf("header body length = %d, want %d", gotBodyLen, len(body))
	}
	if gotOpCode != opcode {
		t.Errorf("header opcode = %d, want %d", gotOpCode, opcode)
	}

	p.Free()
}

func TestPackingOpcodeEmptyBody(t *testing.T) {
	p := network.PackingOpcode(0x0001, nil)

	if p.OpCode() != 0x0001 {
		t.Errorf("OpCode = %d, want %d", p.OpCode(), 0x0001)
	}
	if p.BodyLen() != 0 {
		t.Errorf("BodyLen = %d, want 0", p.BodyLen())
	}
	if len(p.Body()) != 0 {
		t.Errorf("Body should be empty, got %v", p.Body())
	}

	serialized := p.Serialize()
	if len(serialized) != 4 {
		t.Errorf("Serialized length = %d, want 4", len(serialized))
	}

	p.Free()
}

func TestPackingOpcodeOpCodeZero(t *testing.T) {
	p := network.PackingOpcode(0, []byte("data"))
	if p.OpCode() != 0 {
		t.Errorf("OpCode should allow 0, got %d", p.OpCode())
	}
	p.Free()
}

func TestPackingReader(t *testing.T) {
	body := []byte("test packet body")
	p := network.PackingOpcode(0x1234, body)
	defer p.Free()

	reader := bytes.NewReader(p.Serialize())
	p2, err := network.PackingReader(reader)
	if err != nil {
		t.Fatalf("PackingReader failed: %v", err)
	}
	defer p2.Free()

	if p2.OpCode() != 0x1234 {
		t.Errorf("OpCode = %d, want %d", p2.OpCode(), 0x1234)
	}
	if p2.BodyLen() != uint16(len(body)) {
		t.Errorf("BodyLen = %d, want %d", p2.BodyLen(), len(body))
	}
	if !bytes.Equal(p2.Body(), body) {
		t.Errorf("Body mismatch: got %v, want %v", p2.Body(), body)
	}
}

func TestPackingReaderEmptyBody(t *testing.T) {
	p := network.PackingOpcode(0x0001, nil)
	defer p.Free()

	reader := bytes.NewReader(p.Serialize())
	p2, err := network.PackingReader(reader)
	if err != nil {
		t.Fatalf("PackingReader failed: %v", err)
	}
	defer p2.Free()

	if p2.BodyLen() != 0 {
		t.Errorf("BodyLen = %d, want 0", p2.BodyLen())
	}
}

func TestPackingReaderTooLarge(t *testing.T) {
	header := make([]byte, 4)
	binary.BigEndian.PutUint16(header[0:2], 0xFFFF)
	binary.BigEndian.PutUint16(header[2:4], 0x0001)

	reader := bytes.NewReader(header)
	// Reader will try to read 65535 bytes but only 4 are available
	_, err := network.PackingReader(reader)
	if err == nil {
		t.Error("expected error reading packet with body larger than available data")
	}
}

func TestPackingReaderEOF(t *testing.T) {
	reader := bytes.NewReader([]byte{0x00, 0x01})
	_, err := network.PackingReader(reader)
	if err != io.ErrUnexpectedEOF {
		t.Errorf("expected EOF error, got %v", err)
	}
}

func TestPackingReaderIncompleteBody(t *testing.T) {
	header := make([]byte, 4)
	binary.BigEndian.PutUint16(header[0:2], 10)
	binary.BigEndian.PutUint16(header[2:4], 0x0001)

	reader := bytes.NewReader(header)
	_, err := network.PackingReader(reader)
	if err == nil {
		t.Error("expected error for incomplete body")
	}
}

func TestPacketPool(t *testing.T) {
	body := []byte("pool test")
	p1 := network.PackingOpcode(1, body)

	s := p1.Serialize()
	if len(s) != 4+len(body) {
		t.Errorf("serialize length = %d", len(s))
	}

	p1.Free()

	p2 := network.PackingOpcode(2, []byte("another"))
	defer p2.Free()

	if p2.OpCode() != 2 {
		t.Errorf("pooled packet opcode = %d, want 2", p2.OpCode())
	}
}

func TestPacketSerializationRoundtrip(t *testing.T) {
	testCases := []struct {
		opcode uint16
		body   []byte
	}{
		{0x0001, []byte{}},
		{0x00FF, []byte{0x01}},
		{0xFFFF, []byte("hello, world!")},
		{0x8000, make([]byte, 1024)},
	}

	for _, tc := range testCases {
		p := network.PackingOpcode(tc.opcode, tc.body)
		reader := bytes.NewReader(p.Serialize())
		p2, err := network.PackingReader(reader)
		if err != nil {
			t.Errorf("roundtrip failed for opcode=%d, bodyLen=%d: %v", tc.opcode, len(tc.body), err)
			p.Free()
			continue
		}
		if p2.OpCode() != tc.opcode {
			t.Errorf("roundtrip opcode mismatch: got %d, want %d", p2.OpCode(), tc.opcode)
		}
		if !bytes.Equal(p2.Body(), tc.body) {
			t.Errorf("roundtrip body mismatch for len=%d", len(tc.body))
		}
		p.Free()
		p2.Free()
	}
}

func TestMaxPacketSize(t *testing.T) {
	if network.MaxPacketSize != math.MaxUint16 {
		t.Errorf("MaxPacketSize = %d, want %d", network.MaxPacketSize, math.MaxUint16)
	}
}

func TestPackingOpcodeTooLarge(t *testing.T) {
	p := network.PackingOpcode(1, make([]byte, network.MaxPacketSize+1))
	if p != nil {
		t.Fatal("expected nil packet for oversized body")
	}
}

func TestParsePacketMalformed(t *testing.T) {
	if _, err := network.ParsePacket([]byte{0x00, 0x01}); err != network.ErrPacketMalformed {
		t.Fatalf("expected ErrPacketMalformed for short packet, got %v", err)
	}

	headerOnly := make([]byte, 4)
	binary.BigEndian.PutUint16(headerOnly[0:2], 1)
	if _, err := network.ParsePacket(headerOnly); err != network.ErrPacketMalformed {
		t.Fatalf("expected ErrPacketMalformed for length mismatch, got %v", err)
	}
}

func TestListenerFactoryTCP(t *testing.T) {
	lf := network.NewListenerFactory("tcp", "127.0.0.1:0", 5, 10)
	listener, err := lf.Generate()
	if err != nil {
		t.Fatalf("Generate TCP listener failed: %v", err)
	}
	defer listener.Close()

	if listener == nil {
		t.Fatal("listener should not be nil")
	}
}

func TestListenerFactoryWebSocket(t *testing.T) {
	lf := network.NewListenerFactory("websocket", "127.0.0.1:0", 5, 10)
	lf.WithUpgradePath("/ws")
	listener, err := lf.Generate()
	if err != nil {
		t.Fatalf("Generate WebSocket listener failed: %v", err)
	}
	defer listener.Close()

	if listener == nil {
		t.Fatal("listener should not be nil")
	}
}

func TestListenerFactoryUnsupportedNetwork(t *testing.T) {
	lf := network.NewListenerFactory("invalid", "127.0.0.1:0", 5, 10)
	_, err := lf.Generate()
	if err == nil {
		t.Error("expected error for unsupported network type")
	}
	if !strings.Contains(err.Error(), "invalid") {
		t.Errorf("error should mention network type, got: %v", err)
	}
}

func TestListenerFactoryKCP(t *testing.T) {
	lf := network.NewListenerFactory("kcp", "127.0.0.1:0", 5, 10)
	lf.WithKcpMode("fast")
	listener, err := lf.Generate()
	if err != nil {
		t.Fatalf("Generate KCP listener failed: %v", err)
	}
	defer listener.Close()

	if listener == nil {
		t.Fatal("listener should not be nil")
	}
}

func TestNetworkConstants(t *testing.T) {
	if network.TcpNet != "tcp" {
		t.Errorf("TcpNet = %q", network.TcpNet)
	}
	if network.KcpNet != "kcp" {
		t.Errorf("KcpNet = %q", network.KcpNet)
	}
	if network.WebSocketNet != "websocket" {
		t.Errorf("WebSocketNet = %q", network.WebSocketNet)
	}
}

func TestPacketConcurrency(t *testing.T) {
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				p := network.PackingOpcode(uint16(n), []byte("concurrent"))
				_ = p.Serialize()
				p.Free()
			}
		}(i)
	}
	wg.Wait()
}
