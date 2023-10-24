package steps

import (
	"fmt"
	"time"

	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/process"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type KymaAppendModules struct {
	operationManager *process.OperationManager
	logger           logrus.FieldLogger
}

func (k *KymaAppendModules) Name() string {
	return "Kyma_Append_Modules"
}

func NewKymaAppendModules(os storage.Operations) *KymaAppendModules {
	return &KymaAppendModules{operationManager: process.NewOperationManager(os)}
}

func (k *KymaAppendModules) Run(operation internal.Operation, logger logrus.FieldLogger) (internal.Operation, time.Duration, error) {
	if operation.Type != internal.OperationTypeProvision {
		return operation, 0, nil
	}

	kymaTemplate := operation.InputCreator.Configuration().KymaTemplate
	decodeKymaTemplate, err := DecodeKymaTemplate(kymaTemplate)
	if err != nil {
		return operation, 0, nil
	}
	modules := operation.ProvisioningParameters.Parameters.Modules
	switch {
	case modules == nil:
		logger.Info("module params section not set, the default kyma template will be used")
		break
	case modules.Default:
		logger.Info("default option set to true in module params section. the default one will be used")
		break
	case !modules.Default:
		{
			logger.Infof("provisioning kyma: custom module list provided, with number of items: %d", len(modules.List))
			if err := k.appendModules(decodeKymaTemplate, modules); err != nil {
				logger.Errorf("Unable to append modules to kyma template: %s", err.Error())
				return k.operationManager.OperationFailed(operation, "Unable to append modules to kyma template:", err, logger)
			}
			kymaTemplate, err = encodeKymaTemplate(decodeKymaTemplate)
			if err != nil {
				logger.Errorf("Unable to create yaml kyma template within added modules: %s", err.Error())
				return k.operationManager.OperationFailed(operation, "unable to create yaml kyma template within added modules", err, logger)
			}
			logger.Info("encoded kyma template with modules attached with success")
			break
		}
	default:
		logger.Info("not supported case in switch, the default kyma template will be used")
		break
	}
	return operation, 0, nil
}

// To consider using -> unstructured.SetNestedSlice()
func (k *KymaAppendModules) appendModules(kyma *unstructured.Unstructured, modules *internal.ModulesDTO) error {
	const (
		specKey    = "spec"
		modulesKey = "modules"
	)
	if kyma == nil {
		return fmt.Errorf("kyma unstructured object not passed to append modules")
	}
	if modules == nil {
		return fmt.Errorf("modules not passed to append modules")
	}
	content := kyma.Object
	specSection, ok := content[specKey]
	if !ok {
		return fmt.Errorf("getting spec content of kyma template")
	}
	spec, ok := specSection.(map[string]interface{})
	if !ok {
		return fmt.Errorf("converting spec of kyma template")
	}
	modulesSection, ok := spec[modulesKey]
	if !ok {
		return fmt.Errorf("getting modules content of kyma template")
	}
	var toInsert []interface{}
	if modules.List == nil || len(modules.List) == 0 {
		k.logger.Info("no modules set for kyma during provisioning")
		toInsert = make([]interface{}, 0)
	} else {
		k.logger.Info("modules are set for kyma during provisioning")
		toInsert = make([]interface{}, len(modules.List))
		for i := range modules.List {
			toInsert[i] = modules.List[i]
		}
	}

	modulesSection = toInsert
	spec[modulesKey] = modulesSection
	kyma.Object[specKey] = specSection

	k.logger.Info("modules attached to kyma successfully")
	return nil
}
