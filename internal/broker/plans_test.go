package broker

import (
	"bytes"
	"encoding/json"
	"os"
	"path"
	"testing"

	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/stretchr/testify/assert"

	"github.com/stretchr/testify/require"
)

func TestSchemaGenerator(t *testing.T) {
	awsMachineNamesReduced := AwsMachinesNames()
	awsMachinesDisplayReduced := AwsMachinesDisplay()

	awsMachineNamesReduced = removeMachinesNamesFromList(awsMachineNamesReduced, "m5.large", "m6i.large")
	delete(awsMachinesDisplayReduced, "m5.large")
	delete(awsMachinesDisplayReduced, "m6i.large")

	tests := []struct {
		name                string
		generator           func(map[string]string, map[string]string, []string, bool, bool) *map[string]interface{}
		machineTypes        []string
		machineTypesDisplay map[string]string
		regionDisplay       map[string]string
		path                string
		file                string
		updateFile          string
		fileOIDC            string
		updateFileOIDC      string
	}{
		{
			name: "AWS schema is correct",
			generator: func(machinesDisplay, regionsDisplay map[string]string, machines []string, additionalParams, update bool) *map[string]interface{} {
				return AWSSchema(machinesDisplay, regionsDisplay, machines, additionalParams, update, false)
			},
			machineTypes:        AwsMachinesNames(),
			machineTypesDisplay: AwsMachinesDisplay(),
			path:                "aws",
			file:                "aws-schema.json",
			updateFile:          "update-aws-schema.json",
			fileOIDC:            "aws-schema-additional-params.json",
			updateFileOIDC:      "update-aws-schema-additional-params.json",
		},
		{
			name: "AWS reduced schema is correct",
			generator: func(machinesDisplay, regionsDisplay map[string]string, machines []string, additionalParams, update bool) *map[string]interface{} {
				return AWSSchema(machinesDisplay, regionsDisplay, machines, additionalParams, update, false)
			},
			machineTypes:        awsMachineNamesReduced,
			machineTypesDisplay: awsMachinesDisplayReduced,
			path:                "aws",
			file:                "aws-schema-reduced.json",
			updateFile:          "update-aws-schema-reduced.json",
			fileOIDC:            "aws-schema-additional-params-reduced.json",
			updateFileOIDC:      "update-aws-schema-additional-params-reduced.json",
		},
		{
			name: "AWS schema with EU access restriction is correct",
			generator: func(machinesDisplay, regionsDisplay map[string]string, machines []string, additionalParams, update bool) *map[string]interface{} {
				return AWSSchema(machinesDisplay, regionsDisplay, machines, additionalParams, update, true)
			},
			machineTypes:        AwsMachinesNames(),
			machineTypesDisplay: AwsMachinesDisplay(),
			path:                "aws",
			file:                "aws-schema-eu.json",
			updateFile:          "update-aws-schema.json",
			fileOIDC:            "aws-schema-additional-params-eu.json",
			updateFileOIDC:      "update-aws-schema-additional-params.json",
		},
		{
			name: "Azure schema is correct",
			generator: func(machinesDisplay, regionsDisplay map[string]string, machines []string, additionalParams, update bool) *map[string]interface{} {
				return AzureSchema(machinesDisplay, regionsDisplay, machines, additionalParams, update, false)
			},
			machineTypes:        AzureMachinesNames(),
			machineTypesDisplay: AzureMachinesDisplay(),
			regionDisplay:       AzureRegionsDisplay(false),
			path:                "azure",
			file:                "azure-schema.json",
			updateFile:          "update-azure-schema.json",
			fileOIDC:            "azure-schema-additional-params.json",
			updateFileOIDC:      "update-azure-schema-additional-params.json",
		},
		{
			name: "Azure schema with EU access restriction is correct",
			generator: func(machinesDisplay, regionsDisplay map[string]string, machines []string, additionalParams, update bool) *map[string]interface{} {
				return AzureSchema(machinesDisplay, regionsDisplay, machines, additionalParams, update, true)
			},
			machineTypes:        AzureMachinesNames(),
			machineTypesDisplay: AzureMachinesDisplay(),
			regionDisplay:       AzureRegionsDisplay(true),
			path:                "azure",
			file:                "azure-schema-eu.json",
			updateFile:          "update-azure-schema.json",
			fileOIDC:            "azure-schema-additional-params-eu.json",
			updateFileOIDC:      "update-azure-schema-additional-params.json",
		},
		{
			name: "AzureLite schema is correct",
			generator: func(machinesDisplay, regionsDisplay map[string]string, machines []string, additionalParams, update bool) *map[string]interface{} {
				return AzureLiteSchema(machinesDisplay, regionsDisplay, machines, additionalParams, update, false)
			},
			machineTypes:        AzureLiteMachinesNames(),
			machineTypesDisplay: AzureLiteMachinesDisplay(),
			regionDisplay:       AzureRegionsDisplay(false),
			path:                "azure",
			file:                "azure-lite-schema.json",
			updateFile:          "update-azure-lite-schema.json",
			fileOIDC:            "azure-lite-schema-additional-params.json",
			updateFileOIDC:      "update-azure-lite-schema-additional-params.json",
		},
		{
			name: "AzureLite schema with EU access restriction is correct",
			generator: func(machinesDisplay, regionsDisplay map[string]string, machines []string, additionalParams, update bool) *map[string]interface{} {
				return AzureLiteSchema(machinesDisplay, regionsDisplay, machines, additionalParams, update, true)
			},
			machineTypes:        AzureLiteMachinesNames(),
			machineTypesDisplay: AzureLiteMachinesDisplay(),
			regionDisplay:       AzureRegionsDisplay(true),
			path:                "azure",
			file:                "azure-lite-schema-eu.json",
			updateFile:          "update-azure-lite-schema.json",
			fileOIDC:            "azure-lite-schema-additional-params-eu.json",
			updateFileOIDC:      "update-azure-lite-schema-additional-params.json",
		},
		{
			name: "Freemium schema is correct",
			generator: func(machinesDisplay, regionsDisplay map[string]string, machines []string, additionalParams, update bool) *map[string]interface{} {
				return FreemiumSchema(internal.Azure, regionsDisplay, additionalParams, update, false)
			},
			machineTypes:   []string{},
			regionDisplay:  AzureRegionsDisplay(false),
			path:           "azure",
			file:           "free-azure-schema.json",
			updateFile:     "update-free-azure-schema.json",
			fileOIDC:       "free-azure-schema-additional-params.json",
			updateFileOIDC: "update-free-azure-schema-additional-params.json",
		},
		{
			name: " Freemium schema is correct",
			generator: func(machinesDisplay, regionsDisplay map[string]string, machines []string, additionalParams, update bool) *map[string]interface{} {
				return FreemiumSchema(internal.AWS, regionsDisplay, additionalParams, update, false)
			},
			machineTypes:   []string{},
			path:           "aws",
			file:           "free-aws-schema.json",
			updateFile:     "update-free-aws-schema.json",
			fileOIDC:       "free-aws-schema-additional-params.json",
			updateFileOIDC: "update-free-aws-schema-additional-params.json",
		},
		{
			name: "Freemium schema with EU access restriction is correct",
			generator: func(machinesDisplay, regionsDisplay map[string]string, machines []string, additionalParams, update bool) *map[string]interface{} {
				return FreemiumSchema(internal.Azure, regionsDisplay, additionalParams, update, true)
			},
			machineTypes:   []string{},
			regionDisplay:  AzureRegionsDisplay(true),
			path:           "azure",
			file:           "free-azure-schema-eu.json",
			updateFile:     "update-free-azure-schema.json",
			fileOIDC:       "free-azure-schema-additional-params-eu.json",
			updateFileOIDC: "update-free-azure-schema-additional-params.json",
		},
		{
			name: "Freemium schema with EU access restriction is correct",
			generator: func(machinesDisplay, regionsDisplay map[string]string, machines []string, additionalParams, update bool) *map[string]interface{} {
				return FreemiumSchema(internal.AWS, regionsDisplay, additionalParams, update, true)
			},
			machineTypes:   []string{},
			path:           "aws",
			file:           "free-aws-schema-eu.json",
			updateFile:     "update-free-aws-schema.json",
			fileOIDC:       "free-aws-schema-additional-params-eu.json",
			updateFileOIDC: "update-free-aws-schema-additional-params.json",
		},
		{
			name: "GCP schema is correct",
			generator: func(machinesDisplay, regionsDisplay map[string]string, machines []string, additionalParams, update bool) *map[string]interface{} {
				return GCPSchema(machinesDisplay, regionsDisplay, machines, additionalParams, update)
			},
			machineTypes:        GcpMachinesNames(),
			machineTypesDisplay: GcpMachinesDisplay(),
			regionDisplay:       GcpRegionsDisplay(),
			path:                "gcp",
			file:                "gcp-schema.json",
			updateFile:          "update-gcp-schema.json",
			fileOIDC:            "gcp-schema-additional-params.json",
			updateFileOIDC:      "update-gcp-schema-additional-params.json",
		},
		{
			name: "SapConvergedCloud schema is correct",
			generator: func(machinesDisplay, regionsDisplay map[string]string, machines []string, additionalParams, update bool) *map[string]interface{} {
				return SapConvergedCloudSchema(machinesDisplay, regionsDisplay, machines, additionalParams, update)
			},
			machineTypes:        SapConvergedCloudMachinesNames(),
			machineTypesDisplay: SapConvergedCloudMachinesDisplay(),
			path:                "sap-converged-cloud",
			file:                "sap-converged-cloud-schema.json",
			updateFile:          "update-sap-converged-cloud-schema.json",
			fileOIDC:            "sap-converged-cloud-schema-additional-params.json",
			updateFileOIDC:      "update-sap-converged-cloud-schema-additional-params.json",
		},
		{
			name: "Trial schema is correct",
			generator: func(machinesDisplay, regionsDisplay map[string]string, machines []string, additionalParams, update bool) *map[string]interface{} {
				return TrialSchema(additionalParams, update)
			},
			machineTypes:   []string{},
			path:           "azure",
			file:           "azure-trial-schema.json",
			updateFile:     "update-azure-trial-schema.json",
			fileOIDC:       "azure-trial-schema-additional-params.json",
			updateFileOIDC: "update-azure-trial-schema-additional-params.json",
		},
		{
			name: "Own cluster schema is correct",
			generator: func(machinesDisplay, regionsDisplay map[string]string, machines []string, additionalParams, update bool) *map[string]interface{} {
				return OwnClusterSchema(update)
			},
			machineTypes:   []string{},
			path:           ".",
			file:           "own-cluster-schema.json",
			updateFile:     "update-own-cluster-schema.json",
			fileOIDC:       "own-cluster-schema-additional-params.json",
			updateFileOIDC: "update-own-cluster-schema-additional-params.json",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.generator(tt.machineTypesDisplay, tt.regionDisplay, tt.machineTypes, false, false)
			validateSchema(t, Marshal(got), tt.path+"/"+tt.file)

			got = tt.generator(tt.machineTypesDisplay, tt.regionDisplay, tt.machineTypes, false, true)
			validateSchema(t, Marshal(got), tt.path+"/"+tt.updateFile)

			got = tt.generator(tt.machineTypesDisplay, tt.regionDisplay, tt.machineTypes, true, false)
			validateSchema(t, Marshal(got), tt.path+"/"+tt.fileOIDC)

			got = tt.generator(tt.machineTypesDisplay, tt.regionDisplay, tt.machineTypes, true, true)
			validateSchema(t, Marshal(got), tt.path+"/"+tt.updateFileOIDC)
		})
	}
}

func validateSchema(t *testing.T, got []byte, file string) {
	var prettyWant bytes.Buffer
	want := readJsonFile(t, file)
	if len(want) > 0 {
		err := json.Indent(&prettyWant, []byte(want), "", "  ")
		if err != nil {
			t.Error(err)
			t.Fail()
		}
	}

	var prettyGot bytes.Buffer
	if len(got) > 0 {
		err := json.Indent(&prettyGot, got, "", "  ")
		if err != nil {
			t.Error(err)
			t.Fail()
		}
	}
	if !assert.JSONEq(t, prettyGot.String(), prettyWant.String()) {
		t.Errorf("%v Schema() = \n######### GOT ###########%v\n######### ENDGOT ########, want \n##### WANT #####%v\n##### ENDWANT #####", file, prettyGot.String(), prettyWant.String())
	}
}

func readJsonFile(t *testing.T, file string) string {
	t.Helper()

	filename := path.Join("testdata", file)
	jsonFile, err := os.ReadFile(filename)
	require.NoError(t, err)

	return string(jsonFile)
}
