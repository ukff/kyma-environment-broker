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

func OverrideLabelsOrAnnotations(k8sObject *unstructured.Unstructured, metadataType MetadataType, values map[string]string) error {
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

func ChangeOneLabelOrAnnotation(k8sObject *unstructured.Unstructured, metadataType MetadataType, key, value string) error {
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
			labels := (*k8sObject).GetLabels()
			labels[key] = value
			(*k8sObject).SetAnnotations(map[string]string{key: value})
		}
	default:
		return fmt.Errorf("unknown metadata type")
	}

	return nil
}

func ChangeManyLabelsOrAnnotations(k8sObject *unstructured.Unstructured, metadataType MetadataType, values map[string]string) error {
	if k8sObject == nil {
		return fmt.Errorf("object is nil")
	}

	if metadataType != Labels && metadataType != Annotations {
		return fmt.Errorf("unknown metadata type")
	}

	contains := func(slice []string, value string) bool {
		for _, item := range slice {
			if item == value {
				return true
			}
		}
		return false
	}

	switch metadataType {
	case Labels:
		{
			labels := (*k8sObject).GetLabels()
			for key, value := range values {
				if contains(labels, key) {
					labels[key] = value
				}
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
