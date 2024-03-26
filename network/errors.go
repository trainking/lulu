package network

import "github.com/pkg/errors"

var (
	ErrNoImplementNetwork = errors.New("no implement network: ")
	ErrConnClosing        = errors.New("session connect is closing")
	ErrUserNoIn           = errors.New("user no in instance")
	ErrWrongOpCode        = errors.New("wrong opcode")
	ErrWsListenerClosed   = errors.New("websocket listener closed")
)
