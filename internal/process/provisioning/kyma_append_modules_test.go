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
	kymaAppendModulesTestAssets     = "kyma_append_modules_tests_assets"
)

func execTest(t *testing.T, parameters *internal.ModulesDTO, in, out string, expectedErr bool) {
	db := storage.NewMemoryStorage()
	operation := fixture.FixOperation(uuid.NewString(), uuid.NewString(), internal.OperationTypeProvision)
	operation.KymaTemplate = internal.GetFile(t, fmt.Sprintf("%s/%s", kymaAppendModulesTestAssets, in))
	expectedKymaTemplate := internal.GetFile(t, fmt.Sprintf("%s/%s", kymaAppendModulesTestAssets, out))
	operation.ProvisioningParameters.Parameters.Modules = parameters
	db.Operations().InsertOperation(operation)
	svc := NewKymaAppendModules(db.Operations())

	// when
	op, backoff, err := svc.Run(operation, logrus.New())
	if expectedErr {
		assert.Equal(t, kymaExpectedNamespace, op.KymaResourceNamespace)
		assert.YAMLEq(t, op.KymaTemplate, op.KymaTemplate)
		require.Error(t, err)
	} else {
		// then
		assert.Zero(t, backoff)
		assert.Equal(t, kymaExpectedNamespace, op.KymaResourceNamespace)
		assert.YAMLEq(t, expectedKymaTemplate, op.KymaTemplate)
		require.NoError(t, err)
	}
}

// when given kyma template without any default modules set

func TestKymaAppendModulesWithEmptyDefaultOnes1_Run(t *testing.T) {
	params := func() *internal.ModulesDTO {
		modules := &internal.ModulesDTO{}
		modules.Default = ptr.Bool(false)
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
		return modules
	}
	execTest(t, params(), givenKymaTemplateWithoutModules, "kyma_template_output_1.yaml", false)
}

func TestKymaAppendModulesWithEmptyDefaultOnes1x_Run(t *testing.T) {
	params := func() *internal.ModulesDTO {
		modules := &internal.ModulesDTO{}
		modules.Default = ptr.Bool(false)
		modules.List = make([]*internal.ModuleDTO, 0)
		m1 := internal.ModuleDTO{
			Name:                 technicalNameBtpManager,
			CustomResourcePolicy: internal.CreateAndDelete,
		}
		modules.List = append(modules.List, &m1)
		return modules
	}
	execTest(t, params(), givenKymaTemplateWithoutModules, "kyma_template_output_3.yaml", false)
}

func TestKymaAppendModulesWithEmptyDefaultOnes111_Run(t *testing.T) {
	params := func() *internal.ModulesDTO {
		modules := &internal.ModulesDTO{}
		modules.Default = ptr.Bool(true)
		return modules
	}
	execTest(t, params(), givenKymaTemplateWithoutModules, "kyma_template_output_0.yaml", false)
}

func TestKymaAppendModulesWithEmptyDefaultOnes11_Run(t *testing.T) {
	params := func() *internal.ModulesDTO {
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
		return modules
	}
	execTest(t, params(), givenKymaTemplateWithoutModules, givenKymaTemplateWithoutModules, true)
}

func TestKymaAppendModulesWithEmptyDefaultOnes2_Run(t *testing.T) {
	params := func() *internal.ModulesDTO {
		modules := &internal.ModulesDTO{}
		modules.Default = ptr.Bool(false)
		modules.List = make([]*internal.ModuleDTO, 0)
		m1 := internal.ModuleDTO{
			Name:                 technicalNameBtpManager,
			Channel:              internal.Fast,
			CustomResourcePolicy: internal.CreateAndDelete,
		}
		modules.List = append(modules.List, &m1)
		return modules
	}
	execTest(t, params(), givenKymaTemplateWithoutModules, "kyma_template_output_2.yaml", false)
}

