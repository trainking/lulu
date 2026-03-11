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
			mgr.mu.Lock()
			if oldSession, ok := mgr.sessions[s.UserID]; ok {
				oldSession.Destroy()
			}
			mgr.sessions[s.UserID] = s
			mgr.mu.Unlock()
		case s := <-mgr.sessionsDel:
			mgr.mu.Lock()
			if _session, ok := mgr.sessions[s.UserID]; ok && _session.ID == s.ID {
				delete(mgr.sessions, s.UserID)
				_session.Destroy()
			}
			mgr.mu.Unlock()
		}
	}
}

// Add 增加会话，进入会话管理器
func (mgr *SessionManager) Add(s *Session) {
	mgr.sessionsAdd <- s
}

// Del 删除会话，从会话管理器中删除
func (mgr *SessionManager) Del(s *Session) {
	mgr.sessionsDel <- s
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
