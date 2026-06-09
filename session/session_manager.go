package session

import "sync"

type (
	// SessionManager 会话管理器
	SessionManager struct {
		sessions    map[uint64]*Session // 有效会话的集合，key：userID
		sessionsAdd chan *Session
		sessionsDel chan *Session
		closeChan   chan struct{}
		mu          sync.RWMutex // 保护 sessions 的并发访问
	}
)

// NewSessionManager 创建一个会话管理器
func NewSessionManager() *SessionManager {
	mgr := &SessionManager{
		sessions:    make(map[uint64]*Session),
		sessionsAdd: make(chan *Session),
		sessionsDel: make(chan *Session),
		closeChan:   make(chan struct{}),
	}

	go mgr.handle()

	return mgr
}

// handle 处理会话管理器
func (mgr *SessionManager) handle() {
	for {
		select {
		case <-mgr.closeChan:
			return
		case s := <-mgr.sessionsAdd:
			var oldSession *Session
			mgr.mu.Lock()
			userID := s.GetUserID()
			if old, ok := mgr.sessions[userID]; ok {
				oldSession = old
			}
			mgr.sessions[userID] = s
			mgr.mu.Unlock()
			if oldSession != nil {
				oldSession.destroy(false)
			}
		case s := <-mgr.sessionsDel:
			var removed *Session
			mgr.mu.Lock()
			userID := s.GetUserID()
			if _session, ok := mgr.sessions[userID]; ok && _session.ID == s.ID {
				delete(mgr.sessions, userID)
				removed = _session
			}
			mgr.mu.Unlock()
			if removed != nil {
				removed.destroy(false)
			}
		}
	}
}

// Add 增加会话，进入会话管理器
func (mgr *SessionManager) Add(s *Session) {
	select {
	case <-mgr.closeChan:
		return
	case mgr.sessionsAdd <- s:
	}
}

// Del 删除会话，从会话管理器中删除
func (mgr *SessionManager) Del(s *Session) {
	select {
	case <-mgr.closeChan:
		return
	case mgr.sessionsDel <- s:
	}
}

// Get 获取玩家的会话
func (mgr *SessionManager) Get(userID uint64) (*Session, bool) {
	mgr.mu.RLock()
	defer mgr.mu.RUnlock()
	s, ok := mgr.sessions[userID]
	return s, ok
}

// Len 返回当前会话的数量
func (mgr *SessionManager) Len() int {
	mgr.mu.RLock()
	defer mgr.mu.RUnlock()
	return len(mgr.sessions)
}

// Close 关闭会话管理器，停止 handle goroutine
func (mgr *SessionManager) Close() {
	select {
	case <-mgr.closeChan:
		return
	default:
		close(mgr.closeChan)
	}
}
