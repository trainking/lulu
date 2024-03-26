/*
network 定义对网络的抽象。
*/
package network

import (
	"crypto/tls"

	"github.com/pkg/errors"
)

const (
	TcpNet       = "tcp"
	KcpNet       = "kcp"
	WebSocketNet = "websocket"
)

type (
	// Listener 监听器接口
	Listener interface {
		// Accept 接受连接
		Accept() (Conn, error)

		// Close 关闭监听器
		Close()
	}

	// Conn 传输连接的抽象
	Conn interface {
		// ReadPacket 读取报文
		ReadPacket() (Packet, error)

		// WritePacket 写入报文
		WritePacket(Packet) error

		// GetRealIP 获取真实的IP
		GetRealIP() string

		// Close 关闭连接
		Close()
	}

	// Config 网络配置
	Config struct {
		Addr         string      // 监听地址
		TLSConfig    *tls.Config // TLS配置
		WriteTimeout int         // 写入超时时间
		ReadTimeout  int         // 读取超时时间

		WSUpgradePath string // websocket升级路径
		KcpMode       string // kcp模式
	}

	// ListenerFactory 监听器工厂
	ListenerFactory struct {
		network       string
		address       string
		writeTimeout  int
		readTimeout   int
		tlsConf       *tls.Config
		kcpMode       string
		wsUpgradePath string
	}
)

// NewListenerFactory 创建监听器工厂
func NewListenerFactory(network string, address string, writeTimeout int, readTimeout int) *ListenerFactory {
	return &ListenerFactory{
		network:      network,
		address:      address,
		writeTimeout: writeTimeout,
		readTimeout:  readTimeout,
	}
}

// WithTlsConfig 设置TLS配置
func (l *ListenerFactory) WithTlsConfig(tlsConf *tls.Config) {
	l.tlsConf = tlsConf
}

// WithKcpMode 设置KCP模式
func (l *ListenerFactory) WithKcpMode(kcpMode string) {
	l.kcpMode = kcpMode
}

// WithUpgradePath 设置websocket升级路径
func (l *ListenerFactory) WithUpgradePath(wsUpgradePath string) {
	l.wsUpgradePath = wsUpgradePath
}

// Generate 创建监听器
func (l *ListenerFactory) Generate() (Listener, error) {
	var netConfig = Config{
		Addr:         l.address,
		WriteTimeout: l.writeTimeout,
		ReadTimeout:  l.readTimeout,
	}

	if l.tlsConf != nil {
		netConfig.TLSConfig = l.tlsConf
	}

	var listener Listener
	var err error

	switch l.network {
	case TcpNet:
		listener, err = NewKcpListener(&netConfig)
	case KcpNet:
		netConfig.KcpMode = l.kcpMode
		listener, err = NewTcpListener(&netConfig)
	case WebSocketNet:
		netConfig.WSUpgradePath = l.wsUpgradePath
		listener, err = NewWebSocketListener(&netConfig)
	default:
		return nil, errors.Wrap(ErrNoImplementNetwork, l.network)
	}

	return listener, err
}
