package network

import (
	"encoding/binary"
	"io"
	"sync"
)

var (
	// DefaultPacketPool 将包结构内存加入池
	DefaultPacketPool = sync.Pool{
		New: func() interface{} {
			return new(DefaultPacket)
		},
	}
)

type (
	// Packet 对数据打包的接口
	Packet interface {
		// Serialize 序列化
		Serialize() []byte

		// OpeCode 获取该包的OpCode
		OpCode() uint16

		// BodyLen 内容长度
		BodyLen() uint16

		// Body 获取完整body
		Body() []byte

		// Free 释放空间
		Free()
	}

	// DefaultPacket 默认的包实现
	DefaultPacket struct {
		buff []byte
	}
)

// NewDefaultPacket 创建一个默认的包
func NewDefaultPacket(buff []byte) Packet {
	p := DefaultPacketPool.Get().(*DefaultPacket)
	p.buff = buff

	return p
}

// Serialize 序列化，输出完整的字符数组
func (p *DefaultPacket) Serialize() []byte {
	return p.buff
}

// OpCode 包的2-3位为OpCode
func (p *DefaultPacket) OpCode() uint16 {
	return binary.BigEndian.Uint16(p.buff[2:4])
}

// BodyLen 报文内容长度
func (p *DefaultPacket) BodyLen() uint16 {
	return binary.BigEndian.Uint16(p.buff[0:2])
}

// Body 读取body所有字符
func (p *DefaultPacket) Body() []byte {
	return p.buff[4:]
}

// Free 释放空间
func (p *DefaultPacket) Free() {
	p.buff = p.buff[:0]

	// 回到池中
	DefaultPacketPool.Put(p)
}

// PackingReader 从io.Reader中读取一个Packet
func PackingReader(r io.Reader) (Packet, error) {
	// 4字节头
	var headrBytes = make([]byte, 4)

	// 读取头
	if _, err := io.ReadFull(r, headrBytes); err != nil {
		return nil, err
	}

	bodyLength := binary.BigEndian.Uint16(headrBytes[0:2])
	var buff []byte
	if bodyLength > 0 {
		buff = make([]byte, bodyLength)

		// 读取body
		if _, err := io.ReadFull(r, buff); err != nil {
			return nil, err
		}
	}

	pbuff := make([]byte, 4+len(buff))
	copy(pbuff[0:], headrBytes)
	if bodyLength > 0 {
		copy(pbuff[4:], buff)
	}

	return NewDefaultPacket(pbuff), nil
}

// PackingOpcode 加入opcode方式，创建一个Packet
func PackingOpcode(opcode uint16, msg []byte) Packet {
	bodyLen := len(msg)
	buff := make([]byte, 4+bodyLen)
	binary.BigEndian.PutUint16(buff[0:2], uint16(bodyLen))
	binary.BigEndian.PutUint16(buff[2:4], opcode)
	if bodyLen > 0 {
		copy(buff[4:], msg)
	}

	return NewDefaultPacket(buff)
}
