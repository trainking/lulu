package session

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/trainking/lulu/network"
	"google.golang.org/protobuf/proto"
)

type (
	// Session 会话，会话是对玩家连接的一个包装
	Session struct {
		ID     int64        // 会话 ID， 唯一标识
		Conn   network.Conn // 会话连接
		UserID uint64       // 用户 ID

		callback  SessionCallback // 回调接口
		closeChan chan struct{}   // 关闭信号
		closeOnce sync.Once       // 控制关闭单例
		validChan chan uint64     // 验证通过信号
		lastTick  int64           // 最后一次计数刷新时间 (Unix 秒)
		msgCount  int32           // 当前周期的消息计数
	}

	// SessionCallback 会话回调接口
	SessionCallback interface {
		// OnConnect 当连接建立时调用
		OnConnect(*Session)

		// OnMessage 当连接处理消息时
		OnMessage(*Session, network.Packet)

		// OnDisconnect 当连接断开时
		OnDisconnect(*Session)

		// GetMsgOpCode 获取消息的 OpCode
		GetMsgOpCode(msg proto.Message) (uint16, error)
	}
)

// NewSession 创建会话
func NewSession(conn network.Conn, callback SessionCallback) *Session {
	return &Session{
		ID:        time.Now().UnixNano(),
		Conn:      conn,
		callback:  callback,
		closeChan: make(chan struct{}),
		validChan: make(chan uint64, 1), // 使用缓冲 channel 防止阻塞
	}
}

// Run 运行会话
func (s *Session) Run() {
	defer func() {
		if e := recover(); e != nil {
			fmt.Printf("read loop error: %v\n", e)
		}
		s.Destroy()
	}()

	for {
		select {
		case <-s.closeChan:
			return
		default:
		}

		p, err := s.Conn.ReadPacket()
		if err != nil {
			return
		}

		s.callback.OnMessage(s, p)
	}
}

// SetUserID 设置用户 ID
func (s *Session) SetUserID(userID uint64) {
	select {
	case s.validChan <- userID:
		s.UserID = userID
	case <-s.closeChan:
		// session 已关闭，不再需要验证
		return
	}
}

// WaitValid 等待验证
func (s *Session) WaitValid() <-chan uint64 {
	return s.validChan
}

// IsValid 是否有效
func (s *Session) IsValid() bool {
	return s.UserID != 0
}

// CheckFlood 检查是否洪水攻击，每分钟超过 limit 返回 true
func (s *Session) CheckFlood(limit int) bool {
	if limit <= 0 {
		return false
	}
	now := time.Now().Unix()
	last := atomic.LoadInt64(&s.lastTick)
	if now-last >= 60 {
		if atomic.CompareAndSwapInt64(&s.lastTick, last, now) {
			atomic.StoreInt32(&s.msgCount, 1)
			return false
		}
	}
	return atomic.AddInt32(&s.msgCount, 1) > int32(limit)
}

// Send 向此 session 推送消息
func (s *Session) Send(msg proto.Message) error {
	opcode, err := s.callback.GetMsgOpCode(msg)
	if err != nil {
		return err
	}

	msgB, err := proto.Marshal(msg)
	if err != nil {
		return err
	}

	pakcket := network.PackingOpcode(opcode, msgB)
	return s.Conn.WritePacket(pakcket)
}

// Destroy 销毁会话
func (s *Session) Destroy() {
	s.closeOnce.Do(func() {
		close(s.closeChan)
		s.Conn.Close()
		s.callback.OnDisconnect(s)
	})
}
