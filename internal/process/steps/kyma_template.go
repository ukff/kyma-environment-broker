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
}

var _ process.Step = &InitKymaTemplate{}

func NewInitKymaTemplate(os storage.Operations) *InitKymaTemplate {
	return &InitKymaTemplate{operationManager: process.NewOperationManager(os)}
}

func (s *InitKymaTemplate) Name() string {
	return "Init_Kyma_Template"
}

func (s *InitKymaTemplate) Run(operation internal.Operation, logger logrus.FieldLogger) (internal.Operation, time.Duration, error) {
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

	logger.Info("applied kyma will be: %s", kymaTemplate)
	return s.operationManager.UpdateOperation(operation, func(op *internal.Operation) {
		op.KymaResourceNamespace = decodeKymaTemplate.GetNamespace()
		op.KymaTemplate = kymaTemplate
	}, logger)
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
