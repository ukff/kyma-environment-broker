package provisioning

import (
	"fmt"
	"time"

	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/process"
	"github.com/kyma-project/kyma-environment-broker/internal/process/steps"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type KymaAppendModules struct {
	operationManager *process.OperationManager
	logger           logrus.FieldLogger
}

var _ process.Step = &KymaAppendModules{}

func (k *KymaAppendModules) Name() string {
	return "Kyma_Append_Modules"
}

func NewKymaAppendModules(os storage.Operations) *KymaAppendModules {
	return &KymaAppendModules{operationManager: process.NewOperationManager(os)}
}

func (k *KymaAppendModules) Run(operation internal.Operation, logger logrus.FieldLogger) (internal.Operation, time.Duration, error) {
	k.logger = logger
	if operation.Type != internal.OperationTypeProvision {
		k.logger.Infof("%s is supposed to run only for Provisioning, skipping logic.", k.Name())
		return operation, 0, nil
	}

	switch modules := operation.ProvisioningParameters.Parameters.Modules; {
	case modules != nil && modules.UseDefault == nil && modules.List != nil:
		k.logger.Info("modules parameters are set, default option is set to false, custom modules will be appended")
		return k.handleCustomModules(operation, modules)
	default:
		k.logger.Info("default Kyma modules will be applied")
		k.logger.Infof("Template: %v", operation.KymaTemplate)
		return operation, 0, nil
	}
}

func (k *KymaAppendModules) handleCustomModules(operation internal.Operation, modules *internal.ModulesDTO) (internal.Operation, time.Duration, error) {
	k.logger.Infof("provisioning kyma: custom module list provided, with number of items: %d", len(modules.List))
	decodeKymaTemplate, err := steps.DecodeKymaTemplate(operation.KymaTemplate)
	if err != nil {
		errMsg := "while decoding kyma template from previous step"
		return k.operationManager.OperationFailed(operation, errMsg, fmt.Errorf("%s", errMsg), k.logger)
	}

	if err := k.appendModules(decodeKymaTemplate, modules); err != nil {
		k.logger.Errorf("Unable to append modules to kyma template: %s", err.Error())
		return k.operationManager.OperationFailed(operation, "Unable to append modules to kyma template:", err, k.logger)
	}
	updatedKymaTemplate, err := steps.EncodeKymaTemplate(decodeKymaTemplate)
	if err != nil {
		k.logger.Errorf("Unable to create yaml kyma template within added modules: %s", err.Error())
		return k.operationManager.OperationFailed(operation, "unable to create yaml kyma template within added modules", err, k.logger)
	}
	k.logger.Info("encoded kyma template with modules attached with success")
	return k.operationManager.UpdateOperation(operation, func(op *internal.Operation) {
		op.KymaResourceNamespace = decodeKymaTemplate.GetNamespace()
		op.KymaTemplate = updatedKymaTemplate
	}, k.logger)
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

	modulesSection = modules.List
	spec[modulesKey] = modulesSection
	kyma.Object[specKey] = specSection

	k.logger.Info("modules attached to kyma successfully")
	return nil
}
