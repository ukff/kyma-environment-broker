package error

import (
	"encoding/json"
	"errors"
	"strings"

	gcli "github.com/kyma-project/kyma-environment-broker/internal/third_party/machinebox/graphql"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	apierr2 "k8s.io/apimachinery/pkg/api/meta"
)

const OperationTimeOutMsg string = "operation has reached the time limit"

type ErrorReporter interface {
	error
	Reason() ErrReason
	Dependency() Dependency
}

// error reporter
type LastError struct {
	message   string
	reason    ErrReason
	component Dependency
}

type LastErrorJSON struct {
	Message   string     `json:"message"`
	Reason    ErrReason  `json:"reason"`
	Component Dependency `json:"component"`
}

type ErrReason string

const (
	ErrMsgNotSet                ErrReason = "err_msg_not_set"
	ErrNotSet                   ErrReason = "err_not_set"
	ErrKEBInternal              ErrReason = "err_keb_internal"
	ErrKEBTimeOut               ErrReason = "err_keb_timeout"
	ErrProvisionerNilLastError  ErrReason = "err_provisioner_nil_last_error"
	ErrHttpStatusCode           ErrReason = "err_http_status_code"
	ErrClusterNotFound          ErrReason = "err_cluster_not_found"
	ErrK8SUnexpectedServerError ErrReason = "err_k8s_unexpected_server_error"
	ErrK8SUnexpectedObjectError ErrReason = "err_k8s_unexpected_object_error"
	ErrK8SNoMatchError          ErrReason = "err_k8s_no_match_error"
	ErrK8SAmbiguousError        ErrReason = "err_k8s_ambiguous_error"
)

type Dependency string

const (
	UnknownDependency     Dependency = "unknown"
	KebDbDependency       Dependency = "db - keb"
	K8sDependency         Dependency = "k8s client - keb"
	KEBDependency         Dependency = "keb"
	EDPDependency         Dependency = "edp"
	ProvisionerDependency Dependency = "provisioner"
	ReconcileDependency   Dependency = "reconciler"
	KIMDependency         Dependency = "kim"
	LMDepedency           Dependency = "lifecycle-manager"
)

func (err LastError) Reason() ErrReason {
	return err.reason
}

func (err LastError) Dependency() Dependency {
	return err.component
}

func (err LastError) Error() string {
	return err.message
}

func (err LastError) SetComponent(component Dependency) LastError {
	err.component = component
	return err
}

func (err LastError) SetReason(reason ErrReason) LastError {
	err.reason = reason
	return err
}

func (err LastError) SetMessage(msg string) LastError {
	err.message = msg
	return err
}

func TimeoutError(msg string) LastError {
	return LastError{
		message:   msg,
		reason:    ErrKEBTimeOut,
		component: KEBDependency,
	}
}

// resolve error component and reason
func ReasonForError(err error) LastError {
	if err == nil {
		return LastError{}
	}

	cause := UnwrapAll(err)

	if lastErr := checkK8SError(cause); lastErr.component == K8sDependency {
		lastErr.message = err.Error()
		return lastErr
	}

	if status := ErrorReporter(nil); errors.As(cause, &status) {
		return LastError{
			message:   err.Error(),
			reason:    status.Reason(),
			component: status.Dependency(),
		}
	}

	if ee, ok := cause.(gcli.ExtendedError); ok {
		var errReason ErrReason
		var errComponent Dependency

		reason, found := ee.Extensions()["error_reason"]
		if found {
			if r, ok := reason.(string); ok {
				errReason = ErrReason(r)
			}
		}
		component, found := ee.Extensions()["error_component"]
		if found {
			if c, ok := component.(string); ok {
				errComponent = Dependency(c)
			}
		}

		return LastError{
			message:   err.Error(),
			reason:    errReason,
			component: errComponent,
		}
	}

	if strings.Contains(err.Error(), OperationTimeOutMsg) {
		return TimeoutError(err.Error())
	}

	return LastError{
		message:   err.Error(),
		reason:    ErrKEBInternal,
		component: KEBDependency,
	}
}

func checkK8SError(cause error) LastError {
	lastErr := LastError{}
	status := apierr.APIStatus(nil)

	switch {
	case errors.As(cause, &status):
		if apierr.IsUnexpectedServerError(cause) {
			lastErr.reason = ErrK8SUnexpectedServerError
		} else {
			// reason could be an empty unknown ""
			lastErr.reason = ErrReason(apierr.ReasonForError(cause))
		}
		lastErr.component = K8sDependency
		return lastErr
	case apierr.IsUnexpectedObjectError(cause):
		lastErr.reason = ErrK8SUnexpectedObjectError
	case apierr2.IsAmbiguousError(cause):
		lastErr.reason = ErrK8SAmbiguousError
	case apierr2.IsNoMatchError(cause):
		lastErr.reason = ErrK8SNoMatchError
	}

	if lastErr.reason != "" {
		lastErr.component = K8sDependency
	}

	return lastErr
}

// UnwrapOnce accesses the direct cause of the error if any, otherwise
// returns nil.
func UnwrapOnce(err error) (cause error) {
	switch e := err.(type) {
	case interface{ Unwrap() error }:
		return e.Unwrap()
	}
	return nil
}

// UnwrapAll accesses the root cause object of the error.
// If the error has no cause (leaf error), it is returned directly.
// this is a replacement for github.com/pkg/errors.Cause
func UnwrapAll(err error) error {
	for {
		if cause := UnwrapOnce(err); cause != nil {
			err = cause
			continue
		}
		break
	}
	return err
}

func (l LastError) MarshalJSON() ([]byte, error) {
	return json.Marshal(
		LastErrorJSON{
			Message:   l.message,
			Reason:    l.reason,
			Component: l.component,
		})
}

func (l *LastError) UnmarshalJSON(data []byte) error {
	tmp := &LastErrorJSON{}
	if err := json.Unmarshal(data, &tmp); err != nil {
		return err
	}
	l.message = tmp.Message
	l.reason = tmp.Reason
	l.component = tmp.Component
	return nil
}

func (ll LastErrorJSON) ToDTO() LastError {
	return LastError{
		message:   ll.Message,
		reason:    ll.Reason,
		component: ll.Component,
	}
}
