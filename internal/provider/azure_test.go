package provider

import (
	"testing"

	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/ptr"
)

func TestAzureDefaults(t *testing.T) {

	// given
	aws := AzureInputProvider{
		MultiZone: true,
		ProvisioningParameters: internal.ProvisioningParameters{
			Parameters:     internal.ProvisioningParametersDTO{Region: ptr.String("eastus")},
			PlatformRegion: "cf-eu11",
		},
	}

	// when
	values := aws.Provide()

	// then

	assertValues(t, Values{
		DefaultAutoScalerMax: 20,
		DefaultAutoScalerMin: 3,
		ZonesCount:           3,
		Zones:                []string{"1", "2", "3"},
		ProviderType:         "azure",
		DefaultMachineType:   "Standard_D2s_v5",
		Region:               "eastus",
		Purpose:              "production",
	}, values)
}

func TestAzureSpecific(t *testing.T) {

	// given
	azure := AzureInputProvider{
		MultiZone: true,
		ProvisioningParameters: internal.ProvisioningParameters{
			Parameters: internal.ProvisioningParametersDTO{
				MachineType: ptr.String("Standard_D48_v3"),
				Region:      ptr.String("uksouth"),
			},
			PlatformRegion:   "cf-eu11",
			PlatformProvider: "azure",
		},
	}

	// when
	values := azure.Provide()

	// then

	assertValues(t, Values{
		// default values does not depend on provisioning parameters
		DefaultAutoScalerMax: 20,
		DefaultAutoScalerMin: 3,
		ZonesCount:           3,
		Zones:                []string{"1", "2", "3"},
		ProviderType:         "azure",
		DefaultMachineType:   "Standard_D2s_v5",
		Region:               "uksouth",
		Purpose:              "production",
	}, values)
}
