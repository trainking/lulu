package tests

import (
	"context"
	"testing"

	"github.com/trainking/lulu"
	"github.com/trainking/lulu/session"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func TestContextBind(t *testing.T) {
	msg := &wrapperspb.StringValue{Value: "hello"}
	body, err := proto.Marshal(msg)
	if err != nil {
		t.Fatal(err)
	}

	s := &session.Session{}
	app := lulu.New(&lulu.Config{Address: "127.0.0.1:0", NetWork: "tcp"})
	ctx := lulu.NewContext(context.Background(), app, s, 1001, body)

	target := &wrapperspb.StringValue{}
	if err := ctx.Bind(target); err != nil {
		t.Fatalf("Bind failed: %v", err)
	}
	if target.Value != "hello" {
		t.Errorf("Bind result = %q, want %q", target.Value, "hello")
	}
}

func TestContextSession(t *testing.T) {
	s := &session.Session{}
	app := lulu.New(&lulu.Config{Address: "127.0.0.1:0", NetWork: "tcp"})
	ctx := lulu.NewContext(context.Background(), app, s, 1001, nil)

	if ctx.Session() != s {
		t.Error("Context.Session() should return the session passed to NewContext")
	}
}

func TestContextApp(t *testing.T) {
	app := lulu.New(&lulu.Config{Address: "127.0.0.1:0", NetWork: "tcp"})
	ctx := lulu.NewContext(context.Background(), app, &session.Session{}, 1001, nil)

	if ctx.App() != app {
		t.Error("Context.App() should return the app passed to NewContext")
	}
}

func TestContextGetOpCode(t *testing.T) {
	app := lulu.New(&lulu.Config{Address: "127.0.0.1:0", NetWork: "tcp"})
	ctx := lulu.NewContext(context.Background(), app, &session.Session{}, 0xABCD, nil)

	if ctx.GetOpCode() != 0xABCD {
		t.Errorf("GetOpCode = %d, want %d", ctx.GetOpCode(), 0xABCD)
	}
}

func TestContextBackgroundContext(t *testing.T) {
	bg := context.WithValue(context.Background(), struct{}{}, "value")
	app := lulu.New(&lulu.Config{Address: "127.0.0.1:0", NetWork: "tcp"})
	ctx := lulu.NewContext(bg, app, &session.Session{}, 100, nil)

	if ctx.Context() != bg {
		t.Error("Context.Context() should return the wrapped context")
	}
}
