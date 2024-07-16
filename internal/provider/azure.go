package provider

import (
	"math/rand"

	"github.com/kyma-project/kyma-environment-broker/internal"
)

type (
	AzureInputProvider struct {
		MultiZone              bool
		ProvisioningParameters internal.ProvisioningParameters
	}
)

func (p *AzureInputProvider) Provide() Values {
	zonesCount := p.zonesCount()
	zones := p.zones()
	region := DefaultAzureRegion
	if p.ProvisioningParameters.Parameters.Region != nil {
		region = *p.ProvisioningParameters.Parameters.Region
	}
	return Values{
		DefaultAutoScalerMax: 20,
		DefaultAutoScalerMin: 3,
		ZonesCount:           zonesCount,
		Zones:                zones,
		ProviderType:         "azure",
		DefaultMachineType:   DefaultAzureMachineType,
		Region:               region,
		Purpose:              PurposeProduction,
	}
}

func (p *AzureInputProvider) zonesCount() int {
	zonesCount := 1
	if p.MultiZone {
		zonesCount = DefaultAzureMultiZoneCount
	}
	return zonesCount
}

func (p *AzureInputProvider) zones() []string {
	return p.generateRandomAzureZones(p.zonesCount())
}

func (p *AzureInputProvider) generateRandomAzureZones(zonesCount int) []string {
	zones := []string{"1", "2", "3"}
	if zonesCount > 3 {
		zonesCount = 3
	}

	rand.Shuffle(len(zones), func(i, j int) { zones[i], zones[j] = zones[j], zones[i] })
	return zones[:zonesCount]
}
