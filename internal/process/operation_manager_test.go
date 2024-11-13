package process

import (
	"fmt"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kyma-project/kyma-environment-broker/internal"
	kebErr "github.com/kyma-project/kyma-environment-broker/internal/error"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
)

func Test_OperationManager_RetryOperationOnce(t *testing.T) {
	// given
	memory := storage.NewMemoryStorage()
	operations := memory.Operations()
	opManager := NewOperationManager(operations)
	op := internal.Operation{}
	op.UpdatedAt = time.Now()
	retryInterval := time.Hour
	errMsg := fmt.Errorf("ups ... ")

	// this is required to avoid storage retries (without this statement there will be an error => retry)
	err := operations.InsertOperation(op)
	require.NoError(t, err)

	// then - first call
	op, when, err := opManager.RetryOperationOnce(op, errMsg.Error(), errMsg, retryInterval, fixLogger())

	// when - first retry
	assert.True(t, when > 0)
	assert.Nil(t, err)

	// then - second call
	t.Log(op.UpdatedAt.String())
	op.UpdatedAt = op.UpdatedAt.Add(-retryInterval - time.Second) // simulate wait of first retry
	t.Log(op.UpdatedAt.String())
	op, when, err = opManager.RetryOperationOnce(op, errMsg.Error(), errMsg, retryInterval, fixLogger())

	// when - second call => no retry
	assert.True(t, when == 0)
	assert.NotNil(t, err)
}

func Test_OperationManager_RetryOperation(t *testing.T) {
	// given
	memory := storage.NewMemoryStorage()
	operations := memory.Operations()
	opManager := NewOperationManager(operations)
	op := internal.Operation{}
	op.UpdatedAt = time.Now()
	retryInterval := time.Hour
	errorMessage := "ups ... "
	errOut := fmt.Errorf("error occurred")
	maxtime := time.Hour * 3 // allow 2 retries

	// this is required to avoid storage retries (without this statement there will be an error => retry)
	err := operations.InsertOperation(op)
	require.NoError(t, err)

	// then - first call
	op, when, err := opManager.RetryOperation(op, errorMessage, errOut, retryInterval, maxtime, fixLogger())

	// when - first retry
	assert.True(t, when > 0)
	assert.Nil(t, err)

	// then - second call
	t.Log(op.UpdatedAt.String())
	op.UpdatedAt = op.UpdatedAt.Add(-retryInterval - time.Second) // simulate wait of first retry
	t.Log(op.UpdatedAt.String())
	op, when, err = opManager.RetryOperation(op, errorMessage, errOut, retryInterval, maxtime, fixLogger())

	// when - second call => retry
	assert.True(t, when > 0)
	assert.Nil(t, err)
}

func Test_OperationManager_LastError(t *testing.T) {
	t.Run("when all last error field set with 1 component", func(t *testing.T) {
		memory := storage.NewMemoryStorage()
		operations := memory.Operations()
		opManager := NewOperationManagerExtendent(operations, "some_step", kebErr.ProvisionerDependency)
		op := internal.Operation{}
		err := operations.InsertOperation(op)
		require.NoError(t, err)
		op, _, err = opManager.OperationFailed(op, "friendly message", fmt.Errorf("technical err"), fixLogger())
		assert.EqualValues(t, "provisioner", op.LastError.Dependency())
		assert.EqualValues(t, "technical err", op.LastError.Error())
		assert.EqualValues(t, "friendly message", op.LastError.Reason())
	})

	t.Run("when all last error field set with 3 components", func(t *testing.T) {
		memory := storage.NewMemoryStorage()
		operations := memory.Operations()
		opManager := NewOperationManagerExtendent(operations, "some_step", kebErr.ProvisionerDependency, kebErr.ReconcileDependency, kebErr.KebDbDependency)
		op := internal.Operation{}
		err := operations.InsertOperation(op)
		require.NoError(t, err)
		op, _, err = opManager.OperationFailed(op, "friendly message", fmt.Errorf("technical err"), fixLogger())
		assert.EqualValues(t, "provisioner,reconciler,db - keb", op.LastError.Dependency())
		assert.EqualValues(t, "technical err", op.LastError.Error())
		assert.EqualValues(t, "friendly message", op.LastError.Reason())
	})

	t.Run("when no error passed", func(t *testing.T) {
		memory := storage.NewMemoryStorage()
		operations := memory.Operations()
		opManager := NewOperationManagerExtendent(operations, "some_step", kebErr.ProvisionerDependency, kebErr.ReconcileDependency)
		op := internal.Operation{}
		err := operations.InsertOperation(op)
		require.NoError(t, err)
		op, _, err = opManager.OperationFailed(op, "friendly message", nil, fixLogger())
		assert.EqualValues(t, "provisioner,reconciler", op.LastError.Dependency())
		assert.EqualValues(t, "err_not_set", op.LastError.Error())
		assert.EqualValues(t, "friendly message", op.LastError.Reason())
	})

	t.Run("when no description passed", func(t *testing.T) {
		memory := storage.NewMemoryStorage()
		operations := memory.Operations()
		opManager := NewOperationManagerExtendent(operations, "some_step", kebErr.ProvisionerDependency)
		op := internal.Operation{}
		err := operations.InsertOperation(op)
		require.NoError(t, err)
		op, _, err = opManager.OperationFailed(op, "", fmt.Errorf("technical err"), fixLogger())
		assert.EqualValues(t, "provisioner", op.LastError.Dependency())
		assert.EqualValues(t, "technical err", op.LastError.Error())
		assert.EqualValues(t, "err_msg_not_set", op.LastError.Reason())
	})

	t.Run("when no description and no err passed", func(t *testing.T) {
		memory := storage.NewMemoryStorage()
		operations := memory.Operations()
		opManager := NewOperationManagerExtendent(operations, "some_step", kebErr.ReconcileDependency)
		op := internal.Operation{}
		err := operations.InsertOperation(op)
		require.NoError(t, err)
		op, _, err = opManager.OperationFailed(op, "", nil, fixLogger())
		assert.EqualValues(t, "reconciler", op.LastError.Dependency())
		assert.EqualValues(t, "err_not_set", op.LastError.Error())
		assert.EqualValues(t, "err_msg_not_set", op.LastError.Reason())
	})
}

func fixLogger() logrus.FieldLogger {
	return logrus.StandardLogger()
}
