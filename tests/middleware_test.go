package tests

import (
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/trainking/lulu"
	"github.com/trainking/lulu/network"
	"github.com/trainking/lulu/session"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func TestMiddlewareChaining(t *testing.T) {
	order := make([]string, 0)

	mw1 := func(next lulu.Handler) lulu.Handler {
		return func(ctx lulu.Context) error {
			order = append(order, "mw1-before")
			err := next(ctx)
			order = append(order, "mw1-after")
			return err
		}
	}
	mw2 := func(next lulu.Handler) lulu.Handler {
		return func(ctx lulu.Context) error {
			order = append(order, "mw2-before")
			err := next(ctx)
			order = append(order, "mw2-after")
			return err
		}
	}

	handler := func(ctx lulu.Context) error {
		order = append(order, "handler")
		return nil
	}

	// Build chain: mw1(mw2(handler))
	h := mw1(mw2(handler))

	app := lulu.New(&lulu.Config{Address: "127.0.0.1:0", NetWork: "tcp"})
	ctx := lulu.NewContext(nil, app, &session.Session{}, 0, nil)
	if err := h(ctx); err != nil {
		t.Fatalf("handler chain failed: %v", err)
	}

	expected := []string{"mw1-before", "mw2-before", "handler", "mw2-after", "mw1-after"}
	if len(order) != len(expected) {
		t.Fatalf("order length = %d, want %d", len(order), len(expected))
	}
	for i, v := range expected {
		if order[i] != v {
			t.Errorf("order[%d] = %q, want %q", i, order[i], v)
		}
	}
}

func TestMiddlewareShortCircuit(t *testing.T) {
	errSentinel := errors.New("short circuit")

	mw := func(next lulu.Handler) lulu.Handler {
		return func(ctx lulu.Context) error {
			return errSentinel
		}
	}

	handlerCalled := false
	handler := func(ctx lulu.Context) error {
		handlerCalled = true
		return nil
	}

	h := mw(handler)
	app := lulu.New(&lulu.Config{Address: "127.0.0.1:0", NetWork: "tcp"})
	ctx := lulu.NewContext(nil, app, &session.Session{}, 0, nil)

	err := h(ctx)
	if err != errSentinel {
		t.Errorf("expected sentinel error, got %v", err)
	}
	if handlerCalled {
		t.Error("handler should not have been called after short circuit")
	}
}

func TestMiddlewareValidSession(t *testing.T) {
	mw := lulu.MiddlewareValidSession()

	handlerCalled := false
	handler := func(ctx lulu.Context) error {
		handlerCalled = true
		return nil
	}

	h := mw(handler)

	// Test with invalid session (UserID = 0, session created without channels)
	s := &session.Session{}
	app := lulu.New(&lulu.Config{Address: "127.0.0.1:0", NetWork: "tcp"})
	ctx := lulu.NewContext(nil, app, s, 0, nil)

	err := h(ctx)
	if err != lulu.ErrSessionInvalid {
		t.Errorf("expected ErrSessionInvalid, got %v", err)
	}
	if handlerCalled {
		t.Error("handler should not be called when session is invalid")
	}

	// Test with valid session — use NewSession so channels are initialized
	handlerCalled = false
	conn := &mockConn{}
	cb := &mockSessionCallback{}
	s2 := session.NewSession(conn, cb)
	s2.SetUserID(12345)

	ctx2 := lulu.NewContext(nil, app, s2, 0, nil)
	err = h(ctx2)
	if err != nil {
		t.Errorf("unexpected error with valid session: %v", err)
	}
	if !handlerCalled {
		t.Error("handler should be called when session is valid")
	}
}

func TestValidSessionRunsBeforeCustomMiddleware(t *testing.T) {
	app := lulu.New(&lulu.Config{
		Address:    "127.0.0.1:0",
		NetWork:    "tcp",
		HeartLimit: 100,
	})

	var customCalled int32
	var handlerCalled int32
	msg := &wrapperspb.Int32Value{Value: 1}
	app.Route().Register(msg, uint16(77),
		lulu.WithRegisterHandler(func(ctx lulu.Context) error {
			atomic.AddInt32(&handlerCalled, 1)
			return nil
		}),
		lulu.WithRegisterMiddleware(func(next lulu.Handler) lulu.Handler {
			return func(ctx lulu.Context) error {
				atomic.AddInt32(&customCalled, 1)
				return next(ctx)
			}
		}),
	)

	s := session.NewSession(&mockConn{}, app)
	p := network.PackingOpcode(77, nil)
	app.OnMessage(s, p)
	time.Sleep(100 * time.Millisecond)

	if atomic.LoadInt32(&customCalled) != 0 {
		t.Fatal("custom middleware should not run for invalid session")
	}
	if atomic.LoadInt32(&handlerCalled) != 0 {
		t.Fatal("handler should not run for invalid session")
	}
}

// mockConn and mockSessionCallback are defined in session_test.go
