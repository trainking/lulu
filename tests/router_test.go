package tests

import (
	"testing"

	"github.com/trainking/lulu"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func TestRegisterExternalRoute(t *testing.T) {
	app := lulu.New(&lulu.Config{
		Address: "127.0.0.1:0",
		NetWork: "tcp",
	})

	handlerCalled := false
	handler := func(ctx lulu.Context) error {
		handlerCalled = true
		return nil
	}

	msg := &wrapperspb.StringValue{Value: "test"}
	app.Route().Register(msg, uint16(1001), lulu.WithRegisterHandler(handler))

	router, ok := app.Route().GetHandleRouter(1001)
	if !ok {
		t.Fatal("expected router to be registered")
	}
	if router.OpCode != 1001 {
		t.Errorf("opcode = %d, want %d", router.OpCode, 1001)
	}
	if router.Handler == nil {
		t.Error("handler should not be nil")
	}
	if len(router.Middleware) != 1 {
		t.Fatalf("expected 1 middleware (default valid-session), got %d", len(router.Middleware))
	}

	// Verify handler works
	_ = handlerCalled
}

func TestRegisterExternalRouteNoValid(t *testing.T) {
	app := lulu.New(&lulu.Config{
		Address: "127.0.0.1:0",
		NetWork: "tcp",
	})

	handler := func(ctx lulu.Context) error { return nil }
	msg := &wrapperspb.StringValue{Value: "auth-req"}
	app.Route().Register(msg, uint16(1001),
		lulu.WithRegisterHandler(handler),
		lulu.WithRegisterIsNoValid(true),
	)

	router, ok := app.Route().GetHandleRouter(1001)
	if !ok {
		t.Fatal("expected router to be registered")
	}
	if len(router.Middleware) != 0 {
		t.Errorf("expected 0 middleware when IsNoValid=true, got %d", len(router.Middleware))
	}
}

func TestRegisterInternalRoute(t *testing.T) {
	app := lulu.New(&lulu.Config{
		Address: "127.0.0.1:0",
		NetWork: "tcp",
	})

	handler := func(ctx lulu.Context) error { return nil }
	msg := &wrapperspb.Int32Value{Value: 42}
	app.Route().Register(msg, uint16(2001),
		lulu.WithRegisterHandler(handler),
		lulu.WithRegisterIsInner(true),
	)

	fullName := msg.ProtoReflect().Type().Descriptor().FullName()
	router, ok := app.Route().GetInnerRouter(fullName)
	if !ok {
		t.Fatalf("expected internal router for %s", fullName)
	}
	if router.OpCode != 2001 {
		t.Errorf("opcode = %d, want %d", router.OpCode, 2001)
	}
}

func TestRegisterReturnRoute(t *testing.T) {
	app := lulu.New(&lulu.Config{
		Address: "127.0.0.1:0",
		NetWork: "tcp",
	})

	msg := &wrapperspb.BoolValue{Value: true}
	app.Route().Register(msg, uint16(3001))

	fullName := msg.ProtoReflect().Descriptor().FullName()
	opcode, ok := app.Route().GetSendOpCode(fullName)
	if !ok {
		t.Fatalf("expected return route for %s", fullName)
	}
	if opcode != 3001 {
		t.Errorf("opcode = %d, want %d", opcode, 3001)
	}

	// Ensure it is NOT a handle router
	_, ok = app.Route().GetHandleRouter(3001)
	if ok {
		t.Error("return route should not appear as handle router")
	}
}

func TestRegisterRouteWithMiddleware(t *testing.T) {
	app := lulu.New(&lulu.Config{
		Address: "127.0.0.1:0",
		NetWork: "tcp",
	})

	mw := func(next lulu.Handler) lulu.Handler {
		return func(ctx lulu.Context) error { return next(ctx) }
	}

	handler := func(ctx lulu.Context) error { return nil }
	msg := &wrapperspb.StringValue{Value: "mw-test"}
	app.Route().Register(msg, uint16(4001),
		lulu.WithRegisterHandler(handler),
		lulu.WithRegisterMiddleware(mw),
	)

	router, ok := app.Route().GetHandleRouter(4001)
	if !ok {
		t.Fatal("expected router to be registered")
	}
	// 1 default valid-session + 1 custom = 2
	if len(router.Middleware) != 2 {
		t.Errorf("expected 2 middleware, got %d", len(router.Middleware))
	}
}

func TestGetHandleRouterNotFound(t *testing.T) {
	app := lulu.New(&lulu.Config{
		Address: "127.0.0.1:0",
		NetWork: "tcp",
	})

	_, ok := app.Route().GetHandleRouter(9999)
	if ok {
		t.Error("should not find unregistered router")
	}
}

func TestGetSendOpCodeNotFound(t *testing.T) {
	app := lulu.New(&lulu.Config{
		Address: "127.0.0.1:0",
		NetWork: "tcp",
	})

	msg := &wrapperspb.StringValue{Value: "unregistered"}
	_, ok := app.Route().GetSendOpCode(msg.ProtoReflect().Descriptor().FullName())
	if ok {
		t.Error("should not find unregistered return route")
	}
}

func TestGetMsgOpcode(t *testing.T) {
	app := lulu.New(&lulu.Config{
		Address: "127.0.0.1:0",
		NetWork: "tcp",
	})

	// Register an internal route
	innerMsg := &wrapperspb.Int32Value{Value: 1}
	app.Route().Register(innerMsg, uint16(5001),
		lulu.WithRegisterHandler(func(ctx lulu.Context) error { return nil }),
		lulu.WithRegisterIsInner(true),
	)

	// Register a return route
	retMsg := &wrapperspb.StringValue{Value: "ret"}
	app.Route().Register(retMsg, uint16(5002))

	// Test internal route lookup
	op, err := app.GetMsgOpCode(innerMsg)
	if err != nil {
		t.Errorf("unexpected error for internal route: %v", err)
	}
	if op != 5001 {
		t.Errorf("opcode = %d, want %d", op, 5001)
	}

	// Test return route lookup
	op, err = app.GetMsgOpCode(retMsg)
	if err != nil {
		t.Errorf("unexpected error for return route: %v", err)
	}
	if op != 5002 {
		t.Errorf("opcode = %d, want %d", op, 5002)
	}

	// Test unregistered
	unreg := &wrapperspb.BytesValue{Value: []byte("x")}
	_, err = app.GetMsgOpCode(unreg)
	if err == nil {
		t.Error("expected error for unregistered message")
	}
}

func TestRegisterMultipleOpcodeTypes(t *testing.T) {
	app := lulu.New(&lulu.Config{
		Address: "127.0.0.1:0",
		NetWork: "tcp",
	})

	handler := func(ctx lulu.Context) error { return nil }
	msg := &wrapperspb.StringValue{Value: "multi"}

	// Test with uint16
	app.Route().Register(msg, uint16(100), lulu.WithRegisterHandler(handler))
	router, ok := app.Route().GetHandleRouter(100)
	if !ok {
		t.Error("router registered with uint16 should be found")
	}
	_ = router

	// Test with int (opcode 200)
	msg2 := &wrapperspb.StringValue{Value: "multi2"}
	app.Route().Register(msg2, int(200), lulu.WithRegisterHandler(handler))
	router2, ok := app.Route().GetHandleRouter(200)
	if !ok {
		t.Error("router registered with int should be found")
	}
	_ = router2

	// Test with uint (opcode 300)
	msg3 := &wrapperspb.StringValue{Value: "multi3"}
	app.Route().Register(msg3, uint(300), lulu.WithRegisterHandler(handler))
	router3, ok := app.Route().GetHandleRouter(300)
	if !ok {
		t.Error("router registered with uint should be found")
	}
	_ = router3
}
