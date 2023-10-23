package steps

import (
	"fmt"
	"os"
	"testing"

	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/fixture"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	kymaTemplateTestAssets = "kyma_template_test_assets"
	withDefaultModules     = "with_default_modules"
	withoutDefaultModules  = "without_default_modules"
)

func getYaml(t *testing.T, path, name string) string {
	file, err := os.ReadFile(fmt.Sprintf("%s/%s/%s", kymaTemplateTestAssets, path, name))
	assert.NoError(t, err)
	return string(file)
}

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

func TestInitKymaTemplateWithModules1_Run(t *testing.T) {
	getTestCase1Params := func() *internal.ModulesDTO {
		modules := &internal.ModulesDTO{}
		modules.List = make([]*internal.ModuleDTO, 0)
		m1 := internal.ModuleDTO{
			Name:                 "btp",
			Channel:              internal.Fast,
			CustomResourcePolicy: internal.CreateAndDelete,
		}
		m2 := internal.ModuleDTO{
			Name:                 "keda",
			Channel:              internal.Regular,
			CustomResourcePolicy: internal.CreateAndDelete,
		}
		modules.List = append(modules.List, &m1, &m2)
		return modules
	}

	// given
	db := storage.NewMemoryStorage()
	operation := fixture.FixOperation("op-id", "inst-id", internal.OperationTypeProvision)
	operation.ProvisioningParameters.Parameters.Modules = getTestCase1Params()
	db.Operations().InsertOperation(operation)
	svc := NewInitKymaTemplate(db.Operations())
	ic := fixture.FixInputCreator("aws")
	ic.Config = &internal.ConfigForPlan{
		KymaTemplate: getYaml(t, withoutDefaultModules, "default.yaml"),
	}
	operation.InputCreator = ic

	KymaTemplateOutput := getYaml(t, withoutDefaultModules, "testcase1.yaml")

	// when
	op, backoff, err := svc.Run(operation, logrus.New())
	require.NoError(t, err)

	// then
	assert.Zero(t, backoff)
	assert.Equal(t, "kyma-system", op.KymaResourceNamespace)

	assert.YAMLEq(t, op.KymaTemplate, KymaTemplateOutput)
}

func TestInitKymaTemplateWithModules2_Run(t *testing.T) {
	getTestCase2Params := func() *internal.ModulesDTO {
		modules := &internal.ModulesDTO{}
		modules.List = make([]*internal.ModuleDTO, 0)
		m1 := internal.ModuleDTO{
			Name:                 "btp",
			Channel:              internal.Fast,
			CustomResourcePolicy: internal.CreateAndDelete,
		}
		modules.List = append(modules.List, &m1)
		return modules
	}

	// given
	db := storage.NewMemoryStorage()
	operation := fixture.FixOperation("op-id", "inst-id", internal.OperationTypeProvision)
	operation.ProvisioningParameters.Parameters.Modules = getTestCase2Params()
	db.Operations().InsertOperation(operation)
	svc := NewInitKymaTemplate(db.Operations())
	ic := fixture.FixInputCreator("aws")
	ic.Config = &internal.ConfigForPlan{
		KymaTemplate: getYaml(t, withoutDefaultModules, "default.yaml"),
	}
	operation.InputCreator = ic

	KymaTemplateOutput := getYaml(t, withoutDefaultModules, "testcase2.yaml")

	// when
	op, backoff, err := svc.Run(operation, logrus.New())
	require.NoError(t, err)

	// then
	assert.Zero(t, backoff)
	assert.Equal(t, "kyma-system", op.KymaResourceNamespace)

	assert.YAMLEq(t, op.KymaTemplate, KymaTemplateOutput)
}

