/*
router定义路由转发的包
*/
package lulu

import (
	"reflect"
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
func opcodeChange(opcode interface{}) uint16 {
	var _op uint16
	switch reflect.TypeOf(opcode).Kind() {
	case reflect.Int32, reflect.Int, reflect.Int64:
		_op = uint16(reflect.ValueOf(opcode).Int())
	case reflect.Uint, reflect.Uint16, reflect.Uint32:
		_op = uint16(reflect.ValueOf(opcode).Uint())
	default:
	}
	return _op
}
