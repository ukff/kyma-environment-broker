package provisioning

import (
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/fixture"
	"github.com/kyma-project/kyma-environment-broker/internal/ptr"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	technicalNameBtpManager         = "btp-operator"
	technicalNameKeda               = "keda"
	givenKymaTemplateWithModules    = "given_kyma_with_modules.yaml"
	givenKymaTemplateWithoutModules = "given_kyma_without_modules.yaml"
	kymaExpectedNamespace           = "kyma-system"
	overrideKymaModulesPath         = "override_kyma_modules_tests_assets"
)

func execTest(t *testing.T, parameters *internal.ModulesDTO, in, out string) {
	db := storage.NewMemoryStorage()
	operation := fixture.FixOperation(uuid.NewString(), uuid.NewString(), internal.OperationTypeProvision)
	operation.KymaTemplate = internal.GetFileWithTest(t, fmt.Sprintf("%s/%s", overrideKymaModulesPath, in))
	expectedKymaTemplate := internal.GetFileWithTest(t, fmt.Sprintf("%s/%s", overrideKymaModulesPath, out))
	operation.ProvisioningParameters.Parameters.Modules = parameters
	err := db.Operations().InsertOperation(operation)
	assert.NoError(t, err)
	svc := NewOverrideKymaModules(db.Operations())

	// when
	op, backoff, err := svc.Run(operation, logrus.New())
	assert.Zero(t, backoff)
	assert.Equal(t, kymaExpectedNamespace, op.KymaResourceNamespace)
	assert.YAMLEq(t, expectedKymaTemplate, op.KymaTemplate)
	require.NoError(t, err)
}

// when given kyma template without any default modules set

func TestKymaAppendModulesWithEmptyDefaultOnes1_Run(t *testing.T) {
	modules := &internal.ModulesDTO{}
	modules.List = make([]*internal.ModuleDTO, 0)
	m1 := internal.ModuleDTO{
		Name:                 technicalNameBtpManager,
		CustomResourcePolicy: internal.CreateAndDelete,
	}
	m2 := internal.ModuleDTO{
		Name:    technicalNameKeda,
		Channel: internal.Regular,
	}
	modules.List = append(modules.List, &m1, &m2)
	execTest(t, modules, givenKymaTemplateWithoutModules, "kyma-with-keda-and-btp-manager_1.yaml")
}

func TestKymaAppendModulesWithEmptyDefaultOnes2_Run(t *testing.T) {
	modules := &internal.ModulesDTO{}
	modules.List = make([]*internal.ModuleDTO, 0)
	m1 := internal.ModuleDTO{
		Name:                 technicalNameBtpManager,
		CustomResourcePolicy: internal.CreateAndDelete,
	}
	modules.List = append(modules.List, &m1)

	execTest(t, modules, givenKymaTemplateWithoutModules, "kyma-with-btp-manager-2.yaml")
}

func TestKymaAppendModulesWithEmptyDefaultOnes3_Run(t *testing.T) {
	modules := &internal.ModulesDTO{}
	modules.Default = ptr.Bool(true)
	execTest(t, modules, givenKymaTemplateWithoutModules, "kyma-no-modules.yaml")
}

func TestKymaAppendModulesWithEmptyDefaultOnes4_Run(t *testing.T) {
	modules := &internal.ModulesDTO{}
	modules.Default = ptr.Bool(true)
	modules.List = make([]*internal.ModuleDTO, 0)
	m1 := internal.ModuleDTO{
		Name:                 technicalNameBtpManager,
		Channel:              internal.Fast,
		CustomResourcePolicy: internal.CreateAndDelete,
	}
	m2 := internal.ModuleDTO{
		Name:                 technicalNameKeda,
		Channel:              internal.Regular,
		CustomResourcePolicy: internal.CreateAndDelete,
	}
	modules.List = append(modules.List, &m1, &m2)
	execTest(t, modules, givenKymaTemplateWithoutModules, givenKymaTemplateWithoutModules)
}

