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
	case "kyma":
		{
			return schema.GroupVersionKind{
				Group:   "kyma-project.io",
				Version: "v1",
				Kind:    "Kyma",
			}, nil
		}
	case "gardenercluster":
		{
			return schema.GroupVersionKind{
				Group:   "infrastructuremanager.kyma-project.io",
				Version: "v1",
				Kind:    "GardenerCluster",
			}, nil
		}
	case "runtime":
		{
			var runtime imv1.Runtime
			return runtime.GroupVersionKind(), nil
		}
	}
	return schema.GroupVersionKind{}, fmt.Errorf("unknown name: %s", name)
}
