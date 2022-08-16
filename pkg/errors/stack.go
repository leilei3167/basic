package errors

import (
	"fmt"
	"io"
	"path"
	"runtime"
	"strconv"
	"strings"
)

const (
	stackDepth = 32
)

type Frame uintptr //代表堆栈中的一个帧,他的值是程序计数器+1

func (f Frame) pc() uintptr { return uintptr(f) - 1 }

func (f Frame) file() string { //根据给定的帧,返回当前程序计数器所在的函数的所在文件名
	fn := runtime.FuncForPC(f.pc())
	if fn == nil {
		return "unknown"
	}
	file, _ := fn.FileLine(f.pc())
	return file
}
func (f Frame) line() int { //根据给定的帧,返回当前程序计数器所在的函数函数所在行号
	fn := runtime.FuncForPC(f.pc())
	if fn == nil {
		return 0
	}
	_, line := fn.FileLine(f.pc())
	return line
}

func (f Frame) name() string { //返回当前计数器所在的函数名称
	fn := runtime.FuncForPC(f.pc())
	if fn == nil {
		return "unknown"
	}
	return fn.Name()
}

func (f Frame) Format(s fmt.State, verb rune) {
	switch verb {
	case 's':
		switch {
		case s.Flag('+'): //打印出文件名函数名
			//打印堆栈信息
			io.WriteString(s, f.name())
			io.WriteString(s, "\n\t")
			io.WriteString(s, f.file())
		default:
			io.WriteString(s, path.Base(f.file())) //默认只打印文件的名称
		}

	case 'd':
		io.WriteString(s, strconv.Itoa(f.line()))
	case 'n':
		io.WriteString(s, funcname(f.name()))
	case 'v':
		f.Format(s, 's')
		io.WriteString(s, ":")
		f.Format(s, 'd')
	}
}
func funcname(name string) string { //只取最后的函数名
	i := strings.LastIndex(name, "/")
	name = name[i+1:]
	i = strings.Index(name, ".")
	return name[i+1:]
}

func (f Frame) MarshalText() ([]byte, error) {
	name := f.name()
	if name == "unknown" {
		return []byte(name), nil
	}
	return []byte(fmt.Sprintf("%s %s:%d", name, f.file(), f.line())), nil

}

type StackTrace []Frame

func (st StackTrace) Format(s fmt.State, verb rune) {
	switch verb {
	case 'v':
		switch {
		case s.Flag('+'):
			//打印所有的帧
			for _, f := range st {
				io.WriteString(s, "\n")
				f.Format(s, verb)
			}
		case s.Flag('#'):
			fmt.Fprintf(s, "%#v", []Frame(st))
		default:
			st.formatSlice(s, verb)
		}

	case 's':
		st.formatSlice(s, verb)
	}
}

func (st StackTrace) formatSlice(s fmt.State, verb rune) {
	io.WriteString(s, "[")
	for i, f := range st {
		if i > 0 {
			io.WriteString(s, " ")
		}
		f.Format(s, verb)
	}
	io.WriteString(s, "]")
}

//用于记录堆栈深度
type stack []uintptr //代表调用堆栈,由许多个帧组成

func callers() *stack {
	var pcs [stackDepth]uintptr
	n := runtime.Callers(3, pcs[:]) // callers自身(1),callers这个函数(2),调用callers的函数3(如withstack),跳过记录这3个函数
	var st stack = pcs[0:n]
	return &st
}

func (s *stack) Format(st fmt.State, verb rune) {
	switch verb {
	case 'v':
		switch {
		case st.Flag('+'):
			for _, pc := range *s {
				f := Frame(pc)
				fmt.Fprintf(st, "\n%+v", f)
			}
		}
	}
}

func (s *stack) StackTrace() StackTrace {
	f := make([]Frame, len(*s))
	for i := 0; i < len(f); i++ {
		f[i] = Frame((*s)[i]) //将*stack中的每一个堆栈转换为帧
	}
	return f
}