func TestKymaAppendModulesWithEmptyDefaultOnes5_Run(t *testing.T) {
	modules := &internal.ModulesDTO{}
	modules.List = make([]*internal.ModuleDTO, 0)
	m1 := internal.ModuleDTO{
		Name:                 technicalNameBtpManager,
		Channel:              internal.Fast,
		CustomResourcePolicy: internal.CreateAndDelete,
	}
	modules.List = append(modules.List, &m1)
	execTest(t, modules, givenKymaTemplateWithoutModules, "kyma-with-btp-manager.yaml")
}

func TestKymaAppendModulesWithEmptyDefaultOnes6_Run(t *testing.T) {
	modules := &internal.ModulesDTO{
		Default: nil,
		List:    make([]*internal.ModuleDTO, 0),
	}
	m1 := internal.ModuleDTO{
		Name:                 technicalNameBtpManager,
		Channel:              internal.Fast,
		CustomResourcePolicy: internal.CreateAndDelete,
	}
	modules.List = append(modules.List, &m1)
	execTest(t, modules, givenKymaTemplateWithoutModules, "kyma-with-btp-manager.yaml")
}

func TestKymaAppendModulesWithEmptyDefaultOnes7_Run(t *testing.T) {
	execTest(t, &internal.ModulesDTO{}, givenKymaTemplateWithoutModules, "kyma-no-modules.yaml")
}

func TestKymaAppendModulesWithEmptyDefaultOnes8_Run(t *testing.T) {
	execTest(t, nil, givenKymaTemplateWithoutModules, "kyma-no-modules.yaml")
}

func TestKymaAppendModulesWithEmptyDefaultOnes9_Run(t *testing.T) {
	modules := &internal.ModulesDTO{}
	modules.List = make([]*internal.ModuleDTO, 0)
	execTest(t, modules, givenKymaTemplateWithoutModules, "kyma-no-modules.yaml")
}

func TestKymaAppendModulesWithEmptyDefaultOnes10_Run(t *testing.T) {
	modules := &internal.ModulesDTO{}
	modules.List = nil
	execTest(t, modules, givenKymaTemplateWithoutModules, "kyma-no-modules.yaml")
}

func TestKymaAppendModulesWithEmptyDefaultOnes11_Run(t *testing.T) {
	execTest(t, nil, givenKymaTemplateWithoutModules, "kyma-no-modules.yaml")
}

// when given kyma template have default modules set

func TestKymaAppendModulesWithDefaultOnesSet1_Run(t *testing.T) {
	execTest(t, nil, givenKymaTemplateWithModules, givenKymaTemplateWithModules)
}

func TestKymaAppendModulesWithDefaultOnesSet2_Run(t *testing.T) {
	modules := &internal.ModulesDTO{}
	modules.List = make([]*internal.ModuleDTO, 0)
	m1 := internal.ModuleDTO{
		Name:                 technicalNameBtpManager,
		Channel:              internal.Fast,
		CustomResourcePolicy: internal.CreateAndDelete,
	}
	m2 := internal.ModuleDTO{
		Name:                 technicalNameKeda,
		Channel:              internal.Regular,
		CustomResourcePolicy: internal.Ignore,
	}
	modules.List = append(modules.List, &m1, &m2)
	execTest(t, modules, givenKymaTemplateWithModules, "kyma-with-keda-and-btp-manager_2.yaml")
}

func TestKymaAppendModulesWithDefaultOnesSet3_Run(t *testing.T) {
	modules := &internal.ModulesDTO{}
	modules.Default = ptr.Bool(true)
	modules.List = make([]*internal.ModuleDTO, 0)
	m1 := internal.ModuleDTO{
		Name:                 technicalNameBtpManager,
		Channel:              internal.Regular,
		CustomResourcePolicy: internal.CreateAndDelete,
	}
	modules.List = append(modules.List, &m1)
	execTest(t, modules, givenKymaTemplateWithModules, givenKymaTemplateWithModules)
}

func TestKymaAppendModulesWithDefaultOnesSet4_Run(t *testing.T) {
	execTest(t, &internal.ModulesDTO{}, givenKymaTemplateWithModules, givenKymaTemplateWithModules)
}

func TestKymaAppendModulesWithDefaultOnesSet5_Run(t *testing.T) {
	execTest(t, nil, givenKymaTemplateWithModules, givenKymaTemplateWithModules)
}
