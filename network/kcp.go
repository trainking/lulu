package network

import (
	"crypto/tls"
	"net"
	"time"

	"github.com/xtaci/kcp-go"
)

type (
	// KcpListener KCP监听器
	KcpListener struct {
		listener net.Listener
		config   *Config
	}

	KcpConn struct {
		conn   net.Conn
		config *Config
	}
)

// NewKcpListener 创建KCP监听器
func NewKcpListener(config *Config) (Listener, error) {
	l, err := kcp.Listen(config.Addr)
	if err != nil {
		return nil, err
	}

	return &KcpListener{listener: l, config: config}, nil
}

func (l *KcpListener) Accept() (Conn, error) {
	c, err := l.listener.Accept()
	if err != nil {
		return nil, err
	}

	kcpConn := c.(*kcp.UDPSession)
	// 极速模式；普通模式参数为 0, 40, 0, 0
	if l.config.KcpMode == "nomarl" {
		kcpConn.SetNoDelay(0, 40, 0, 0)
	} else {
		kcpConn.SetNoDelay(1, 10, 2, 1)
	}

	kcpConn.SetStreamMode(true)
	kcpConn.SetWindowSize(4096, 4096)
	kcpConn.SetReadBuffer(4 * 65536 * 1024)
	kcpConn.SetWriteBuffer(4 * 65536 * 1024)
	kcpConn.SetACKNoDelay(true)

	if l.config.TLSConfig != nil {
		tlsConn := tls.Server(kcpConn, l.config.TLSConfig)
		return NewKcpConn(tlsConn, l.config), nil
	}

	return NewKcpConn(kcpConn, l.config), nil
}

// Close 关闭连接的监听器
func (l *KcpListener) Close() {
	l.listener.Close()
}

// NewKcpConn 创建一个KCP连接
func NewKcpConn(c net.Conn, config *Config) Conn {
	return &KcpConn{conn: c, config: config}
}

// ReadPacket 读取报文
func (k *KcpConn) ReadPacket() (Packet, error) {
	if k.config.ReadTimeout > 0 {
		k.conn.SetReadDeadline(time.Now().Add(time.Duration(k.config.ReadTimeout) * time.Second))
	}

	return PackingReader(k.conn)
}

// WritePacket 写入报文
func (k *KcpConn) WritePacket(p Packet) error {
	if k.config.WriteTimeout > 0 {
		k.conn.SetWriteDeadline(time.Now().Add(time.Duration(k.config.WriteTimeout) * time.Second))
	}

	_, err := k.conn.Write(p.Serialize())
	return err
}

// GetReadIP 获取真实的IP
func (k *KcpConn) GetRealIP() string {
	return k.conn.RemoteAddr().String()
}

// Close 关闭连接
func (k *KcpConn) Close() {
	k.conn.Close()
}
