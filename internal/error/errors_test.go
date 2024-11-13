package error_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/kyma-project/kyma-environment-broker/internal/edp"
	kebError "github.com/kyma-project/kyma-environment-broker/internal/error"
	"github.com/kyma-project/kyma-environment-broker/internal/storage/dberr"
	"github.com/stretchr/testify/assert"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	apierr2 "k8s.io/apimachinery/pkg/api/meta"
)

func TestLastError(t *testing.T) {
	t.Run("report correct reason and component", func(t *testing.T) {
		// given
		edpErr := edp.NewEDPBadRequestError("id", fmt.Sprintf("Bad request: %s", "response"))
		expectEdpMsg := fmt.Sprintf("Bad request: %s", "response")

		edpConfErr := edp.NewEDPConflictError("id", fmt.Sprintf("Resource %s already exists", "id"))
		expectEdpConfMsg := "Resource id already exists"

		dbErr := fmt.Errorf("something: %w", dberr.NotFound("Some NotFound apperror, %s", "Some pkg err"))
		expectDbErr := fmt.Sprintf("something: Some NotFound apperror, Some pkg err")

		timeoutErr := fmt.Errorf("something: %w", fmt.Errorf("operation has reached the time limit: 2h"))
		expectTimeoutMsg := "something: operation has reached the time limit: 2h"

		// when
		edpLastErr := kebError.ReasonForError(edpErr)
		edpConfLastErr := kebError.ReasonForError(edpConfErr)
		dbLastErr := kebError.ReasonForError(dbErr)
		timeoutLastErr := kebError.ReasonForError(timeoutErr)

		// then
		assert.Equal(t, edp.ErrEDPBadRequest, edpLastErr.Reason())
		assert.Equal(t, kebError.EDPDependency, edpLastErr.Dependency())
		assert.Equal(t, expectEdpMsg, edpLastErr.Error())

		assert.Equal(t, edp.ErrEDPConflict, edpConfLastErr.Reason())
		assert.Equal(t, kebError.EDPDependency, edpConfLastErr.Dependency())
		assert.Equal(t, expectEdpConfMsg, edpConfLastErr.Error())
		assert.True(t, edp.IsConflictError(edpConfErr))

		assert.Equal(t, dberr.ErrDBNotFound, dbLastErr.Reason())
		assert.Equal(t, kebError.KebDbDependency, dbLastErr.Dependency())
		assert.Equal(t, expectDbErr, dbLastErr.Error())

		assert.Equal(t, kebError.ErrKEBTimeOut, timeoutLastErr.Reason())
		assert.Equal(t, kebError.KEBDependency, timeoutLastErr.Dependency())
		assert.Equal(t, expectTimeoutMsg, timeoutLastErr.Error())
	})
}

func TestTemporaryErrorToLastError(t *testing.T) {
	t.Run("wrapped temporary error", func(t *testing.T) {
		// given
		err := kebError.LastError{}.
			SetMessage(fmt.Sprintf("Got status %d", 502)).
			SetReason(kebError.ErrHttpStatusCode).
			SetComponent(kebError.ReconcileDependency)
		tempErr := fmt.Errorf("something else: %w", kebError.WrapNewTemporaryError(fmt.Errorf("something: %w", err)))
		expectMsg := fmt.Sprintf("something else: something: Got status %d", 502)

		edpTempErr := kebError.WrapNewTemporaryError(edp.NewEDPOtherError("id", http.StatusRequestTimeout, "EDP server returns failed status %s", "501"))
		expectEdpMsg := fmt.Sprintf("EDP server returns failed status %s", "501")

		// when
		lastErr := kebError.ReasonForError(tempErr)
		edpLastErr := kebError.ReasonForError(edpTempErr)

		// then
		assert.Equal(t, kebError.ErrHttpStatusCode, lastErr.Reason())
		assert.Equal(t, kebError.ReconcileDependency, lastErr.Dependency())
		assert.Equal(t, expectMsg, lastErr.Error())
		assert.True(t, kebError.IsTemporaryError(tempErr))

		assert.Equal(t, edp.ErrEDPTimeout, edpLastErr.Reason())
		assert.Equal(t, kebError.EDPDependency, edpLastErr.Dependency())
		assert.Equal(t, expectEdpMsg, edpLastErr.Error())
		assert.True(t, kebError.IsTemporaryError(edpTempErr))
	})

	t.Run("new temporary error", func(t *testing.T) {
		// given
		tempErr := fmt.Errorf("something: %w", kebError.NewTemporaryError("temporary error..."))
		expectMsg := "something: temporary error..."

		// when
		lastErr := kebError.ReasonForError(tempErr)

		// then
		assert.Equal(t, kebError.ErrKEBInternal, lastErr.Reason())
		assert.Equal(t, kebError.KEBDependency, lastErr.Dependency())
		assert.Equal(t, expectMsg, lastErr.Error())
		assert.True(t, kebError.IsTemporaryError(tempErr))
	})
}

func TestNotFoundError(t *testing.T) {
	// given
	err := fmt.Errorf("something: %w", kebError.NotFoundError{})

	// when
	lastErr := kebError.ReasonForError(err)

	// then
	assert.EqualError(t, lastErr, "something: not found")
	assert.Equal(t, kebError.ErrClusterNotFound, lastErr.Reason())
	assert.Equal(t, kebError.ReconcileDependency, lastErr.Dependency())
	assert.True(t, kebError.IsNotFoundError(err))
}

func TestK8SLastError(t *testing.T) {
	// given
	errBadReq := fmt.Errorf("something: %w", apierr.NewBadRequest("bad request here"))
	errUnexpObj := fmt.Errorf("something: %w", &apierr.UnexpectedObjectError{})
	errAmbi := fmt.Errorf("something: %w", &apierr2.AmbiguousResourceError{})
	errNoMatch := fmt.Errorf("something: %w", &apierr2.NoKindMatchError{})

	// when
	lastErrBadReq := kebError.ReasonForError(errBadReq)
	lastErrUnexpObj := kebError.ReasonForError(errUnexpObj)
	lastErrAmbi := kebError.ReasonForError(errAmbi)
	lastErrNoMatch := kebError.ReasonForError(errNoMatch)

	// then
	assert.EqualError(t, lastErrBadReq, "something: bad request here")
	assert.Equal(t, kebError.ErrReason("BadRequest"), lastErrBadReq.Reason())
	assert.Equal(t, kebError.K8sDependency, lastErrBadReq.Dependency())

	assert.ErrorContains(t, lastErrUnexpObj, "something: unexpected object: ")
	assert.Equal(t, kebError.ErrK8SUnexpectedObjectError, lastErrUnexpObj.Reason())
	assert.Equal(t, kebError.K8sDependency, lastErrUnexpObj.Dependency())

	assert.ErrorContains(t, lastErrAmbi, "matches multiple resources or kinds")
	assert.Equal(t, kebError.ErrK8SAmbiguousError, lastErrAmbi.Reason())
	assert.Equal(t, kebError.K8sDependency, lastErrAmbi.Dependency())

	assert.ErrorContains(t, lastErrNoMatch, "something: no matches for kind")
	assert.Equal(t, kebError.ErrK8SNoMatchError, lastErrNoMatch.Reason())
	assert.Equal(t, kebError.K8sDependency, lastErrNoMatch.Dependency())
}
