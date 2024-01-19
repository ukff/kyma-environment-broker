package kubeconfig

type NotFoundError struct {
	msg string
}

func NewNotFoundError(msg string) error {
	return &NotFoundError{msg: msg}
}

func (e *NotFoundError) Error() string {
	return e.msg
}

func (e *NotFoundError) IsNotFound() bool {
	return true
}

func IsNotFound(err error) bool {
	nf, ok := err.(interface {
		IsNotFound() bool
	})
	return ok && nf.IsNotFound()
}
