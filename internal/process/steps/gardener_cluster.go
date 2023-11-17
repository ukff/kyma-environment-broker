package steps

import (
	"context"
	"fmt"
	"time"

	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/process"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func NewSyncGardenerCluster(os storage.Operations, k8sClient client.Client) syncGardenerCluster {
	return syncGardenerCluster{
		k8sClient:        k8sClient,
		operationManager: process.NewOperationManager(os),
	}
}

type syncGardenerCluster struct {
	k8sClient        client.Client
	operationManager *process.OperationManager
}

func (_ *syncGardenerCluster) Name() string {
	return "Sync_GardenerCluster"
}

func (s *syncGardenerCluster) Run(operation internal.Operation, log logrus.FieldLogger) (internal.Operation, time.Duration, error) {
	gardenerCluster, err := s.GetOrCreateNewGardenerCluster(operation.RuntimeID, operation.KymaResourceNamespace)
	if err != nil {
		log.Errorf("unable to get GardenerCluster %s/%s", operation.KymaResourceNamespace, operation.RuntimeID)
		return s.operationManager.RetryOperation(operation, "unable to get GardenerCluster", err, 3*time.Second, 20*time.Second, log)
	}
	gardenerCluster.SetShootName(operation.ShootName)
	gardenerCluster.SetKubecofigSecret(fmt.Sprintf("kubeconfig-%s", operation.RuntimeID), operation.KymaResourceNamespace)

	obj := gardenerCluster.ToUnstructured()
	ApplyLabelsAndAnnotationsForLM(obj, operation)

	if gardenerCluster.ExistsInTheCluster() {
		err := s.k8sClient.Update(context.Background(), obj)
		if err != nil {
			log.Errorf("unable to update GardenerCluster %s/%s: %s", operation.KymaResourceNamespace, operation.RuntimeID, err.Error())
			return s.operationManager.RetryOperation(operation, "unable to update GardenerCluster", err, 3*time.Second, 20*time.Second, log)
		}
	} else {
		err := s.k8sClient.Create(context.Background(), obj)
		if err != nil {
			log.Errorf("unable to create GardenerCluster %s/%s: ", operation.KymaResourceNamespace, operation.RuntimeID, err.Error())
			return s.operationManager.RetryOperation(operation, "unable to create GardenerCluster", err, 3*time.Second, 20*time.Second, log)
		}
	}

	return operation, 0, nil
}

func (s *syncGardenerCluster) GetOrCreateNewGardenerCluster(name, namespace string) (*GardenerCluster, error) {
	gardenerCluster := NewGardenerCluster(name, namespace)
	existing := &unstructured.Unstructured{}
	existing.SetGroupVersionKind(GardenerClusterGVK())
	err := s.k8sClient.Get(context.Background(), gardenerCluster.ObjectKey(), existing)
	switch {
	case errors.IsNotFound(err):
		return gardenerCluster, nil
	case err != nil:
		return nil, err
	}
	return NewGardenerClusterFromUnstructured(existing), nil
}

func NewGardenerClusterFromUnstructured(u *unstructured.Unstructured) *GardenerCluster {
	return &GardenerCluster{obj: u}
}

func GardenerClusterGVK() schema.GroupVersionKind {
	return schema.GroupVersionKind{
		Group:   "infrastructuremanager.kyma-project.io",
		Version: "v1",
		Kind:    "GardenerCluster",
	}
}

type GardenerCluster struct {
	obj *unstructured.Unstructured
}

func NewGardenerCluster(name, namespace string) *GardenerCluster {
	gardenerCluster := &unstructured.Unstructured{}
	gardenerCluster.SetGroupVersionKind(GardenerClusterGVK())
	gardenerCluster.SetName(name)
	gardenerCluster.SetNamespace(namespace)

	gardenerCluster.Object["spec"] = map[string]interface{}{
		"kubeconfig": map[string]interface{}{},
		"shoot":      map[string]interface{}{},
	}
	return &GardenerCluster{obj: gardenerCluster}
}

func (c *GardenerCluster) ObjectKey() client.ObjectKey {
	return client.ObjectKeyFromObject(c.obj)
}

func (c *GardenerCluster) SetShootName(shootName string) {
	c.obj.Object["spec"].(map[string]interface{})["shoot"] = map[string]interface{}{
		"name": shootName,
	}
}

func (c *GardenerCluster) SetKubecofigSecret(name, namespace string) {
	c.obj.Object["spec"].(map[string]interface{})["kubeconfig"] = map[string]interface{}{
		"secret": map[string]interface{}{
			"name":      name,
			"namespace": namespace,
			"key":       "config",
		},
	}
}

func (c *GardenerCluster) ToUnstructured() *unstructured.Unstructured {
	return c.obj
}

func (c *GardenerCluster) ToYaml() ([]byte, error) {
	return yaml.Marshal(c.obj.Object)
}

func (c *GardenerCluster) ExistsInTheCluster() bool {
	return c.obj.GetResourceVersion() != ""
}
