package errors

import (
	"fmt"
	"reflect"

	"github.com/pkg/errors"
)

type Code int

const (
	CodeGeneral Code = iota * 1000
)

const (
	UnknownError Code = CodeGeneral + iota
	IllegalArgumentError
	UnsupportedError
	InvalidStateError
	NotFoundError
	InvalidNetworkError
	TimeoutError
)

var (
	ErrUnknown         = NewBase(UnknownError, "UnknownError")
	ErrIllegalArgument = NewBase(IllegalArgumentError, "IllegalArgument")
	ErrInvalidState    = NewBase(InvalidStateError, "InvalidState")
	ErrUnsupported     = NewBase(UnsupportedError, "Unsupported")
	ErrNotFound        = NewBase(NotFoundError, "NotFound")
	ErrInvalidNetwork  = NewBase(InvalidNetworkError, "InvalidNetwork")
	ErrTimeout         = NewBase(TimeoutError, "Timeout")
)

func (c Code) New(msg string) error {
	return Errorc(c, msg)
}

func (c Code) Errorf(f string, args ...interface{}) error {
	return Errorcf(c, f, args...)
}

func (c Code) Wrap(e error, msg string) error {
	return Wrapc(e, c, msg)
}

func (c Code) Wrapf(e error, f string, args ...interface{}) error {
	return Wrapcf(e, c, f, args...)
}

func (c Code) Equals(e error) bool {
	if e == nil {
		return false
	}
	return CodeOf(e) == c
}

/*------------------------------------------------------------------------------
Simple mapping to github.com/pkg/errors for easy stack print
*/

// New makes an error including a stack without any code
// If you want to make base error without stack
func New(msg string) error {
	return errors.New(msg)
}

func Errorf(f string, args ...interface{}) error {
	return errors.Errorf(f, args...)
}

func WithStack(e error) error {
	return errors.WithStack(e)
}

/*------------------------------------------------------------------------------
Base error only with message and code

For general usage, you may return this directly.
*/

type baseError struct {
	code Code
	msg  string
}

func (e *baseError) Error() string {
	return e.msg
}

func (e *baseError) ErrorCode() Code {
	return e.code
}

func (e *baseError) Format(f fmt.State, c rune) {
	switch c {
	case 'v', 's', 'q':
		fmt.Fprintf(f, "E%04d:%s", e.code, e.msg)
	}
}

func (e *baseError) Equals(err error) bool {
	if err == nil {
		return false
	}
	return CodeOf(err) == e.code
}

func NewBase(code Code, msg string) *baseError {
	return &baseError{code, msg}
}

/*------------------------------------------------------------------------------
Coded error object
*/

type codedError struct {
	code Code
	error
}

func (e *codedError) Format(f fmt.State, c rune) {
	switch c {
	case 'v':
		if f.Flag('+') {
			fmt.Fprintf(f, "E%04d:%+v", e.code, e.error)
			return
		}
		fallthrough
	case 's', 'q':
		fmt.Fprintf(f, "E%04d:%s", e.code, e.Error())
	}
}

func (e *codedError) ErrorCode() Code {
	return e.code
}

func (e *codedError) Unwrap() error {
	return e.error
}

func Errorc(code Code, msg string) error {
	return &codedError{
		code:  code,
		error: errors.New(msg),
	}
}

func Errorcf(code Code, f string, args ...interface{}) error {
	return &codedError{
		code:  code,
		error: errors.Errorf(f, args...),
	}
}

func WithCode(err error, code Code) error {
	if _, ok := CoderOf(err); ok {
		return Wrapc(err, code, err.Error())
	}
	return &codedError{
		code:  code,
		error: err,
	}
}

type messageError struct {
	error
	origin error
}

func (e *messageError) Format(f fmt.State, c rune) {
	switch c {
	case 'v':
		if f.Flag('+') {
			fmt.Fprintf(f, "%+v", e.error)
			fmt.Fprintf(f, "\nWrapping %+v", e.origin)
			return
		}
		fallthrough
	case 's', 'q':
		fmt.Fprintf(f, "%s", e.error)
	}
}

func (e *messageError) Unwrap() error {
	return e.origin
}

func Wrap(e error, msg string) error {
	return &messageError{
		error:  errors.New(msg),
		origin: e,
	}
}

func Wrapf(e error, f string, args ...interface{}) error {
	return &messageError{
		error:  errors.Errorf(f, args...),
		origin: e,
	}
}

type wrappedError struct {
	error
	code   Code
	origin error
}

func (e *wrappedError) Format(f fmt.State, c rune) {
	switch c {
	case 'v':
		if f.Flag('+') {
			fmt.Fprintf(f, "E%04d:%+v", e.code, e.error)
			fmt.Fprintf(f, "\nWrapping %+v", e.origin)
			return
		}
		fallthrough
	case 'q', 's':
		fmt.Fprintf(f, "E%04d:%s", e.code, e.error)
	}
}

func (e *wrappedError) Unwrap() error {
	return e.origin
}

func (e *wrappedError) ErrorCode() Code {
	return e.code
}

func Wrapc(e error, c Code, msg string) error {
	return &wrappedError{
		error:  errors.New(msg),
		code:   c,
		origin: e,
	}
}

func Wrapcf(e error, c Code, f string, args ...interface{}) error {
	return &wrappedError{
		error:  errors.Errorf(f, args...),
		code:   c,
		origin: e,
	}
}

type ErrorCoder interface {
	error
	ErrorCode() Code
}

func CoderOf(e error) (ErrorCoder, bool) {
	var coder ErrorCoder
	if AsValue(&coder, e) {
		return coder, true
	}
	return nil, false
}

func CodeOf(e error) Code {
	if coder, ok := CoderOf(e); ok {
		return coder.ErrorCode()
	}
	return UnknownError
}

// Unwrapper is interface to unwrap the error to get the origin error.
type Unwrapper interface {
	Unwrap() error
}

// Is checks whether err is caused by the target.
func Is(err, target error) bool {
	type causer interface {
		Cause() error
	}

	type unwrapper interface {
		Unwrap() error
	}

	for {
		if err == target {
			return true
		}
		if cause, ok := err.(causer); ok {
			err = cause.Cause()
		} else if unwrap, ok := err.(unwrapper); ok {
			err = unwrap.Unwrap()
		} else {
			return false
		}
	}
}

// AsValue checks whether the err is caused by specified typed error, and
// store it to the ptr.
func AsValue(ptr interface{}, err error) bool {
	type causer interface {
		Cause() error
	}

	type unwrapper interface {
		Unwrap() error
	}

	value := reflect.ValueOf(ptr)
	if value.Kind() != reflect.Ptr {
		return false
	} else {
		value = value.Elem()
	}
	valueType := value.Type()

	for {
		errValue := reflect.ValueOf(err)
		if errValue.Type().AssignableTo(valueType) {
			value.Set(errValue)
			return true
		}
		if cause, ok := err.(causer); ok {
			err = cause.Cause()
		} else if unwrap, ok := err.(unwrapper); ok {
			err = unwrap.Unwrap()
		} else {
			return false
		}
	}
}