func TestInitKymaTemplateWithModules3_Run(t *testing.T) {
	getTestCase3Params := func() *internal.ModulesDTO {
		return &internal.ModulesDTO{}
	}

	// given
	db := storage.NewMemoryStorage()
	operation := fixture.FixOperation("op-id", "inst-id", internal.OperationTypeProvision)
	operation.ProvisioningParameters.Parameters.Modules = getTestCase3Params()
	db.Operations().InsertOperation(operation)
	svc := NewInitKymaTemplate(db.Operations())
	ic := fixture.FixInputCreator("aws")
	ic.Config = &internal.ConfigForPlan{
		KymaTemplate: getYaml(t, withoutDefaultModules, "default.yaml"),
	}
	operation.InputCreator = ic

	KymaTemplateOutput := getYaml(t, withoutDefaultModules, "testcase3.yaml")

	// when
	op, backoff, err := svc.Run(operation, logrus.New())
	require.NoError(t, err)

	// then
	assert.Zero(t, backoff)
	assert.Equal(t, "kyma-system", op.KymaResourceNamespace)

	assert.YAMLEq(t, op.KymaTemplate, KymaTemplateOutput)
}

func TestInitKymaTemplateWithModules4_Run(t *testing.T) {
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

	KymaTemplateOutput := getYaml(t, withoutDefaultModules, "testcase4.yaml")

	// when
	op, backoff, err := svc.Run(operation, logrus.New())
	require.NoError(t, err)

	// then
	assert.Zero(t, backoff)
	assert.Equal(t, "kyma-system", op.KymaResourceNamespace)

	assert.YAMLEq(t, op.KymaTemplate, KymaTemplateOutput)
}

func TestInitKymaTemplateWithModules5_Run(t *testing.T) {
	getTestCase5Params := func() *internal.ModulesDTO {
		modules := &internal.ModulesDTO{}
		modules.List = make([]*internal.ModuleDTO, 0)
		return modules
	}

	// given
	db := storage.NewMemoryStorage()
	operation := fixture.FixOperation("op-id", "inst-id", internal.OperationTypeProvision)
	operation.ProvisioningParameters.Parameters.Modules = getTestCase5Params()
	db.Operations().InsertOperation(operation)
	svc := NewInitKymaTemplate(db.Operations())
	ic := fixture.FixInputCreator("aws")
	ic.Config = &internal.ConfigForPlan{
		KymaTemplate: getYaml(t, withoutDefaultModules, "default.yaml"),
	}
	operation.InputCreator = ic

	KymaTemplateOutput := getYaml(t, withoutDefaultModules, "testcase5.yaml")

	// when
	op, backoff, err := svc.Run(operation, logrus.New())
	require.NoError(t, err)

	// then
	assert.Zero(t, backoff)
	assert.Equal(t, "kyma-system", op.KymaResourceNamespace)

	assert.YAMLEq(t, op.KymaTemplate, KymaTemplateOutput)
}

func TestInitKymaTemplateWithModules6_Run(t *testing.T) {
	getTestCase6Params := func() *internal.ModulesDTO {
		modules := &internal.ModulesDTO{}
		modules.List = make([]*internal.ModuleDTO, 0)
		m1 := internal.ModuleDTO{
			Name:                 "keda",
			Channel:              internal.Regular,
			CustomResourcePolicy: internal.CreateAndDelete,
		}
		modules.List = append(modules.List, &m1)
		return modules
	}

	// given
	db := storage.NewMemoryStorage()
	operation := fixture.FixOperation("op-id", "inst-id", internal.OperationTypeProvision)
	operation.ProvisioningParameters.Parameters.Modules = getTestCase6Params()
	db.Operations().InsertOperation(operation)
	svc := NewInitKymaTemplate(db.Operations())
	ic := fixture.FixInputCreator("aws")
	ic.Config = &internal.ConfigForPlan{
		KymaTemplate: getYaml(t, withDefaultModules, "default.yaml"),
	}
	operation.InputCreator = ic

	KymaTemplateOutput := getYaml(t, withDefaultModules, "testcase1.yaml")

	// when
	op, backoff, err := svc.Run(operation, logrus.New())
	require.NoError(t, err)

	// then
	assert.Zero(t, backoff)
	assert.Equal(t, "kyma-system", op.KymaResourceNamespace)

	assert.YAMLEq(t, op.KymaTemplate, KymaTemplateOutput)
}
