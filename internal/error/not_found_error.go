package error

type NotFoundError struct {
}

func (NotFoundError) Error() string {
	return "not found"
}

func (NotFoundError) IsNotFound() bool {
	return true
}

func (NotFoundError) GetReason() Code {
	return ClusterNotFoundCode
}

func (NotFoundError) GetDependency() Dependency {
	return ReconcileDependency
}

func IsNotFoundError(err error) bool {
	cause := UnwrapAll(err)
	nfe, ok := cause.(interface {
		IsNotFound() bool
	})
	return ok && nfe.IsNotFound()
}
