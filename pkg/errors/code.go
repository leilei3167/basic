package errors

import (
	"fmt"
	"net/http"
	"sync"
)

var (
	unknownCoder = defaultCoder{C: 1, HTTP: http.StatusInternalServerError,
		Ext: "An internal server error occurred", Ref: "none"}
	codes   = map[int]Coder{}
	codeMux = &sync.Mutex{}
)

func init() {
	codes[unknownCoder.Code()] = unknownCoder
}

//Coder 接口定义了一个错误码的详细信息接口
type Coder interface {
	HTTPStatus() int
	String() string
	Reference() string
	Code() int
}

type defaultCoder struct {
	C    int
	HTTP int
	Ext  string
	Ref  string
}

func (coder defaultCoder) Code() int {
	return coder.C
}

func (coder defaultCoder) String() string {
	return coder.Ext
}

func (coder defaultCoder) HTTPStatus() int {
	if coder.HTTP == 0 {
		return 500
	}
	return coder.HTTP
}

func (coder defaultCoder) Reference() string {
	return coder.Ref
}

//将错误和对应的错误信息注册

func Register(coder Coder) {
	if coder.Code() == 0 {
		panic(`请设置非0错误码`)
	}
	codeMux.Lock()
	defer codeMux.Unlock()
	codes[coder.Code()] = coder
}

//MustRegister 不允许对同一个错误码重复注册
func MustRegister(coder Coder) {
	if coder.Code() == 0 {
		panic(`请设置非0错误码`)
	}
	codeMux.Lock()
	defer codeMux.Unlock()

	if _, ok := codes[coder.Code()]; ok {
		panic(fmt.Sprintf("code: %d is already exist", coder.Code()))
	}

	codes[coder.Code()] = coder
}

func ParseCoder(err error) Coder {
	if err == nil {
		return nil
	}

	if v, ok := err.(*withCode); ok { //必须是*withCode类型
		if coder, ok := codes[v.code]; ok {
			return coder
		}
	}
	return unknownCoder
}

//IsCode 判断某个错误及其错误链上是否有对应错误码的错误
func IsCode(err error, code int) bool {
	if v, ok := err.(*withCode); ok {
		if v.code == code {
			return true
		}

		if v.cause != nil { //如果其还有底层错误,则继续匹配
			return IsCode(v.cause, code)
		}
		return false
	}
	return false
}
