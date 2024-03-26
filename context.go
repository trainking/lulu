package lulu

import (
	"context"

	"github.com/trainking/lulu/session"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

type (
	// Context 抽象每个Handler的调用参数
	Context interface {

		// Context 返回一个context.Context
		Context() context.Context

		// Bind 取出请求参数
		Bind(m protoreflect.ProtoMessage) error

		// Session 获取这个玩家的Session
		Session() *session.Session

		// GetOpCode 获取此次请求的opcode
		GetOpCode() uint16
	}

	// DefaultContext 默认Context实现
	DefaultContext struct {
		a       *App
		session *session.Session
		ctx     context.Context
		opcode  uint16
		body    []byte
	}
)

// NewContext 创建一个Context，提供给Handler使用
func NewContext(ctx context.Context, a *App, session *session.Session, opcode uint16, body []byte) Context {

	return &DefaultContext{
		a:       a,
		session: session,
		ctx:     ctx,
		opcode:  opcode,
		body:    body,
	}
}

// Context 返回一个context.Context
func (c *DefaultContext) Context() context.Context {
	return c.ctx
}

// Bind 取出请求参数
func (c *DefaultContext) Bind(m protoreflect.ProtoMessage) error {
	return proto.Unmarshal(c.body, m)
}

// Session 获取这个玩家的Session
func (c *DefaultContext) Session() *session.Session {
	return c.session
}

// App 获取App实例
func (c *DefaultContext) App() *App {
	return c.a
}

// GetOpCode 获取此次处理的opcode
func (c *DefaultContext) GetOpCode() uint16 {
	return c.opcode
}
