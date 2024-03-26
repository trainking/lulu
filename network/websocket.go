package network

import (
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

type (
	// WebSocketListener WebSocket监听器
	WebSocketListener struct {
		server    *http.Server
		connChan  chan Conn
		closeChan chan struct{}
		config    *Config

		ugrader websocket.Upgrader
	}

	// WebSocketConn WebSocket连接
	WebSocketConn struct {
		conn   *websocket.Conn
		config *Config
		realIp string
	}
)

// NewWebSocketListener 创建WebSocket监听器
func NewWebSocketListener(config *Config) (Listener, error) {
	l := &WebSocketListener{
		connChan:  make(chan Conn),
		closeChan: make(chan struct{}),
		config:    config,
		ugrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true
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
				fmt.Printf("Websocket ListenAndServeTLS Error: %s\n", err)
			}
		}()

	} else {
		go func() {
			if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				fmt.Printf("Websocket ListenAndServe Error: %s\n", err)
			}
		}()
	}

	l.server = server

	return l, nil
}

// handleWebSocket 将http升级成websocket
func (l *WebSocketListener) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := l.ugrader.Upgrade(w, r, nil)
	if err != nil {
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
	l.connChan <- wConn
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

// Close 关闭连接
func (l *WebSocketListener) Close() {
	close(l.closeChan)
	l.server.Close()
}

// NewWebSocketConn 读取WebSocket连接
func NewWebSocketConn(c *websocket.Conn, config *Config, realIP string) Conn {
	return &WebSocketConn{conn: c, config: config, realIp: realIP}
}

// ReadPacket 读取数据包
func (w *WebSocketConn) ReadPacket() (Packet, error) {
	if w.config.ReadTimeout > 0 {
		w.conn.SetReadDeadline(time.Now().Add(time.Duration(w.config.WriteTimeout) * time.Second))
	}

	_, message, err := w.conn.ReadMessage()
	if err != nil {
		return nil, err
	}

	return NewDefaultPacket(message), nil
}

// WritePacket 写入数据包
func (w *WebSocketConn) WritePacket(p Packet) error {
	if w.config.WriteTimeout > 0 {
		w.conn.SetWriteDeadline(time.Now().Add(time.Duration(w.config.WriteTimeout) * time.Second))
	}

	return w.conn.WriteMessage(websocket.BinaryMessage, p.Serialize())
}

// GetRealIP 获取真实的IP
func (w *WebSocketConn) GetRealIP() string {
	return w.realIp
}

// Close 关闭连接
func (w *WebSocketConn) Close() {
	if w.conn != nil {
		w.conn.Close()
	}
}
