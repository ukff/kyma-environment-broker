package kubeconfig

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/api/errors"

	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const kcpNamespace = "kcp-system"

type SecretProvider struct {
	kcpK8sClient client.Client
}

func NewK8sClientFromSecretProvider(kcpK8sClient client.Client) *SecretProvider {
	return &SecretProvider{
		kcpK8sClient: kcpK8sClient,
	}
}

func (p *SecretProvider) KubeconfigForRuntimeID(runtimeId string) ([]byte, error) {
	kubeConfigSecret := &v1.Secret{}
	err := p.kcpK8sClient.Get(context.Background(), p.objectKey(runtimeId), kubeConfigSecret)
	if errors.IsNotFound(err) {
		return nil, NewNotFoundError("secret not found")
	}
	if err != nil {
		return nil, fmt.Errorf("while getting secret from kcp for runtimeId=%s", runtimeId)
	}
	config, ok := kubeConfigSecret.Data["config"]
	if !ok {
		return nil, fmt.Errorf("while getting 'config' from secret from %s", p.objectKey(runtimeId))
	}
	if len(config) == 0 {
		return nil, fmt.Errorf("empty kubeconfig")
	}
	return config, nil
}

func (p *SecretProvider) objectKey(runtimeId string) client.ObjectKey {
	return client.ObjectKey{
		Namespace: kcpNamespace,
		Name:      fmt.Sprintf("kubeconfig-%s", runtimeId),
	}
}

func (p *SecretProvider) K8sClientForRuntimeID(runtimeID string) (client.Client, error) {
	kubeconfig, err := p.KubeconfigForRuntimeID(runtimeID)
	if err != nil {
		return nil, err
	}
	restCfg, err := clientcmd.RESTConfigFromKubeConfig(kubeconfig)
	if err != nil {
		return nil, err
	}

	k8sCli, err := client.New(restCfg, client.Options{
		Scheme: scheme.Scheme,
	})
	return k8sCli, err
}

type FakeProvider struct {
	c client.Client
}

func NewFakeK8sClientProvider(c client.Client) *FakeProvider {
	return &FakeProvider{c: c}
}

func (p *FakeProvider) K8sClientForRuntimeID(_ string) (client.Client, error) {
	if p.c == nil {
		return nil, fmt.Errorf("unable to get client")
	}
	return p.c, nil
}

func (p *FakeProvider) KubeconfigForRuntimeID(runtimeId string) ([]byte, error) {
	return []byte("fake kubeconfig"), nil
}
