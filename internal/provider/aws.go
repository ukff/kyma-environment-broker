package provider

import (
	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/euaccess"
)

type (
	AWSInputProvider struct {
		MultiZone              bool
		ProvisioningParameters internal.ProvisioningParameters
	}

	AWSTrialInputProvider struct {
		PlatformRegionMapping  map[string]string
		UseSmallerMachineTypes bool
		ProvisioningParameters internal.ProvisioningParameters
	}
)

func (p *AWSInputProvider) Provide() Values {
	zonesCount := p.zonesCount()
	zones := p.zones()
	region := DefaultAWSRegion
	if p.ProvisioningParameters.Parameters.Region != nil {
		region = *p.ProvisioningParameters.Parameters.Region
	}
	return Values{
		DefaultAutoScalerMax: 20,
		DefaultAutoScalerMin: 3,
		ZonesCount:           zonesCount,
		Zones:                zones,
		ProviderType:         "aws",
		DefaultMachineType:   DefaultAWSMachineType,
		Region:               region,
		Purpose:              PurposeProduction, //TODO - default value is define per landscape in mgmt plane config
	}
}

func (p *AWSInputProvider) zonesCount() int { //TODO - this is called twice per provisioning
	zonesCount := 1
	if p.MultiZone {
		zonesCount = DefaultAWSMultiZoneCount
	}
	return zonesCount
}

func (p *AWSInputProvider) zones() []string {
	region := DefaultAWSRegion
	if p.ProvisioningParameters.Parameters.Region != nil {
		region = *p.ProvisioningParameters.Parameters.Region
	}
	return MultipleZonesForAWSRegion(region, p.zonesCount())
}

func (p *AWSTrialInputProvider) Provide() Values {
	machineType := DefaultOldAWSTrialMachineType
	if p.UseSmallerMachineTypes {
		machineType = DefaultAWSMachineType
	}
	region := p.region()

	return Values{
		DefaultAutoScalerMax: 1,
		DefaultAutoScalerMin: 1,
		ZonesCount:           1,
		Zones:                MultipleZonesForAWSRegion(region, 1),
		ProviderType:         "aws",
		DefaultMachineType:   machineType,
		Region:               region,
		Purpose:              PurposeEvaluation,
	}
}

func (p *AWSTrialInputProvider) region() string {
	if euaccess.IsEURestrictedAccess(p.ProvisioningParameters.PlatformRegion) {
		return DefaultEuAccessAWSRegion
	}
	if p.ProvisioningParameters.PlatformRegion != "" {
		abstractRegion, found := p.PlatformRegionMapping[p.ProvisioningParameters.PlatformRegion]
		if found {
			return *toAWSSpecific[abstractRegion]
		}
	}
	if p.ProvisioningParameters.Parameters.Region != nil && *p.ProvisioningParameters.Parameters.Region != "" {
		return *toAWSSpecific[*p.ProvisioningParameters.Parameters.Region]
	}
	return DefaultAWSTrialRegion
}
