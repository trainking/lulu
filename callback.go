package lulu

import (
	"sync/atomic"

	"github.com/trainking/lulu/network"
	"github.com/trainking/lulu/session"
	"google.golang.org/protobuf/proto"
)

// OnConnect 连接建立回调
func (a *App) OnConnect(s *session.Session) {
	if a.connectEvent != nil {
		a.connectEvent(s)
	}
}

// OnMessage 收到消息回调
func (a *App) OnMessage(s *session.Session, p network.Packet) {
	// 消息洪水检查
	if s.CheckFlood(a.Config.HeartLimit) {
		s.Destroy()
		return
	}

	router, ok := a.RouterManager.GetHandleRouter(p.OpCode())
	if !ok {
		return
	}

	go a.asyncHandleMessage(s, router, p)
}

// OnDisconnect 连接断开回调
func (a *App) OnDisconnect(s *session.Session) {
	atomic.AddInt32(&a.connCount, -1)
	a.SessionManager.Del(s)
	if a.disconnectEvent != nil {
		a.disconnectEvent(s)
	}
}

// GetMsgOpCode 获取消息的 OpCode
func (a *App) GetMsgOpCode(msg proto.Message) (uint16, error) {
	return a.RouterManager.GetMsgOpcode(msg)
}
