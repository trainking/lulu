package lulu

import (
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

type (
	// RouterManager 路由管理器
	RouterManager struct {
		handleRouter map[uint16]Router
		innderRouter map[protoreflect.FullName]Router
		outSendMap   map[protoreflect.FullName]interface{}
	}
)

// NewRouterManager 创建路由管理器
func NewRouterManager() *RouterManager {
	return &RouterManager{
		handleRouter: make(map[uint16]Router),
		innderRouter: make(map[protoreflect.FullName]Router),
		outSendMap:   make(map[protoreflect.FullName]interface{}),
	}
}

// Regsiter 注册路由
func (r *RouterManager) Register(msg proto.Message, opcode interface{}, opts ...RegisterOptions) {
	rp := NewRegisterParams(opts...)
	if rp.Handler == nil {
		r.outSendMap[msg.ProtoReflect().Descriptor().FullName()] = opcode
		return
	}

	var m []Middleware
	if !rp.IsNoValid {
		m = append(m, MiddlewareValidSession())
	}

	if len(rp.Middleware) > 0 {
		m = append(m, rp.Middleware...)
	}
	if rp.IsInner {
		_op, err := opcodeChange(opcode)
		if err == ErrOpCode {
			return
		}
		r.innderRouter[msg.ProtoReflect().Type().Descriptor().FullName()] = Router{
			OpCode:     _op,
			Handler:    rp.Handler,
			Middleware: m,
		}
	} else {
		_op, err := opcodeChange(opcode)
		if err == ErrOpCode {
			return
		}
		r.handleRouter[_op] = Router{
			OpCode:     _op,
			Handler:    rp.Handler,
			Middleware: m,
		}
	}
}

// GetHandleRouter 获取请求处理路由
func (r *RouterManager) GetHandleRouter(opcode uint16) (Router, bool) {
	_r, ok := r.handleRouter[opcode]
	return _r, ok
}

// GetInnerRouter 获取内部路由
func (r *RouterManager) GetInnerRouter(msgName protoreflect.FullName) (Router, bool) {
	_r, ok := r.innderRouter[msgName]
	return _r, ok
}

// GetSendOpCode 获取返回消息对应的opcode
func (r *RouterManager) GetSendOpCode(msgName protoreflect.FullName) (uint16, bool) {
	_op, ok := r.outSendMap[msgName]
	if !ok {
		return 0, false
	}
	cd, err := opcodeChange(_op)
	if err != nil {
		return 0, false
	}
	return cd, true
}

// GetMsgOpcde 获取消息对应的opcode，只获取内部路由和返回消息
func (r *RouterManager) GetMsgOpcde(msg proto.Message) (uint16, error) {
	msgName := msg.ProtoReflect().Descriptor().FullName()
	if _router, ok := r.innderRouter[msgName]; ok {
		return _router.OpCode, nil
	}

	if opcode, ok := r.outSendMap[msgName]; ok {
		cd, err := opcodeChange(opcode)
		if err != nil {
			return 0, err
		}
		return cd, nil
	}

	return 0, ErrNoRegister
}
