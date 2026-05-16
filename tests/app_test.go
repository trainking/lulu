package tests

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/trainking/lulu"
	"github.com/trainking/lulu/network"
	"github.com/trainking/lulu/session"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func TestAppNew(t *testing.T) {
	cfg := &lulu.Config{
		Address: "127.0.0.1:0",
		NetWork: "tcp",
	}
	app := lulu.New(cfg)

	if app.Config != cfg {
		t.Error("app should hold the config")
	}
	if app.Route() == nil {
		t.Error("Route() should return non-nil RouterManager")
	}
	if app.SessionManager == nil {
		t.Error("SessionManager should be initialized")
	}
}

func TestAppSetConnectEvent(t *testing.T) {
	app := lulu.New(&lulu.Config{
		Address: "127.0.0.1:0",
		NetWork: "tcp",
	})

	var called bool
	var calledSessionID int64
	app.SetConnectEvent(func(s *session.Session) error {
		called = true
		calledSessionID = s.ID
		return nil
	})

	s := session.NewSession(&mockConn{}, app)
	app.OnConnect(s)

	if !called {
		t.Error("connect event should be called")
	}
	if calledSessionID != s.ID {
		t.Errorf("session ID = %d, want %d", calledSessionID, s.ID)
	}
}

func TestAppSetDisconnectEvent(t *testing.T) {
	app := lulu.New(&lulu.Config{
		Address: "127.0.0.1:0",
		NetWork: "tcp",
	})

	var called bool
	app.SetDisconnectEvent(func(s *session.Session) error {
		called = true
		return nil
	})

	s := session.NewSession(&mockConn{}, app)
	app.OnDisconnect(s)

	if !called {
		t.Error("disconnect event should be called")
	}
}

func TestAppGetMsgOpCode(t *testing.T) {
	app := lulu.New(&lulu.Config{
		Address: "127.0.0.1:0",
		NetWork: "tcp",
	})

	msg := &wrapperspb.StringValue{Value: "test"}
	app.Route().Register(msg, uint16(42))

	op, err := app.GetMsgOpCode(msg)
	if err != nil {
		t.Fatalf("GetMsgOpCode failed: %v", err)
	}
	if op != 42 {
		t.Errorf("opcode = %d, want %d", op, 42)
	}
}

func TestAppOnMessageFlood(t *testing.T) {
	app := lulu.New(&lulu.Config{
		Address:    "127.0.0.1:0",
		NetWork:    "tcp",
		HeartLimit: 5,
	})

	msg := &wrapperspb.Int32Value{Value: 1}
	handlerCalled := int32(0)
	app.Route().Register(msg, uint16(1), lulu.WithRegisterHandler(
		func(ctx lulu.Context) error {
			atomic.AddInt32(&handlerCalled, 1)
			return nil
		},
	), lulu.WithRegisterIsNoValid(true))

	body, _ := proto.Marshal(msg)

	// Each message uses a fresh packet to avoid sharing across goroutines
	for i := 0; i < 5; i++ {
		s2 := session.NewSession(&mockConn{}, app)
		p := network.PackingOpcode(1, body)
		app.OnMessage(s2, p)
	}

	// Send one more — should trigger flood
	s3 := session.NewSession(&mockConn{}, app)
	p := network.PackingOpcode(1, body)
	app.OnMessage(s3, p)

	time.Sleep(200 * time.Millisecond)
}

func TestAppModuleLifecycle(t *testing.T) {
	app := lulu.New(&lulu.Config{
		Address: "127.0.0.1:0",
		NetWork: "tcp",
	})

	var initOrder []string
	var destroyOrder []string

	m1 := &testModule{
		name: "mod-a",
		onInit:    func() { initOrder = append(initOrder, "mod-a") },
		onDestroy: func() { destroyOrder = append(destroyOrder, "mod-a") },
	}
	m2 := &testModule{
		name: "mod-b",
		onInit:    func() { initOrder = append(initOrder, "mod-b") },
		onDestroy: func() { destroyOrder = append(destroyOrder, "mod-b") },
	}

	// Test init order — first registered, first initialized
	m1.OnInit(app)
	m2.OnInit(app)

	if len(initOrder) != 2 {
		t.Fatalf("expected 2 modules initialized, got %d: %v", len(initOrder), initOrder)
	}
	if initOrder[0] != "mod-a" || initOrder[1] != "mod-b" {
		t.Errorf("init order = %v, want [mod-a mod-b]", initOrder)
	}

	// Test destroy order — reverse of init
	m2.OnDestroy()
	m1.OnDestroy()

	if len(destroyOrder) != 2 {
		t.Fatalf("expected 2 modules destroyed, got %d: %v", len(destroyOrder), destroyOrder)
	}
	if destroyOrder[0] != "mod-b" || destroyOrder[1] != "mod-a" {
		t.Errorf("destroy order = %v, want [mod-b mod-a]", destroyOrder)
	}
}

