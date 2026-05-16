package tests

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/trainking/lulu"
	"github.com/trainking/lulu/network"
	"github.com/trainking/lulu/session"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

// TestConcurrentSessionManagerAddGet tests concurrent add/get/del
func TestConcurrentSessionManagerAddGet(t *testing.T) {
	mgr := session.NewSessionManager()
	defer mgr.Close()

	var wg sync.WaitGroup
	numGoroutines := 20
	numOpsPerGoroutine := 50

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(base int) {
			defer wg.Done()
			for j := 0; j < numOpsPerGoroutine; j++ {
				conn := &mockConn{}
				cb := &mockSessionCallback{}
				s := session.NewSession(conn, cb)
				userID := uint64(base*10000 + j)
				s.SetUserID(userID)
				mgr.Add(s)
			}
		}(i)
	}

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 200; j++ {
				mgr.Get(uint64(j % 500))
			}
		}()
	}

	wg.Wait()
	time.Sleep(100 * time.Millisecond)
	t.Logf("Session count after concurrent ops: %d", mgr.Len())
}

// TestConcurrentRouterLookup tests concurrent route lookups
// Note: routes are registered sequentially (as in real server init), then
// lookups are done concurrently (as in real request handling).
func TestConcurrentRouterLookup(t *testing.T) {
	app := lulu.New(&lulu.Config{
		Address: "127.0.0.1:0",
		NetWork: "tcp",
	})

	handler := func(ctx lulu.Context) error { return nil }

	// Register routes sequentially (real-world init pattern)
	numRoutes := 200
	for i := 0; i < numRoutes; i++ {
		msg := &wrapperspb.Int32Value{Value: int32(i)}
		app.Route().Register(msg, uint16(i), lulu.WithRegisterHandler(handler))
	}

	// Concurrent lookups (real-world request handling pattern)
	var wg sync.WaitGroup
	numGoroutines := 20
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				_, ok := app.Route().GetHandleRouter(uint16(j % numRoutes))
				if !ok && j < numRoutes {
					t.Errorf("route %d not found", j%numRoutes)
				}
			}
		}()
	}

	wg.Wait()
}

// TestConcurrentCheckFlood tests CheckFlood under concurrent access
func TestConcurrentCheckFlood(t *testing.T) {
	conn := &mockConn{}
	cb := &mockSessionCallback{}
	s := session.NewSession(conn, cb)

	limit := 1000
	var triggered int32
	var wg sync.WaitGroup
	numGoroutines := 50
	messagesPerGoroutine := 30

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < messagesPerGoroutine; j++ {
				if s.CheckFlood(limit) {
					atomic.AddInt32(&triggered, 1)
				}
			}
		}()
	}

	wg.Wait()

	totalMessages := numGoroutines * messagesPerGoroutine
	t.Logf("Flood test: %d messages, %d triggered, limit=%d",
		totalMessages, triggered, limit)

	if atomic.LoadInt32(&triggered) == int32(totalMessages) {
		t.Error("all messages triggered flood — limit may not be working")
	}
}

// TestConcurrentSessionDestroy tests destroying sessions concurrently
func TestConcurrentSessionDestroy(t *testing.T) {
	var wg sync.WaitGroup

	for i := 0; i < 50; i++ {
		conn := &mockConn{}
		cb := &mockSessionCallback{}
		s := session.NewSession(conn, cb)

		for j := 0; j < 5; j++ {
			wg.Add(1)
			go func(session *session.Session) {
				defer wg.Done()
				session.Destroy()
			}(s)
		}
	}

	wg.Wait()
}

// TestConcurrentPacketPool tests packet pool under heavy concurrent use
func TestConcurrentPacketPool(t *testing.T) {
	var wg sync.WaitGroup
	numGoroutines := 100
	opsPerGoroutine := 200

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			for j := 0; j < opsPerGoroutine; j++ {
				body := []byte(fmt.Sprintf("g%d-m%d", n, j))
				p := network.PackingOpcode(uint16(j%65536), body)
				_ = p.Serialize()
				_ = p.OpCode()
				_ = p.BodyLen()
				_ = p.Body()
				p.Free()
			}
		}(i)
	}

	wg.Wait()
}

// TestConcurrentAppCreateDestroy tests rapid create/destroy cycles
func TestConcurrentAppCreateDestroy(t *testing.T) {
	var wg sync.WaitGroup

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			app := lulu.New(&lulu.Config{
				Address: "127.0.0.1:0",
				NetWork: "tcp",
			})
			time.Sleep(10 * time.Millisecond)
			app.Destroy()
		}()
	}

	wg.Wait()
}

// Benchmark: SessionManager Add operations
func BenchmarkSessionManagerAdd(b *testing.B) {
	mgr := session.NewSessionManager()
	defer mgr.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		conn := &mockConn{}
		cb := &mockSessionCallback{}
		s := session.NewSession(conn, cb)
		s.SetUserID(uint64(i))
		mgr.Add(s)
	}
	b.StopTimer()
	time.Sleep(50 * time.Millisecond)
}

// Benchmark: SessionManager Get operations
func BenchmarkSessionManagerGet(b *testing.B) {
	mgr := session.NewSessionManager()
	defer mgr.Close()

	for i := 0; i < 1000; i++ {
		conn := &mockConn{}
		cb := &mockSessionCallback{}
		s := session.NewSession(conn, cb)
		s.SetUserID(uint64(i))
		mgr.Add(s)
	}
	time.Sleep(100 * time.Millisecond)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mgr.Get(uint64(i % 1000))
	}
}

// Benchmark: Packet creation and serialization
func BenchmarkPackingOpcode(b *testing.B) {
	body := []byte("benchmark packet body data")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p := network.PackingOpcode(uint16(i%65536), body)
		_ = p.Serialize()
		p.Free()
	}
}

// Benchmark: Router lookup
func BenchmarkRouterLookup(b *testing.B) {
	app := lulu.New(&lulu.Config{
		Address: "127.0.0.1:0",
		NetWork: "tcp",
	})

	handler := func(ctx lulu.Context) error { return nil }
	for i := 0; i < 1000; i++ {
		msg := &wrapperspb.Int32Value{Value: int32(i)}
		app.Route().Register(msg, uint16(i), lulu.WithRegisterHandler(handler))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		app.Route().GetHandleRouter(uint16(i % 1000))
	}
}

// Benchmark: Middleware chain execution
func BenchmarkMiddlewareChain(b *testing.B) {
	mw := func(next lulu.Handler) lulu.Handler {
		return func(ctx lulu.Context) error {
			return next(ctx)
		}
	}
	handler := func(ctx lulu.Context) error { return nil }

	chain := mw(mw(mw(mw(mw(handler)))))

	app := lulu.New(&lulu.Config{Address: "127.0.0.1:0", NetWork: "tcp"})
	ctx := lulu.NewContext(nil, app, &session.Session{}, 0, nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		chain(ctx)
	}
}
