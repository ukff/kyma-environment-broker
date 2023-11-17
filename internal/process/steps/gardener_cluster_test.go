package steps

import (
	"context"
	"testing"

	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/fixture"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestGardenerCluster(t *testing.T) {
	// Given
	g := NewGardenerCluster("gc", "kcp-system")
	g.SetKubecofigSecret("kubeconfig-gc", "kcp-system")
	g.SetShootName("c-12345")

	// When
	d, _ := g.ToYaml()

	// Then
	expectedYaml := `
apiVersion: infrastructuremanager.kyma-project.io/v1
kind: GardenerCluster
metadata:
  name: gc
  namespace: kcp-system
spec:
  shoot:
    name: c-12345
  kubeconfig:
    secret:
      key: config
      name: kubeconfig-gc
      namespace: kcp-system
`
	assert.YAMLEq(t, expectedYaml, string(d))
}

func TestSyncGardenerCluster_RunWithExistingREsource(t *testing.T) {
	// given
	os := storage.NewMemoryStorage().Operations()
	existingGC := NewGardenerCluster("runtime-id-000", "kcp-system")
	existingGC.SetShootName("abcd")
	existingAsUnstructured := existingGC.ToUnstructured()
	existingAsUnstructured.SetLabels(map[string]string{"my-label": "01234"})
	k8sClient := fake.NewClientBuilder().WithRuntimeObjects(existingAsUnstructured).Build()
	operation := fixture.FixProvisioningOperation("op", "instance-id")
	operation.KymaResourceNamespace = "kcp-system"
	operation.RuntimeID = "runtime-id-000"
	operation.ShootName = "c-12345"
	svc := NewSyncGardenerCluster(os, k8sClient)

	// when
	_, backoff, err := svc.Run(operation, logrus.New())
	assert.Zero(t, backoff)
	assert.NoError(t, err)
	assertGardenerClusterSpec(t, `
apiVersion: infrastructuremanager.kyma-project.io/v1
kind: GardenerCluster
metadata:
  name: runtime-id-000
  namespace: kcp-system
spec:
  shoot:
    name: c-12345
  kubeconfig:
    secret:
      key: config
      name: kubeconfig-runtime-id-000
      namespace: kcp-system
`, k8sClient)

	// verify existing label if it still exists
	got := &unstructured.Unstructured{}
	got.SetGroupVersionKind(GardenerClusterGVK())
	err = k8sClient.Get(context.Background(), client.ObjectKeyFromObject(existingAsUnstructured), got)
	assert.NoError(t, err)
	assert.Equal(t, "01234", got.GetLabels()["my-label"])
}

func TestSyncGardenerCluster_Run(t *testing.T) {
	// given
	os := storage.NewMemoryStorage().Operations()

	k8sClient := fake.NewClientBuilder().Build()
	svc := NewSyncGardenerCluster(os, k8sClient)
	operation := fixture.FixProvisioningOperation("op", "instance-id")
	operation.KymaResourceNamespace = "kcp-system"
	operation.RuntimeID = "runtime-id-000"
	operation.ShootName = "c-12345"

	// when
	_, backoff, err := svc.Run(operation, logrus.New())

	// then
	assert.Zero(t, backoff)
	assert.NoError(t, err)
	assertGardenerClusterSpec(t, `
apiVersion: infrastructuremanager.kyma-project.io/v1
kind: GardenerCluster
metadata:
  name: runtime-id-000
  namespace: kcp-system
spec:
  shoot:
    name: c-12345
  kubeconfig:
    secret:
      key: config
      name: kubeconfig-runtime-id-000
      namespace: kcp-system
`, k8sClient)
}

func assertGardenerClusterSpec(t *testing.T, s string, k8sClient client.Client) {
	scheme := internal.NewSchemeForTests()
	decoder := serializer.NewCodecFactory(scheme).UniversalDeserializer()
	expected := unstructured.Unstructured{}
	_, _, err := decoder.Decode([]byte(s), nil, &expected)
	assert.NoError(t, err)

	existing := &unstructured.Unstructured{}
	existing.SetGroupVersionKind(GardenerClusterGVK())
	err = k8sClient.Get(context.Background(), client.ObjectKey{
		Namespace: expected.GetNamespace(),
		Name:      expected.GetName(),
	}, existing)
	assert.NoError(t, err)

	assert.Equal(t, expected.Object["spec"], existing.Object["spec"])
}
