package tests

import (
	"sync"
	"testing"
	"time"

	"github.com/trainking/lulu/network"
	"github.com/trainking/lulu/session"
	"google.golang.org/protobuf/proto"
)

// mockConn implements network.Conn for testing
type mockConn struct {
	readPackets []network.Packet
	readIdx     int
	writtenMsgs [][]byte
	closed      bool
	mu          sync.Mutex
}

func (m *mockConn) ReadPacket() (network.Packet, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.readIdx >= len(m.readPackets) {
		return nil, network.ErrConnClosing
	}
	p := m.readPackets[m.readIdx]
	m.readIdx++
	return p, nil
}

func (m *mockConn) WritePacket(p network.Packet) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.writtenMsgs = append(m.writtenMsgs, p.Serialize())
	return nil
}

func (m *mockConn) GetRealIP() string { return "127.0.0.1:12345" }

func (m *mockConn) Close() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.closed = true
}

// mockSessionCallback implements session.SessionCallback for testing
type mockSessionCallback struct {
	onConnect    func(*session.Session)
	onMessage    func(*session.Session, network.Packet)
	onDisconnect func(*session.Session)
	getMsgOpCode func(proto.Message) (uint16, error)
}

func (m *mockSessionCallback) OnConnect(s *session.Session) {
	if m.onConnect != nil {
		m.onConnect(s)
	}
}
func (m *mockSessionCallback) OnMessage(s *session.Session, p network.Packet) {
	if m.onMessage != nil {
		m.onMessage(s, p)
	}
}
func (m *mockSessionCallback) OnDisconnect(s *session.Session) {
	if m.onDisconnect != nil {
		m.onDisconnect(s)
	}
}
func (m *mockSessionCallback) GetMsgOpCode(msg proto.Message) (uint16, error) {
	if m.getMsgOpCode != nil {
		return m.getMsgOpCode(msg)
	}
	return 0, nil
}

func TestNewSession(t *testing.T) {
	conn := &mockConn{}
	cb := &mockSessionCallback{}
	s := session.NewSession(conn, cb)

	if s.ID == 0 {
		t.Error("session ID should not be zero")
	}
	if s.Conn != conn {
		t.Error("session should hold the connection")
	}
}

