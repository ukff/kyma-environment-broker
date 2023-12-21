package kubeconfig

import (
	"context"
	"os"
	"os/exec"
	"strings"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"testing"

	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/envtest"

	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

const envTestAssets = "KUBEBUILDER_ASSETS"

func TestSecretProvider_NoValueInSecret(t *testing.T) {
	// given
	kcpClient := fake.NewClientBuilder().Build()
	kcpClient.Create(context.Background(), &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kubeconfig-runtime00",
			Namespace: "kcp-system",
		},
	})
	provider := SecretProvider{
		kcpK8sClient: kcpClient,
	}

	// when
	_, errKubeconfig := provider.KubeconfigForRuntimeID("runtime00")
	_, errClient := provider.K8sClientForRuntimeID("runtime00")

	// then
	assert.Error(t, errKubeconfig)
	assert.Error(t, errClient)
}

func TestSecretProvider_NoSecret(t *testing.T) {
	// given
	kcpClient := fake.NewClientBuilder().Build()
	provider := SecretProvider{
		kcpK8sClient: kcpClient,
	}

	// when
	_, errKubeconfig := provider.KubeconfigForRuntimeID("runtime00")
	_, errClient := provider.K8sClientForRuntimeID("runtime00")

	// then
	assert.Error(t, errKubeconfig)
	assert.Error(t, errClient)
}

func TestSecretProvider_BadKubeconfig(t *testing.T) {
	// given
	kcpClient := fake.NewClientBuilder().Build()
	kcpClient.Create(context.Background(), &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kubeconfig-runtime00",
			Namespace: "kcp-system",
		},
		Data: map[string][]byte{
			"config": []byte("bad-kubeconfig"),
		},
	})
	provider := SecretProvider{
		kcpK8sClient: kcpClient,
	}

	// when
	_, errClient := provider.K8sClientForRuntimeID("runtime00")

	// then
	assert.Error(t, errClient)
}

func TestSecretProvider_KubernetesAndK8sClientForRuntimeID(t *testing.T) {
	// Given

	// prepare envtest to provide valid kubeconfig
	if os.Getenv(envTestAssets) == "" {
		out, err := exec.Command("/bin/sh", "../../setup-envtest.sh").Output()
		require.NoError(t, err)
		path := strings.Replace(string(out), "\n", "", -1)
		os.Setenv(envTestAssets, path)
	}

	env := envtest.Environment{}
	config, err := env.Start()
	assert.NoError(t, err)
	defer env.Stop()
	kubeconfig := createKubeconfigFileForRestConfig(*config)

	// prepare a k8s client to store a secret with kubeconfig
	kcpClient := fake.NewClientBuilder().Build()
	kcpClient.Create(context.Background(), &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kubeconfig-runtime00",
			Namespace: "kcp-system",
		},
		Data: map[string][]byte{
			"config": kubeconfig,
		},
	})
	provider := SecretProvider{
		kcpK8sClient: kcpClient,
	}

	// when
	kubeconfig, errKubeconfig := provider.KubeconfigForRuntimeID("runtime00")
	k8sClient, errClient := provider.K8sClientForRuntimeID("runtime00")

	// then
	assert.NotEmpty(t, kubeconfig)
	assert.NotNil(t, k8sClient)
	assert.NoError(t, errKubeconfig)
	assert.NoError(t, errClient)
}

func createKubeconfigFileForRestConfig(restConfig rest.Config) []byte {
	const (
		userName    = "user"
		clusterName = "cluster"
		contextName = "context"
	)

	clusters := make(map[string]*clientcmdapi.Cluster)
	clusters[clusterName] = &clientcmdapi.Cluster{
		Server:                   restConfig.Host,
		CertificateAuthorityData: restConfig.CAData,
	}
	contexts := make(map[string]*clientcmdapi.Context)
	contexts[contextName] = &clientcmdapi.Context{
		Cluster:  clusterName,
		AuthInfo: userName,
	}
	authinfos := make(map[string]*clientcmdapi.AuthInfo)
	authinfos[userName] = &clientcmdapi.AuthInfo{
		ClientCertificateData: restConfig.CertData,
		ClientKeyData:         restConfig.KeyData,
	}
	clientConfig := clientcmdapi.Config{
		Kind:           "Config",
		APIVersion:     "v1",
		Clusters:       clusters,
		Contexts:       contexts,
		CurrentContext: contextName,
		AuthInfos:      authinfos,
	}
	kubeconfig, _ := clientcmd.Write(clientConfig)
	return kubeconfig
}
