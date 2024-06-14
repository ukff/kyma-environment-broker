package broker

import (
	"fmt"

	"github.com/kyma-project/kyma-environment-broker/internal/utils"
)

//go:generate mockery --name=RegionReader --output=automock --outpkg=automock --case=underscore
type RegionReader interface {
	Read(filename string) (map[string][]string, error)
}

type ConvergedCloudRegionProvider interface {
	GetRegions(string) []string
}

type DefaultConvergedCloudRegionsProvider struct {
	// placeholder
	regionConfiguration map[string][]string
}

func NewDefaultConvergedCloudRegionsProvider(regionConfigurationPath string, reader RegionReader) (ConvergedCloudRegionProvider, error) {
	if regionConfigurationPath == "" {
		return nil, fmt.Errorf("regionConfigurationPath cannot be empty")
	}

	if reader == nil {
		return nil, fmt.Errorf("reader cannot be nil")
	}

	regionConfiguration, err := reader.Read(regionConfigurationPath)
	if err != nil {
		return nil, fmt.Errorf("while unmarshalling a file with sap-converged-cloud region mappings: %w", err)
	}

	return &DefaultConvergedCloudRegionsProvider{
		regionConfiguration: regionConfiguration,
	}, nil
}

func (c *DefaultConvergedCloudRegionsProvider) GetRegions(mappedRegion string) []string {
	item, found := c.regionConfiguration[mappedRegion]

	if !found {
		return []string{}
	}

	return item
}

func NewOneForAllConvergedCloudRegionsProvider() ConvergedCloudRegionProvider {
	return &OneForAllConvergedCloudRegionsProvider{}
}

type OneForAllConvergedCloudRegionsProvider struct {
}

func (c *OneForAllConvergedCloudRegionsProvider) GetRegions(mappedRegion string) []string {
	return []string{"eu-de-1"}
}

type YamlRegionReader struct{}

func (u *YamlRegionReader) Read(filename string) (map[string][]string, error) {
	regionMappings := make(map[string][]string)
	err := utils.UnmarshalYamlFile(filename, &regionMappings)
	if err != nil {
		return nil, fmt.Errorf("while unmarshalling a file with region mappings: %w", err)
	}
	return regionMappings, nil
}
