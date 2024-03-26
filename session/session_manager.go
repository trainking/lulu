package session

type (
	// SessionManager 会话管理器
	SessionManager struct {
		sessions    map[uint64]*Session // 有效会话的集合，key：userID
		sessionsAdd chan *Session
		sessionsDel chan *Session
		closeChan   chan struct{}
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
			if oldSession, ok := mgr.sessions[s.UserID]; ok {
				oldSession.Destory()
			}
			mgr.sessions[s.UserID] = s
		case s := <-mgr.sessionsDel:
			if _session, ok := mgr.sessions[s.UserID]; ok && _session.ID == s.ID {
				delete(mgr.sessions, s.UserID)
				_session.Destory()
			}

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
	s, ok := mgr.sessions[userID]
	return s, ok
}

// Len 返回当前会话的数量
func (mgr *SessionManager) Len() int {
	return len(mgr.sessions)
}
