package lulu

import (
	"crypto/tls"
	"net"
	"net/url"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/trainking/lulu/network"
	"github.com/xtaci/kcp-go"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

type (
	// Client 客户端
	Client struct {
		Conn        network.Conn
		closeChan   chan struct{}
		closeOnce   sync.Once
		receiveChan chan network.Packet
	}
)

// NewClient 创建一个客户端
func NewClient(nw string, config *network.Config) (*Client, error) {
	var conn network.Conn
	var err error

	switch nw {
	case network.TcpNet:
		conn, err = newTcpConn(config)
	case network.KcpNet:
		conn, err = newKcpConn(config)
	case network.WebSocketNet:
		conn, err = newWsConn(config)
	default:
		return nil, network.ErrNoImplementNetwork
	}
	if err != nil {
		return nil, err
	}

	client := &Client{
		Conn:        conn,
		closeChan:   make(chan struct{}),
		receiveChan: make(chan network.Packet, 1024),
	}

	go client.receive()

	return client, nil
}

// newTcpConn 创建一个tcp连接
func newTcpConn(config *network.Config) (network.Conn, error) {
	c, err := net.Dial("tcp", config.Addr)
	if err != nil {
		return nil, err
	}

	if config.TLSConfig != nil {
		tlsConn := tls.Client(c, config.TLSConfig)

		return network.NewTcpConn(tlsConn, config)
	}

	return network.NewTcpConn(c, config)
}

// newKcpConn 创建一个kcp连接
func newKcpConn(config *network.Config) (network.Conn, error) {
	c, err := kcp.Dial(config.Addr)
	if err != nil {
		return nil, err
	}

	if config.TLSConfig != nil {
		tlsConn := tls.Client(c, config.TLSConfig)

		return network.NewKcpConn(tlsConn, config), nil
	}
	return network.NewKcpConn(c, config), nil
}

// newWsConn 创建一个websocket连接
func newWsConn(config *network.Config) (network.Conn, error) {
	u := url.URL{Scheme: "ws", Host: config.Addr, Path: config.WSUpgradePath}
	var dialer *websocket.Dialer
	if config.TLSConfig != nil {
		u.Scheme = "wss"
		dialer = &websocket.Dialer{TLSClientConfig: config.TLSConfig}
	} else {
		dialer = websocket.DefaultDialer
	}
	c, _, err := dialer.Dial(u.String(), nil)
	if err != nil {
		return nil, err
	}
	return network.NewWebSocketConn(c, config, ""), nil
}

// Send 发送消息
func (c *Client) Send(opcode uint16, msg protoreflect.ProtoMessage) error {
	msgB, err := proto.Marshal(msg)
	if err != nil {
		return err
	}
	p := network.PackingOpcode(opcode, msgB)

	return c.Conn.WritePacket(p)
}

// Close 关闭连接
func (c *Client) Close() {
	c.closeOnce.Do(func() {
		close(c.closeChan)
		c.Conn.Close()
	})
}

// Receive 接收消息
func (c *Client) Receive() <-chan network.Packet {
	return c.receiveChan
}

// receive 接收服务端消息
func (c *Client) receive() {
	defer func() {
		recover()
		c.Close()
	}()

	for {
		select {
		case <-c.closeChan:
			return
		default:
		}

		n, err := c.Conn.ReadPacket()
		if err != nil {
			return
		}

		if n.OpCode() > 0 {
			c.receiveChan <- n
		}
	}
}
