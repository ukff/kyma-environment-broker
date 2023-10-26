package provisioning

import (
	"fmt"
	"testing"
	
	"github.com/google/uuid"
	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/fixture"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	defaultKymaTemplate                         = "default.yaml"
	expectedNamespace                           = "kyma-system"
	kymaTemplateTestAssets                      = "kyma_append_modules_tests_assets"
	withDefaultModules                          = "with_default_modules"
	withoutDefaultModules                       = "without_default_modules"
	kymaTemplateTestAssetsWithDefaultModules    = fmt.Sprintf("%s/%s", kymaTemplateTestAssets, withDefaultModules)
	kymaTemplateTestAssetsWithoutDefaultModules = fmt.Sprintf("%s/%s", kymaTemplateTestAssets, withoutDefaultModules)
)

func execTest(t *testing.T, m *internal.ModulesDTO, basePath, expected string) {
	db := storage.NewMemoryStorage()
	operation := fixture.FixOperation(uuid.NewString(), uuid.NewString(), internal.OperationTypeProvision)
	operation.KymaTemplate = internal.GetFile(t, fmt.Sprintf("%s/%s", basePath, defaultKymaTemplate))
	expectedKymaTemplate := internal.GetFile(t, fmt.Sprintf("%s/%s", basePath, expected))
	operation.ProvisioningParameters.Parameters.Modules = m
	db.Operations().InsertOperation(operation)
	svc := NewKymaAppendModules(db.Operations())
	
	// when
	op, backoff, err := svc.Run(operation, logrus.New())
	require.NoError(t, err)
	
	// then
	assert.Zero(t, backoff)
	assert.Equal(t, expectedNamespace, op.KymaResourceNamespace)
	assert.YAMLEq(t, op.KymaTemplate, expectedKymaTemplate)
}

func TestKymaAppendModulesWithEmptyDefaultOnes1_Run(t *testing.T) {
	params := func() *internal.ModulesDTO {
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
	execTest(t, params(), kymaTemplateTestAssetsWithoutDefaultModules, "testcase1.yaml")
}

func TestKymaAppendModulesWithEmptyDefaultOnes2_Run(t *testing.T) {
	params := func() *internal.ModulesDTO {
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
	execTest(t, params(), kymaTemplateTestAssetsWithoutDefaultModules, "testcase2.yaml")
}

func TestKymaAppendModulesWithEmptyDefaultOnes3_Run(t *testing.T) {
	params := func() *internal.ModulesDTO {
		return &internal.ModulesDTO{}
	}
	execTest(t, params(), kymaTemplateTestAssetsWithoutDefaultModules, "testcase3.yaml")
}

func TestKymaAppendModulesWithEmptyDefaultOnes4_Run(t *testing.T) {
	execTest(t, nil, kymaTemplateTestAssetsWithoutDefaultModules, "testcase4.yaml")
}

func TestKymaAppendModulesWithEmptyDefaultOnes5_Run(t *testing.T) {
	params := func() *internal.ModulesDTO {
		modules := &internal.ModulesDTO{}
		modules.List = make([]*internal.ModuleDTO, 0)
		return modules
	}
	execTest(t, params(), kymaTemplateTestAssetsWithoutDefaultModules, "testcase5.yaml")
}

func TestKymaAppendModulesWithEmptyDefaultOnes6_Run(t *testing.T) {
	params := func() *internal.ModulesDTO {
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
	
	execTest(t, params(), kymaTemplateTestAssetsWithoutDefaultModules, "testcase6.yaml")
}
