package lulu

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/trainking/lulu/network"
	"github.com/trainking/lulu/session"
)

type internalTestListener struct {
	conn     network.Conn
	accepted bool
	closed   chan struct{}
	mu       sync.Mutex
}

func newInternalTestListener(conn network.Conn) *internalTestListener {
	return &internalTestListener{
		conn:   conn,
		closed: make(chan struct{}),
	}
}

func (l *internalTestListener) Accept() (network.Conn, error) {
	l.mu.Lock()
	if !l.accepted {
		l.accepted = true
		conn := l.conn
		l.mu.Unlock()
		return conn, nil
	}
	l.mu.Unlock()

	<-l.closed
	return nil, errors.New("listener closed")
}

func (l *internalTestListener) Close() {
	select {
	case <-l.closed:
	default:
		close(l.closed)
	}
}

type internalTestConn struct {
	closed chan struct{}
}

func newInternalTestConn() *internalTestConn {
	return &internalTestConn{closed: make(chan struct{})}
}

func (c *internalTestConn) ReadPacket() (network.Packet, error) {
	<-c.closed
	return nil, network.ErrConnClosing
}

func (c *internalTestConn) WritePacket(network.Packet) error {
	return nil
}

func (c *internalTestConn) GetRealIP() string {
	return "127.0.0.1:12345"
}

func (c *internalTestConn) Close() {
	select {
	case <-c.closed:
	default:
		close(c.closed)
	}
}

func TestValidTimerAddsAlreadyValidSession(t *testing.T) {
	app := New(&Config{
		Address:      "127.0.0.1:0",
		NetWork:      network.TcpNet,
		ConnMax:      10,
		ValidTimeout: 0,
		HeartLimit:   100,
	})
	app.listener.Close()

	userID := uint64(4242)
	app.SetConnectEvent(func(s *session.Session) error {
		atomic.StoreUint64(&s.UserID, userID)
		return nil
	})
	app.listener = newInternalTestListener(newInternalTestConn())

	done := make(chan struct{})
	go func() {
		app.run()
		close(done)
	}()
	defer func() {
		app.Destroy()
		select {
		case <-done:
		case <-time.After(time.Second):
			t.Fatal("app.run did not stop")
		}
	}()

	deadline := time.After(time.Second)
	for {
		if _, ok := app.SessionManager.Get(userID); ok {
			return
		}
		select {
		case <-deadline:
			t.Fatal("valid session was not added to SessionManager after timer branch")
		case <-time.After(10 * time.Millisecond):
		}
	}
}

type receiveTestConn struct {
	closed chan struct{}
}

func newReceiveTestConn() *receiveTestConn {
	return &receiveTestConn{closed: make(chan struct{})}
}

func (c *receiveTestConn) ReadPacket() (network.Packet, error) {
	select {
	case <-c.closed:
		return nil, network.ErrConnClosing
	default:
		return network.PackingOpcode(1, nil), nil
	}
}

func (c *receiveTestConn) WritePacket(network.Packet) error {
	return nil
}

func (c *receiveTestConn) GetRealIP() string {
	return "127.0.0.1:12345"
}

func (c *receiveTestConn) Close() {
	select {
	case <-c.closed:
	default:
		close(c.closed)
	}
}

func TestClientReceiveStopsWhenQueueFullAndClosed(t *testing.T) {
	c := &Client{
		Conn:        newReceiveTestConn(),
		closeChan:   make(chan struct{}),
		receiveChan: make(chan network.Packet, 1),
	}

	done := make(chan struct{})
	go func() {
		c.receive()
		close(done)
	}()

	deadline := time.After(time.Second)
	for len(c.receiveChan) == 0 {
		select {
		case <-deadline:
			t.Fatal("receive queue was not filled")
		case <-time.After(10 * time.Millisecond):
		}
	}

	c.Close()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("receive loop did not stop after Close")
	}
}
