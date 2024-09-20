package k8s

import (
	"fmt"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type MetadataType int

const (
	Annotations MetadataType = 0
	Labels      MetadataType = 1
)

const (
	GlobalAccountIdLabel = "kyma-project.io/global-account-id"
)

func AddOrOverrideMetadata(k8sObject *unstructured.Unstructured, metadataType MetadataType, key, value string) error {
	if k8sObject == nil {
		return fmt.Errorf("object is nil")
	}

	if metadataType != Labels && metadataType != Annotations {
		return fmt.Errorf("unknown metadata type")
	}

	switch metadataType {
	case Labels:
		{
			labels := (*k8sObject).GetLabels()
			if labels == nil {
				labels = make(map[string]string)
			}
			labels[key] = value
			(*k8sObject).SetLabels(labels)
		}
	case Annotations:
		{
			//TBA
		}
	default:
		return fmt.Errorf("unknown metadata type")
	}

	return nil
}
