package error

import (
	"fmt"
)

type TemporaryError struct {
	message string
}

func NewTemporaryError(msg string, args ...interface{}) *TemporaryError {
	return &TemporaryError{message: fmt.Sprintf(msg, args...)}
}

func AsTemporaryError(err error, context string, args ...interface{}) *TemporaryError {
	errCtx := fmt.Sprintf(context, args...)
	msg := fmt.Sprintf("%s: %s", errCtx, err.Error())

	return &TemporaryError{message: msg}
}

func (te TemporaryError) Error() string       { return te.message }
func (TemporaryError) Temporary() bool        { return true }
func (TemporaryError) Reason() Code           { return KEBInternalCode }
func (TemporaryError) Dependency() Dependency { return KEBDependency }

func IsTemporaryError(err error) bool {
	cause := UnwrapAll(err)

	nfe, ok := cause.(interface {
		Temporary() bool
	})

	return ok && nfe.Temporary()
}

// can be used for temporary error
// but still storing the original error in case returned to Execute
type WrapTemporaryError struct {
	err error
}

func WrapAsTemporaryError(err error, msg string, args ...interface{}) *WrapTemporaryError {
	return &WrapTemporaryError{err: fmt.Errorf(fmt.Sprintf(msg, args...)+" :%w", err)}
}

func WrapNewTemporaryError(err error) *WrapTemporaryError {
	return &WrapTemporaryError{err: err}
}

func (te WrapTemporaryError) Error() string { return te.err.Error() }
func (WrapTemporaryError) Temporary() bool  { return true }

func (wte WrapTemporaryError) Reason() Code {
	return ReasonForError(wte.err).Reason()
}

func (wte WrapTemporaryError) Dependency() Dependency {
	return ReasonForError(wte.err).Dependency()
}
