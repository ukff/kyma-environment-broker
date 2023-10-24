package steps

import (
	"bytes"
	"fmt"
	"time"

	"gopkg.in/yaml.v2"

	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	k8syaml "k8s.io/apimachinery/pkg/runtime/serializer/yaml"
	k8syamlutil "k8s.io/apimachinery/pkg/util/yaml"

	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/process"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
)

type InitKymaTemplate struct {
	operationManager *process.OperationManager
	logger           logrus.FieldLogger
}

var _ process.Step = &InitKymaTemplate{}

func NewInitKymaTemplate(os storage.Operations) *InitKymaTemplate {
	return &InitKymaTemplate{operationManager: process.NewOperationManager(os)}
}

func (s *InitKymaTemplate) Name() string {
	return "Init_Kyma_Template"
}

func (s *InitKymaTemplate) Run(operation internal.Operation, logger logrus.FieldLogger) (internal.Operation, time.Duration, error) {
	s.logger = logger
	logger.Infof("kyma template generation started for operation type %s", operation.Type)
	kymaTemplate := operation.InputCreator.Configuration().KymaTemplate
	decodeKymaTemplate, err := DecodeKymaTemplate(kymaTemplate)
	if err != nil {
		logger.Errorf("Unable to create kyma template: %s", err.Error())
		return s.operationManager.OperationFailed(operation, "unable to create a kyma template", err, logger)
	}
	if decodeKymaTemplate == nil {
		logger.Error("Unable to create kyma template, object is empty")
		return s.operationManager.OperationFailed(operation, "Unable to create kyma template, object is empty", err, logger)
	}
	logger.Infof("Decoded kyma template: %v", decodeKymaTemplate)

	if operation.Type == internal.OperationTypeProvision {
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
				if err := s.appendModules(decodeKymaTemplate, modules); err != nil {
					logger.Errorf("Unable to append modules to kyma template: %s", err.Error())
					return s.operationManager.OperationFailed(operation, "Unable to append modules to kyma template:", err, logger)
				}
				kymaTemplate, err = encodeKymaTemplate(decodeKymaTemplate)
				if err != nil {
					logger.Errorf("Unable to create yaml kyma template within added modules: %s", err.Error())
					return s.operationManager.OperationFailed(operation, "unable to create yaml kyma template within added modules", err, logger)
				}
				logger.Info("encoded kyma template with modules attached with success")
				break
			}
		default:
			logger.Info("not supported case in switch, the default kyma template will be used")
			break
		}
	}

	logger.Info("applied kyma will be: %s", kymaTemplate)
	return s.operationManager.UpdateOperation(operation, func(op *internal.Operation) {
		op.KymaResourceNamespace = decodeKymaTemplate.GetNamespace()
		op.KymaTemplate = kymaTemplate
	}, logger)
}

// To consider using -> unstructured.SetNestedSlice()
func (s *InitKymaTemplate) appendModules(kyma *unstructured.Unstructured, modules *internal.ModulesDTO) error {
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
		s.logger.Info("no modules set for kyma during provisioning")
		toInsert = make([]interface{}, 0)
	} else {
		s.logger.Info("modules are set for kyma during provisioning")
		toInsert = make([]interface{}, len(modules.List))
		for i := range modules.List {
			toInsert[i] = modules.List[i]
		}
	}

	modulesSection = toInsert
	spec[modulesKey] = modulesSection
	kyma.Object[specKey] = specSection

	s.logger.Info("modules attached to kyma successfully")
	return nil
}

// NOTE: adapter for upgrade_kyma which is currently not using shared staged_manager
type initKymaTemplateUpgradeKyma struct {
	*InitKymaTemplate
}

func InitKymaTemplateUpgradeKyma(os storage.Operations) initKymaTemplateUpgradeKyma {
	return initKymaTemplateUpgradeKyma{NewInitKymaTemplate(os)}
}

func (s initKymaTemplateUpgradeKyma) Run(o internal.UpgradeKymaOperation, logger logrus.FieldLogger) (internal.UpgradeKymaOperation, time.Duration, error) {
	operation, w, err := s.InitKymaTemplate.Run(o.Operation, logger)
	return internal.UpgradeKymaOperation{operation}, w, err
}

func DecodeKymaTemplate(template string) (*unstructured.Unstructured, error) {
	tmpl := []byte(template)
	decoder := k8syamlutil.NewYAMLOrJSONDecoder(bytes.NewReader(tmpl), 512)
	var rawObj runtime.RawExtension
	if err := decoder.Decode(&rawObj); err != nil {
		return nil, err
	}
	obj, _, err := k8syaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme).Decode(rawObj.Raw, nil, nil)
	if err != nil {
		return nil, err
	}
	unstructuredMap, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
	unstructuredObj := &unstructured.Unstructured{Object: unstructuredMap}
	return unstructuredObj, err
}

func encodeKymaTemplate(tmpl *unstructured.Unstructured) (string, error) {
	result, err := yaml.Marshal(tmpl.Object)
	if err != nil {
		return "", fmt.Errorf("while marshal unstructured to yaml: %v", err)
	}
	return string(result), nil
}
