package lulu

import "errors"

var (
	ErrNoRegister     = errors.New("no register router") // 路由未被注册
	ErrSessionInvalid = errors.New("session invalid")    // session无效
)