func TestSessionSetUserID(t *testing.T) {
	conn := &mockConn{}
	cb := &mockSessionCallback{}
	s := session.NewSession(conn, cb)

	if s.IsValid() {
		t.Error("session should not be valid before SetUserID")
	}

	s.SetUserID(42)
	select {
	case uid := <-s.WaitValid():
		if uid != 42 {
			t.Errorf("WaitValid returned %d, want %d", uid, 42)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for valid signal")
	}

	if !s.IsValid() {
		t.Error("session should be valid after SetUserID")
	}
	if s.UserID != 42 {
		t.Errorf("UserID = %d, want %d", s.UserID, 42)
	}
}

func TestSessionDestroy(t *testing.T) {
	conn := &mockConn{}
	disconnectCalled := false
	cb := &mockSessionCallback{
		onDisconnect: func(s *session.Session) {
			disconnectCalled = true
		},
	}
	s := session.NewSession(conn, cb)

	s.Destroy()

	if !conn.closed {
		t.Error("connection should be closed on destroy")
	}
	if !disconnectCalled {
		t.Error("disconnect callback should be called")
	}
}

func TestSessionDestroyIdempotent(t *testing.T) {
	conn := &mockConn{}
	callCount := 0
	cb := &mockSessionCallback{
		onDisconnect: func(s *session.Session) {
			callCount++
		},
	}
	s := session.NewSession(conn, cb)

	s.Destroy()
	s.Destroy()
	s.Destroy()

	if callCount != 1 {
		t.Errorf("disconnect callback called %d times, want 1", callCount)
	}
}

func TestSessionSetUserIDAfterDestroy(t *testing.T) {
	conn := &mockConn{}
	cb := &mockSessionCallback{}
	s := session.NewSession(conn, cb)

	s.Destroy()

	// SetUserID after destroy should not block or panic
	s.SetUserID(99)
	if s.IsValid() {
		t.Error("session should not become valid after destroy")
	}
}

func TestCheckFloodDisabled(t *testing.T) {
	conn := &mockConn{}
	cb := &mockSessionCallback{}
	s := session.NewSession(conn, cb)

	// limit=0 disables flood check
	for i := 0; i < 10000; i++ {
		if s.CheckFlood(0) {
			t.Errorf("CheckFlood(0) should always return false, failed at iteration %d", i)
		}
	}
}

func TestCheckFloodLimit(t *testing.T) {
	conn := &mockConn{}
	cb := &mockSessionCallback{}
	s := session.NewSession(conn, cb)

	limit := 10
	for i := 0; i < limit; i++ {
		if s.CheckFlood(limit) {
			t.Errorf("message %d should not trigger flood", i+1)
		}
	}

	// Next message should trigger flood
	if !s.CheckFlood(limit) {
		t.Error("message should trigger flood after exceeding limit")
	}
}

func TestCheckFloodNegativeLimit(t *testing.T) {
	conn := &mockConn{}
	cb := &mockSessionCallback{}
	s := session.NewSession(conn, cb)

	for i := 0; i < 1000; i++ {
		if s.CheckFlood(-1) {
			t.Error("CheckFlood with negative limit should return false")
		}
	}
}

func TestSessionManagerAddAndGet(t *testing.T) {
	mgr := session.NewSessionManager()
	defer mgr.Close()

	conn := &mockConn{}
	cb := &mockSessionCallback{}
	s := session.NewSession(conn, cb)
	s.SetUserID(100)

	mgr.Add(s)
	time.Sleep(50 * time.Millisecond)

	got, ok := mgr.Get(100)
	if !ok {
		t.Fatal("session should be found after Add")
	}
	if got.UserID != 100 {
		t.Errorf("UserID = %d, want %d", got.UserID, 100)
	}
}

func TestSessionManagerDel(t *testing.T) {
	mgr := session.NewSessionManager()
	defer mgr.Close()

	conn := &mockConn{}
	cb := &mockSessionCallback{}
	s := session.NewSession(conn, cb)
	s.SetUserID(200)

	mgr.Add(s)
	time.Sleep(50 * time.Millisecond)

	if _, ok := mgr.Get(200); !ok {
		t.Fatal("session should exist before del")
	}

	mgr.Del(s)
	time.Sleep(50 * time.Millisecond)

	if _, ok := mgr.Get(200); ok {
		t.Error("session should be removed after del")
	}
}

func TestSessionManagerReplaceDuplicateUserID(t *testing.T) {
	mgr := session.NewSessionManager()
	defer mgr.Close()

	conn1 := &mockConn{}
	cb1 := &mockSessionCallback{}
	s1 := session.NewSession(conn1, cb1)
	s1.SetUserID(300)
	mgr.Add(s1)
	time.Sleep(50 * time.Millisecond)

	conn2 := &mockConn{}
	cb2 := &mockSessionCallback{}
	s2 := session.NewSession(conn2, cb2)
	s2.SetUserID(300)
	mgr.Add(s2)
	time.Sleep(50 * time.Millisecond)

	got, ok := mgr.Get(300)
	if !ok {
		t.Fatal("session should exist")
	}
	if got.ID != s2.ID {
		t.Error("new session should replace old one with same UserID")
	}
	conn1.mu.Lock()
	conn1Closed := conn1.closed
	conn1.mu.Unlock()
	if !conn1Closed {
		t.Error("old connection should be closed on replacement")
	}
}

func TestSessionManagerLen(t *testing.T) {
	mgr := session.NewSessionManager()
	defer mgr.Close()

	if mgr.Len() != 0 {
		t.Errorf("initial len = %d, want 0", mgr.Len())
	}

	for i := 0; i < 5; i++ {
		conn := &mockConn{}
		cb := &mockSessionCallback{}
		s := session.NewSession(conn, cb)
		s.SetUserID(uint64(1000 + i))
		mgr.Add(s)
	}
	time.Sleep(100 * time.Millisecond)

	if mgr.Len() != 5 {
		t.Errorf("len = %d, want 5", mgr.Len())
	}
}

func TestSessionManagerClose(t *testing.T) {
	mgr := session.NewSessionManager()
	mgr.Close()
	// Closing twice should not panic
	mgr.Close()
}

func TestSessionManagerConcurrentClose(t *testing.T) {
	mgr := session.NewSessionManager()

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			mgr.Close()
		}()
	}
	wg.Wait()

	s := session.NewSession(&mockConn{}, &mockSessionCallback{})
	s.SetUserID(500)
	mgr.Add(s)
	mgr.Del(s)
}
