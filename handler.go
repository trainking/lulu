package lulu

import "github.com/trainking/lulu/session"

// Handler 处理函数，对玩家请求，和内部处理的入口handler
type Handler func(Context) error

// SessionEvent 会话事件，连接，断连
type SessionEvent func(*session.Session) error
