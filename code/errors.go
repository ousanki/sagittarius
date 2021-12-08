package code

import (
	"fmt"
	"sync"
)

type code interface {
	Error() string
	Cause() error
}

type withMessage struct {
	cause error
	msg   string
}

func (w *withMessage) Error() string { return w.msg + ": " + w.cause.Error() }

func (w *withMessage) Cause() error  { return w.cause }

func WithMessage(err error, message string) error {
	if err == nil {
		return nil
	}
	return &withMessage{
		cause: err,
		msg:   message,
	}
}

type Error struct {
	Code    int
	Message string
}

func (e *Error) Error() string { return fmt.Sprintf("code:%d, message:%s", e.Code, e.Message) }

func (e *Error) Cause() error { return fmt.Errorf("%d", e.Code) }

func Cause(err error) error {
	type causer interface {
		Cause() error
	}

	for err != nil {
		cause, ok := err.(causer)
		if !ok {
			break
		}
		err = cause.Cause()
	}
	return err
}

// code管理
var codes map[int]code
var _codeMu sync.Mutex
var _codeOnce sync.Once

func init() {
	_codeOnce.Do(func() {
		codes = make(map[int]code)
	})
}

func NewCode(code int, message string) code {
	_codeMu.Lock()
	defer _codeMu.Unlock()

	if _, has := codes[code]; has {
		panic("register code already exist error.")
	}
	c := &Error{
		Code:    code,
		Message: message,
	}
	codes[code] = c
	return c
}

func BuildCode(code int, message string) code {
	c := &Error{
		Code:    code,
		Message: message,
	}
	return c
}

func ErrorIs(c code, err error) bool {
	return Cause(c) == Cause(err)
}