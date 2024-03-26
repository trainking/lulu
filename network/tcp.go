package network

import (
	"crypto/tls"
	"net"
	"time"
)

type (
	// TcpListener TCP监听器
	TcpListener struct {
		listener net.Listener
		config   *Config
	}

	// TcpConn TCP连接
	TcpConn struct {
		conn   net.Conn
		config *Config
	}
)

// NewTcpListener 创建TCP监听器
func NewTcpListener(c *Config) (Listener, error) {
	l, err := net.Listen("tcp", c.Addr)
	if err != nil {
		return nil, err
	}

	return &TcpListener{
		listener: l,
		config:   c,
	}, nil
}

// NewTcpConn 创建TCP连接
func NewTcpConn(conn net.Conn, c *Config) (Conn, error) {
	return &TcpConn{
		conn:   conn,
		config: c,
	}, nil
}

// Accept 接收连接
func (l *TcpListener) Accept() (Conn, error) {
	c, err := l.listener.Accept()
	if err != nil {
		return nil, err
	}

	if l.config.TLSConfig != nil {
		tlsConn := tls.Server(c, l.config.TLSConfig)
		return NewTcpConn(tlsConn, l.config)
	}

	return NewTcpConn(c, l.config)
}

// Close 关闭监听器
func (l *TcpListener) Close() {
	l.listener.Close()
}

// ReadPacket 读取报文
func (c *TcpConn) ReadPacket() (Packet, error) {
	if c.config.ReadTimeout > 0 {
		c.conn.SetReadDeadline(time.Now().Add(time.Duration(c.config.ReadTimeout) * time.Second))
	}
	return PackingReader(c.conn)
}

// WritePacket 写入报文
func (c *TcpConn) WritePacket(p Packet) error {
	if c.config.WriteTimeout > 0 {
		c.conn.SetWriteDeadline(time.Now().Add(time.Duration(c.config.WriteTimeout) * time.Second))
	}

	_, err := c.conn.Write(p.Serialize())
	return err
}

// GetRealIP 获取对端的真实IP
func (c *TcpConn) GetRealIP() string {
	return c.conn.RemoteAddr().String()
}

// Close 关闭连接
func (c *TcpConn) Close() {
	c.conn.Close()
}
