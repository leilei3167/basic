package errors

import (
	stderrors "errors"
	"fmt"
	"io"
)

type baseError struct {
	msg string
	*stack
}

func (b *baseError) Error() string {
	return b.msg
}
func (f *baseError) Format(s fmt.State, verb rune) {
	switch verb {
	case 'v':
		if s.Flag('+') {
			io.WriteString(s, f.msg)
			f.stack.Format(s, verb) //将堆栈格式化
			return
		}
		fallthrough
	case 's':
		io.WriteString(s, f.msg)
	case 'q':
		fmt.Fprintf(s, "%q", f.msg)
	}
}

func New(msg string) error {
	return &baseError{
		msg:   msg,
		stack: callers(),
	}
}

func Errorf(format string, args ...any) error {
	return &baseError{
		msg:   fmt.Sprintf(format, args...),
		stack: callers(),
	}
}

type withStack struct {
	error
	*stack
}

func (w *withStack) Cause() error { return w.error }

func (w *withStack) Unwrap() error { //如果其中的错误有Unwrap方法，则调用该方法
	if e, ok := w.error.(interface{ Unwrap() error }); ok {
		return e.Unwrap()
	}
	return w.error
}

func (w *withStack) Format(s fmt.State, verb rune) {
	switch verb {
	case 'v':
		if s.Flag('+') {
			fmt.Fprintf(s, "%+v", w.Cause())
			w.stack.Format(s, verb)
			return
		}
		fallthrough
	case 's':
		io.WriteString(s, w.Error())
	case 'q':
		fmt.Fprintf(s, "%q", w.Error())
	}
}

func WithStack(err error) error {
	if err == nil {
		return nil
	}

	if e, ok := err.(*withCode); ok {
		return &withCode{
			err:   e.err,
			code:  e.code,
			cause: err,
			stack: callers(),
		}
	}
	return &withStack{err, callers()}

}

type withMessage struct {
	cause error
	msg   string
}

func (w *withMessage) Error() string { return w.msg }
func (w *withMessage) Cause() error  { return w.cause }
func (w *withMessage) Unwrap() error { return w.cause }

func (w *withMessage) Format(s fmt.State, verb rune) {
	switch verb {
	case 'v':
		if s.Flag('+') { //逐层的打印错误
			fmt.Fprintf(s, "%+v\n", w.Cause())
			io.WriteString(s, w.msg)
			return
		}
		fallthrough
	case 's', 'q':
		io.WriteString(s, w.Error())
	}
}

//WithMessage 对指定错误附加上下文信息
func WithMessage(err error, message string) error {
	if err == nil {
		return nil
	}
	return &withMessage{
		cause: err,
		msg:   message,
	}
}

func WithMessagef(err error, format string, args ...any) error {
	if err == nil {
		return nil
	}
	return &withMessage{
		cause: err,
		msg:   fmt.Sprintf(format, args...),
	}
}

type withCode struct {
	err   error
	code  int
	cause error
	*stack
}

func (w *withCode) Error() string { return fmt.Sprintf("%v", w) }
func (w *withCode) Cause() error  { return w.cause }
func (w *withCode) Unwrap() error { return w.cause }

//WithCode 根据业务错误码,创建一个带错误码的错误
func WithCode(code int, format string, args ...any) error {
	return &withCode{
		err:   fmt.Errorf(format, args...),
		code:  code,
		cause: nil,
		stack: callers(),
	}
}

func Wrap(err error, message string) error {
	if err == nil {
		return nil
	}
	//判断是否是带错误码的错误类型,是的话需要保留错误码等信息
	if e, ok := err.(*withCode); ok {
		return &withCode{
			err:   fmt.Errorf(message),
			code:  e.code,
			cause: err,
			stack: callers(),
		}
	}
	err = &withMessage{cause: err, msg: message}
	return &withStack{err, callers()}
}

func Wrapf(err error, format string, args ...any) error {
	if err == nil {
		return nil
	}

	if e, ok := err.(*withCode); ok {
		return &withCode{
			err:   fmt.Errorf(format, args...),
			code:  e.code,
			cause: err,
			stack: callers(),
		}
	}

	err = &withMessage{
		cause: err,
		msg:   fmt.Sprintf(format, args...),
	}
	return &withStack{
		err,
		callers(),
	}
}

func WrapC(err error, code int, format string, args ...any) error {
	if err == nil {
		return nil
	}

	return &withCode{
		err:   fmt.Errorf(format, args...),
		code:  code,
		cause: err,
		stack: callers(),
	}
}

//Cause 返回该错误的底层错误是哪一个
func Cause(err error) error {
	type causer interface{ Cause() error }

	for err != nil {
		cause, ok := err.(causer)
		if !ok { //错误没有Cause方法时退出
			break
		}
		if cause.Cause() == nil { //已达底层
			break
		}
		err = cause.Cause()
	}
	return err
}

//适配标准库

func Is(err, target error) bool {
	return stderrors.Is(err, target)
}

func As(err error, target any) bool {
	return stderrors.As(err, target)
}

func Unwrap(err error) error {
	return stderrors.Unwrap(err)
}