func TestAppDestroyIdempotent(t *testing.T) {
	app := lulu.New(&lulu.Config{
		Address: "127.0.0.1:0",
		NetWork: "tcp",
	})

	app.Destroy()
	app.Destroy()
	app.Destroy()
}

func TestAppAction(t *testing.T) {
	app := lulu.New(&lulu.Config{
		Address: "127.0.0.1:0",
		NetWork: "tcp",
	})

	retMsg := &wrapperspb.StringValue{Value: "response"}
	app.Route().Register(retMsg, uint16(999))

	conn := &mockConn{}
	s := session.NewSession(conn, app)
	s.SetUserID(9999)

	app.SessionManager.Add(s)
	time.Sleep(50 * time.Millisecond)

	app.Action(9999, &wrapperspb.StringValue{Value: "action-test"})
	time.Sleep(50 * time.Millisecond)

	conn.mu.Lock()
	written := len(conn.writtenMsgs)
	conn.mu.Unlock()
	if written == 0 {
		t.Error("Action should have sent a message")
	}
}

func TestAppCallInternalRoute(t *testing.T) {
	app := lulu.New(&lulu.Config{
		Address: "127.0.0.1:0",
		NetWork: "tcp",
	})

	var internalCalled int32
	internalMsg := &wrapperspb.Int32Value{Value: 100}
	app.Route().Register(internalMsg, uint16(555),
		lulu.WithRegisterHandler(func(ctx lulu.Context) error {
			atomic.StoreInt32(&internalCalled, 1)
			return nil
		}),
		lulu.WithRegisterIsInner(true),
	)

	conn := &mockConn{}
	s := session.NewSession(conn, app)
	s.SetUserID(1)

	app.Call(s, &wrapperspb.Int32Value{Value: 100})
	time.Sleep(100 * time.Millisecond)

	if atomic.LoadInt32(&internalCalled) == 0 {
		t.Error("Call should have triggered the internal route handler")
	}
}

func TestAppRunWithModule(t *testing.T) {
	cfg := &lulu.Config{
		Address:          "127.0.0.1:0",
		NetWork:          "websocket",
		WebsocketPath:    "/wse2etest",
		ConnMax:          10,
		ValidTimeout:     5,
		HeartLimit:       1000,
		ConnReadTimeout:  5,
		ConnWriteTimeout: 5,
	}

	app := lulu.New(cfg)

	var initCount int32
	m := &e2eModule{
		name: "e2e",
		onInit: func() {
			atomic.AddInt32(&initCount, 1)
		},
		onLogin: func(ctx lulu.Context) error {
			ctx.Session().SetUserID(1)
			return ctx.Session().Send(&wrapperspb.StringValue{Value: "ok"})
		},
	}

	// Run server in background
	done := make(chan struct{})
	go func() {
		app.Run(m)
		close(done)
	}()

	time.Sleep(300 * time.Millisecond)

	// Module should have been initialized
	if atomic.LoadInt32(&initCount) != 1 {
		t.Errorf("module init count = %d, want 1", initCount)
	}

	// Destroy and wait for clean exit
	app.Destroy()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("app.Run did not return after Destroy")
	}
}

// Test helper types

type testModule struct {
	name      string
	onInit    func()
	onDestroy func()
}

func (m *testModule) Name() string               { return m.name }
func (m *testModule) OnInit(app *lulu.App) error { m.onInit(); return nil }
func (m *testModule) OnDestroy()                 { m.onDestroy() }
func (m *testModule) Route(app *lulu.App)        {}

type e2eModule struct {
	name    string
	onInit  func()
	onLogin func(lulu.Context) error
}

func (m *e2eModule) Name() string { return m.name }
func (m *e2eModule) OnInit(app *lulu.App) error {
	if m.onInit != nil {
		m.onInit()
	}
	loginMsg := &wrapperspb.StringValue{Value: "login"}
	app.Route().Register(loginMsg, uint16(1),
		lulu.WithRegisterHandler(m.onLogin),
		lulu.WithRegisterIsNoValid(true),
	)
	respMsg := &wrapperspb.StringValue{Value: "ok"}
	app.Route().Register(respMsg, uint16(2))
	return nil
}
func (m *e2eModule) OnDestroy()          {}
func (m *e2eModule) Route(app *lulu.App) {}