func TestKymaAppendModulesWithEmptyDefaultOnes2_Run_A(t *testing.T) {
	params := func() *internal.ModulesDTO {
		modules := &internal.ModulesDTO{}
		modules.Default = ptr.Bool(false)
		modules.List = make([]*internal.ModuleDTO, 0)
		m1 := internal.ModuleDTO{
			Name:                 technicalNameBtpManager,
			Channel:              internal.Fast,
			CustomResourcePolicy: internal.CreateAndDelete,
		}
		modules.List = append(modules.List, &m1)
		return modules
	}
	execTest(t, params(), givenKymaTemplateWithoutModules, "kyma_template_output_2.yaml", false)
}

func TestKymaAppendModulesWithEmptyDefaultOnes3_Run(t *testing.T) {
	params := func() *internal.ModulesDTO {
		return &internal.ModulesDTO{}
	}
	execTest(t, params(), givenKymaTemplateWithoutModules, "kyma_template_output_0.yaml", false)
}

func TestKymaAppendModulesWithEmptyDefaultOnes4_Run(t *testing.T) {
	execTest(t, nil, givenKymaTemplateWithoutModules, "kyma_template_output_0.yaml", false)
}

func TestKymaAppendModulesWithEmptyDefaultOnes5_Run(t *testing.T) {
	params := func() *internal.ModulesDTO {
		modules := &internal.ModulesDTO{}
		modules.List = make([]*internal.ModuleDTO, 0)
		return modules
	}
	execTest(t, params(), givenKymaTemplateWithoutModules, "kyma_template_output_0.yaml", false)
}

func TestKymaAppendModulesWithEmptyDefaultOnes6_Run(t *testing.T) {
	params := func() *internal.ModulesDTO {
		modules := &internal.ModulesDTO{}
		modules.List = nil
		return modules
	}
	execTest(t, params(), givenKymaTemplateWithoutModules, "kyma_template_output_0.yaml", false)
}

func TestKymaAppendModulesWithEmptyDefaultOnes7_Run(t *testing.T) {
	params := func() *internal.ModulesDTO {
		return nil
	}
	execTest(t, params(), givenKymaTemplateWithoutModules, "kyma_template_output_0.yaml", false)
}

// when given kyma template have default modules set

func TestKymaAppendModulesWithDefaultOnesSet1_Run(t *testing.T) {
	params := func() *internal.ModulesDTO {
		return nil
	}
	execTest(t, params(), givenKymaTemplateWithModules, givenKymaTemplateWithModules, false)
}

func TestKymaAppendModulesWithDefaultOnesSet2_Run(t *testing.T) {
	params := func() *internal.ModulesDTO {
		modules := &internal.ModulesDTO{}
		modules.Default = ptr.Bool(false)
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
		return modules
	}
	execTest(t, params(), givenKymaTemplateWithModules, "kyma_template_output_4.yaml", false)
}

func TestKymaAppendModulesWithDefaultOnesSet3_Run(t *testing.T) {
	params := func() *internal.ModulesDTO {
		modules := &internal.ModulesDTO{}
		modules.Default = ptr.Bool(true)
		modules.List = make([]*internal.ModuleDTO, 0)
		m1 := internal.ModuleDTO{
			Name:                 technicalNameBtpManager,
			Channel:              internal.Regular,
			CustomResourcePolicy: internal.CreateAndDelete,
		}
		modules.List = append(modules.List, &m1)
		return modules
	}
	execTest(t, params(), givenKymaTemplateWithModules, givenKymaTemplateWithModules, true)
}

func TestKymaAppendModulesWithDefaultOnesSet4_Run(t *testing.T) {
	params := func() *internal.ModulesDTO {
		modules := &internal.ModulesDTO{}
		return modules
	}
	execTest(t, params(), givenKymaTemplateWithModules, givenKymaTemplateWithModules, false)
}

func TestKymaAppendModulesWithDefaultOnesSet5_Run(t *testing.T) {
	params := func() *internal.ModulesDTO {
		return nil
	}
	execTest(t, params(), givenKymaTemplateWithModules, givenKymaTemplateWithModules, false)
}

func TestKymaAppendModulesWithDefaultOnesSet6_Run(t *testing.T) {
	params := func() *internal.ModulesDTO {
		return nil
	}
	execTest(t, params(), givenKymaTemplateWithModules, givenKymaTemplateWithModules, false)
}
