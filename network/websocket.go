package network

import (
	"log"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	// DefaultWebSocketConnChanSize 默认连接通道缓冲大小
	DefaultWebSocketConnChanSize = 64
)

type (
	// WebSocketListener WebSocket 监听器
	WebSocketListener struct {
		server     *http.Server
		connChan   chan Conn
		closeChan  chan struct{}
		closeOnce  sync.Once
		config     *Config
		isClosed   bool

		ugrader websocket.Upgrader
	}

	// WebSocketConn WebSocket 连接
	WebSocketConn struct {
		conn     *websocket.Conn
		config   *Config
		realIp   string
		mu       sync.RWMutex
		isClosed bool
	}
)

// NewWebSocketListener 创建 WebSocket 监听器
func NewWebSocketListener(config *Config) (Listener, error) {
	l := &WebSocketListener{
		connChan:  make(chan Conn, DefaultWebSocketConnChanSize),
		closeChan: make(chan struct{}),
		config:    config,
		ugrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				// 允许同源请求
				origin := r.Header.Get("Origin")
				if origin == "" {
					return true
				}
				// 验证 Origin 是否与请求主机匹配
				host := r.Host
				if host == "" {
					host = r.URL.Host
				}
				// 简单验证：如果 Origin 包含主机地址，则允许
				// 生产环境应该使用白名单机制
				return origin == "http://"+host || origin == "https://"+host || origin == "ws://"+host || origin == "wss://"+host
			},
		},
	}
	http.HandleFunc(config.WSUpgradePath, l.handleWebSocket)

	server := &http.Server{
		Addr: config.Addr,
	}

	if config.TLSConfig != nil {
		server.TLSConfig = config.TLSConfig
		go func() {
			if err := server.ListenAndServeTLS("", ""); err != nil && err != http.ErrServerClosed {
				log.Printf("Websocket ListenAndServeTLS Error: %s\n", err)
			}
		}()

	} else {
		go func() {
			if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				log.Printf("Websocket ListenAndServe Error: %s\n", err)
			}
		}()
	}

	l.server = server

	return l, nil
}

// handleWebSocket 将 http 升级成 websocket
func (l *WebSocketListener) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := l.ugrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket Upgrade Error: %s\n", err)
		return
	}
	clientIP := r.Header.Get("X-Forwarded-For")
	if clientIP == "" {
		clientIP = r.Header.Get("X-Real-IP")
	}
	if clientIP == "" {
		ip, _, _ := net.SplitHostPort(r.RemoteAddr)
		clientIP = ip
	}

	wConn := NewWebSocketConn(conn, l.config, clientIP)

	select {
	case l.connChan <- wConn:
	case <-l.closeChan:
		// 监听器已关闭，拒绝新连接
		wConn.Close()
		log.Printf("WebSocket Listener closed, reject new connection from %s\n", clientIP)
	}
}

// Accept 接收请求
func (l *WebSocketListener) Accept() (Conn, error) {
	select {
	case conn := <-l.connChan:
		return conn, nil
	case <-l.closeChan:
		return nil, ErrWsListenerClosed
	}
}

// Close 关闭连接（幂等）
func (l *WebSocketListener) Close() {
	l.closeOnce.Do(func() {
		l.isClosed = true
		close(l.closeChan)
		if l.server != nil {
			if err := l.server.Close(); err != nil {
				log.Printf("WebSocket Listener Close Error: %s\n", err)
			}
		}
	})
}

// NewWebSocketConn 读取 WebSocket 连接
func NewWebSocketConn(c *websocket.Conn, config *Config, realIP string) Conn {
	return &WebSocketConn{conn: c, config: config, realIp: realIP}
}

// ReadPacket 读取数据包
func (w *WebSocketConn) ReadPacket() (Packet, error) {
	if w.config.ReadTimeout > 0 {
		w.conn.SetReadDeadline(time.Now().Add(time.Duration(w.config.ReadTimeout) * time.Second))
	}

	_, message, err := w.conn.ReadMessage()
	if err != nil {
		return nil, err
	}

	return NewDefaultPacket(message), nil
}

// WritePacket 写入数据包
func (w *WebSocketConn) WritePacket(p Packet) error {
	w.mu.RLock()
	if w.isClosed {
		w.mu.RUnlock()
		return ErrConnClosing
	}
	w.mu.RUnlock()

	if w.config.WriteTimeout > 0 {
		w.conn.SetWriteDeadline(time.Now().Add(time.Duration(w.config.WriteTimeout) * time.Second))
	}

	return w.conn.WriteMessage(websocket.BinaryMessage, p.Serialize())
}

// GetRealIP 获取真实的 IP
func (w *WebSocketConn) GetRealIP() string {
	return w.realIp
}

// Close 关闭连接（幂等）
func (w *WebSocketConn) Close() {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.isClosed {
		return
	}
	w.isClosed = true

	if w.conn != nil {
		w.conn.Close()
	}
}
