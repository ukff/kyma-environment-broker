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

func OverrideMetadata(k8sObject *unstructured.Unstructured, metadataType MetadataType, values map[string]string) error {
	if k8sObject == nil {
		return fmt.Errorf("object is nil")
	}

	if metadataType != Labels && metadataType != Annotations {
		return fmt.Errorf("unknown metadata type")
	}

	switch metadataType {
	case Labels:
		{
			(*k8sObject).SetLabels(values)
		}
	case Annotations:
		{
			(*k8sObject).SetAnnotations(values)
		}
	default:
		return fmt.Errorf("unknown metadata type")
	}

	return nil
}

func ChangeSingleMetadata(k8sObject *unstructured.Unstructured, metadataType MetadataType, key, value string) error {
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
			labels[key] = value
			(*k8sObject).SetLabels(labels)
		}
	case Annotations:
		{
			annotations := (*k8sObject).GetAnnotations()
			annotations[key] = value
			(*k8sObject).SetAnnotations(annotations)
		}
	default:
		return fmt.Errorf("unknown metadata type")
	}

	return nil
}

func ChangeManyMetadata(k8sObject *unstructured.Unstructured, metadataType MetadataType, values map[string]string) error {
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
			for key, value := range values {
				labels[key] = value
			}
			(*k8sObject).SetLabels(labels)
		}
	case Annotations:
		{
			annotations := (*k8sObject).GetAnnotations()
			for key, value := range values {
				annotations[key] = value
			}
			(*k8sObject).SetAnnotations(annotations)
		}
	default:
		return fmt.Errorf("unknown metadata type")
	}

	return nil
}
