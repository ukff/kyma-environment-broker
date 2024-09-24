package k8s

import (
	"fmt"
	"strings"

	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const (
	KymaCr            = "kyma"
	GardenerClusterCr = "gardenercluster"
	RuntimeCr         = "runtime"
)

func GvkByName(name string) (schema.GroupVersionKind, error) {
	if name == "" {
		return schema.GroupVersionKind{}, fmt.Errorf("name is empty")
	}

	name = strings.ToLower(name)
	switch name {
	case KymaCr:
		{
			return schema.GroupVersionKind{
				Group:   "operator.kyma-project.io",
				Version: "v1beta2",
				Kind:    "Kyma",
			}, nil
		}
	case GardenerClusterCr:
		{
			return schema.GroupVersionKind{
				Group:   "infrastructuremanager.kyma-project.io",
				Version: "v1",
				Kind:    "GardenerCluster",
			}, nil
		}
	case RuntimeCr:
		{
			var runtime imv1.Runtime
			return runtime.GroupVersionKind(), nil
		}
	}

	return schema.GroupVersionKind{}, fmt.Errorf("unknown name: %s", name)
}
