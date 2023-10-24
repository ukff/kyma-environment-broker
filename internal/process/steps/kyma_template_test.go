package steps

import (
	"testing"

	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/fixture"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInitKymaTemplate_Run(t *testing.T) {
	// given
	db := storage.NewMemoryStorage()
	operation := fixture.FixOperation("op-id", "inst-id", internal.OperationTypeProvision)
	operation.ProvisioningParameters.Parameters.Modules = nil
	db.Operations().InsertOperation(operation)
	svc := NewInitKymaTemplate(db.Operations())
	ic := fixture.FixInputCreator("aws")
	ic.Config = &internal.ConfigForPlan{
		KymaTemplate: getYaml(t, withoutDefaultModules, "default.yaml"),
	}
	operation.InputCreator = ic

	// when
	op, backoff, err := svc.Run(operation, logrus.New())
	require.NoError(t, err)

	// then
	assert.Zero(t, backoff)
	assert.Equal(t, "kyma-system", op.KymaResourceNamespace)

	assert.YAMLEq(t, op.KymaTemplate, ic.Config.KymaTemplate)
}
