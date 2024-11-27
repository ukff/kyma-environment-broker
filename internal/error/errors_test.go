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
		edpLastErr := kebError.ReasonForError(edpErr, "")
		edpConfLastErr := kebError.ReasonForError(edpConfErr, "")
		dbLastErr := kebError.ReasonForError(dbErr, "")
		timeoutLastErr := kebError.ReasonForError(timeoutErr, "")

		// then
		assert.Equal(t, edp.ErrEDPBadRequest, edpLastErr.GetReason())
		assert.Equal(t, kebError.EDPDependency, edpLastErr.GetDependency())
		assert.Equal(t, expectEdpMsg, edpLastErr.Error())

		assert.Equal(t, edp.ErrEDPConflict, edpConfLastErr.GetReason())
		assert.Equal(t, kebError.EDPDependency, edpConfLastErr.GetDependency())
		assert.Equal(t, expectEdpConfMsg, edpConfLastErr.Error())
		assert.True(t, edp.IsConflictError(edpConfErr))

		assert.Equal(t, dberr.ErrDBNotFound, dbLastErr.GetReason())
		assert.Equal(t, kebError.KebDbDependency, dbLastErr.GetDependency())
		assert.Equal(t, expectDbErr, dbLastErr.Error())

		assert.Equal(t, kebError.KEBTimeOutCode, timeoutLastErr.GetReason())
		assert.Equal(t, kebError.KEBDependency, timeoutLastErr.GetDependency())
		assert.Equal(t, expectTimeoutMsg, timeoutLastErr.Error())
	})
}

func TestTemporaryErrorToLastError(t *testing.T) {
	t.Run("wrapped temporary error", func(t *testing.T) {
		// given
		err := kebError.LastError{}.
			SetMessage(fmt.Sprintf("Got status %d", 502)).
			SetReason(kebError.HttpStatusCode).
			SetComponent(kebError.ReconcileDependency)
		tempErr := fmt.Errorf("something else: %w", kebError.WrapNewTemporaryError(fmt.Errorf("something: %w", err)))
		expectMsg := fmt.Sprintf("something else: something: Got status %d", 502)

		edpTempErr := kebError.WrapNewTemporaryError(edp.NewEDPOtherError("id", http.StatusRequestTimeout, "EDP server returns failed status %s", "501"))
		expectEdpMsg := fmt.Sprintf("EDP server returns failed status %s", "501")

		// when
		lastErr := kebError.ReasonForError(tempErr, "")
		edpLastErr := kebError.ReasonForError(edpTempErr, "")

		// then
		assert.Equal(t, kebError.HttpStatusCode, lastErr.GetReason())
		assert.Equal(t, kebError.ReconcileDependency, lastErr.GetDependency())
		assert.Equal(t, expectMsg, lastErr.Error())
		assert.True(t, kebError.IsTemporaryError(tempErr))

		assert.Equal(t, edp.ErrEDPTimeout, edpLastErr.GetReason())
		assert.Equal(t, kebError.EDPDependency, edpLastErr.GetDependency())
		assert.Equal(t, expectEdpMsg, edpLastErr.Error())
		assert.True(t, kebError.IsTemporaryError(edpTempErr))
	})

	t.Run("new temporary error", func(t *testing.T) {
		// given
		tempErr := fmt.Errorf("something: %w", kebError.NewTemporaryError("temporary error..."))
		expectMsg := "something: temporary error..."

		// when
		lastErr := kebError.ReasonForError(tempErr, "")

		// then
		assert.Equal(t, kebError.KEBInternalCode, lastErr.GetReason())
		assert.Equal(t, kebError.KEBDependency, lastErr.GetDependency())
		assert.Equal(t, expectMsg, lastErr.Error())
		assert.True(t, kebError.IsTemporaryError(tempErr))
	})
}

func TestNotFoundError(t *testing.T) {
	// given
	err := fmt.Errorf("something: %w", kebError.NotFoundError{})

	// when
	lastErr := kebError.ReasonForError(err, "")

	// then
	assert.EqualError(t, lastErr, "something: not found")
	assert.Equal(t, kebError.ClusterNotFoundCode, lastErr.GetReason())
	assert.Equal(t, kebError.ReconcileDependency, lastErr.GetDependency())
	assert.True(t, kebError.IsNotFoundError(err))
}

func TestK8SLastError(t *testing.T) {
	// given
	errBadReq := fmt.Errorf("something: %w", apierr.NewBadRequest("bad request here"))
	errUnexpObj := fmt.Errorf("something: %w", &apierr.UnexpectedObjectError{})
	errAmbi := fmt.Errorf("something: %w", &apierr2.AmbiguousResourceError{})
	errNoMatch := fmt.Errorf("something: %w", &apierr2.NoKindMatchError{})

	// when
	lastErrBadReq := kebError.ReasonForError(errBadReq, "")
	lastErrUnexpObj := kebError.ReasonForError(errUnexpObj, "")
	lastErrAmbi := kebError.ReasonForError(errAmbi, "")
	lastErrNoMatch := kebError.ReasonForError(errNoMatch, "")

	// then
	assert.EqualError(t, lastErrBadReq, "something: bad request here")
	assert.Equal(t, kebError.Reason("BadRequest"), lastErrBadReq.GetReason())
	assert.Equal(t, kebError.K8sDependency, lastErrBadReq.GetDependency())

	assert.ErrorContains(t, lastErrUnexpObj, "something: unexpected object: ")
	assert.Equal(t, kebError.K8SUnexpectedObjectCode, lastErrUnexpObj.GetReason())
	assert.Equal(t, kebError.K8sDependency, lastErrUnexpObj.GetDependency())

	assert.ErrorContains(t, lastErrAmbi, "matches multiple resources or kinds")
	assert.Equal(t, kebError.K8SAmbiguousCode, lastErrAmbi.GetReason())
	assert.Equal(t, kebError.K8sDependency, lastErrAmbi.GetDependency())

	assert.ErrorContains(t, lastErrNoMatch, "something: no matches for kind")
	assert.Equal(t, kebError.K8SNoMatchCode, lastErrNoMatch.GetReason())
	assert.Equal(t, kebError.K8sDependency, lastErrNoMatch.GetDependency())
}
