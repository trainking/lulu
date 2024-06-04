/*
router定义路由转发的包
*/
package lulu

import (
	"reflect"
)

const (
	OpCodeMin = 0
	OpCodeMax = 65535
)

type (
	// Router 路由结构
	Router struct {
		OpCode     uint16
		Handler    Handler
		Middleware []Middleware
	}
)

// opcodeChange opcode的类型转换
func opcodeChange(opcode interface{}) (uint16, error) {
	var _op uint16
	var err error
	switch reflect.TypeOf(opcode).Kind() {
	case reflect.Int32, reflect.Int, reflect.Int64:
		_op = uint16(reflect.ValueOf(opcode).Int())
	case reflect.Uint, reflect.Uint16, reflect.Uint32:
		_op = uint16(reflect.ValueOf(opcode).Uint())
	default:
		err = ErrOpCode
	}
	return _op, err
}
