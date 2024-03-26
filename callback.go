package lulu

import (
	"github.com/trainking/lulu/network"
	"github.com/trainking/lulu/session"
	"google.golang.org/protobuf/proto"
)

// OnConnect 连接建立回调
func (a *App) OnConnect(s *session.Session) {
	a.connectEvent(s)
}

// OnMessage 收到消息回调
func (a *App) OnMessage(s *session.Session, p network.Packet) {
	router, ok := a.RouterManager.GetHandleRouter(p.OpCode())
	if !ok {
		return
	}

	go a.asyncHandleMessage(s, router, p)
}

// OnDisConnect 连接断开回调
func (a *App) OnDisConnect(s *session.Session) {
	a.SessionManager.Del(s)
	a.disconnectEvent(s)
}

// GetMsgOpCode 获取消息的OpCode
func (a *App) GetMsgOpCode(msg proto.Message) (uint16, error) {
	return a.RouterManager.GetMsgOpcde(msg)
}
