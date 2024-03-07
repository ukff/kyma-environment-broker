package deprovisioning

import (
	"testing"

	"github.com/kyma-project/kyma-environment-broker/internal/fixture"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestName(t *testing.T) {

}

func TestArchiveRun(t *testing.T) {
	db := storage.NewMemoryStorage()
	step := NewArchivingStep(db.Operations(), db.Instances(), db.InstancesArchived(), false)

	logger := logrus.New()
	provisioningOperation := fixture.FixProvisioningOperation("op-prov", "inst-id")
	deprovisioningOperation := fixture.FixDeprovisioningOperationAsOperation("op-depr", "inst-id")

	instance := fixture.FixInstance("inst-id")

	err := db.Operations().InsertOperation(provisioningOperation)
	assert.NoError(t, err)
	err = db.Operations().InsertOperation(deprovisioningOperation)
	assert.NoError(t, err)
	err = db.Instances().Insert(instance)
	assert.NoError(t, err)

	_, backoff, err := step.Run(deprovisioningOperation, logger)

	// then
	require.NoError(t, err)
	require.Zero(t, backoff)

	archived, err := db.InstancesArchived().GetByInstanceID("inst-id")
	require.NoError(t, err)
	require.NotNil(t, archived)
}
