package main

import (
	"fmt"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSapConvergedCloudRegionMappings(t *testing.T) {
	suite := NewBrokerSuiteTestWithConvergedCloudRegionMappings(t)
	defer suite.TearDown()

	t.Run("Create catalog - test converged cloud plan not rendered if no region in path", func(t *testing.T) {
		// when
		resp := suite.CallAPI("GET", fmt.Sprintf("oauth/v2/catalog"), ``)

		content, err := io.ReadAll(resp.Body)
		assert.NoError(t, err)
		defer resp.Body.Close()

		// then
		assert.NotContains(t, string(content), "sap-converged-cloud")
		assert.NotContains(t, string(content), "non-existing-1")
		assert.NotContains(t, string(content), "non-existing-2")
		assert.NotContains(t, string(content), "non-existing-3")
		assert.NotContains(t, string(content), "non-existing-4")
	})

	t.Run("Create catalog - test converged cloud plan not render if invalid region in path", func(t *testing.T) {
		// when
		resp := suite.CallAPI("GET", fmt.Sprintf("oauth/non-existing/v2/catalog"), ``)

		content, err := io.ReadAll(resp.Body)
		assert.NoError(t, err)
		defer resp.Body.Close()

		// then
		assert.NotContains(t, string(content), "sap-converged-cloud")
		assert.NotContains(t, string(content), "non-existing-1")
		assert.NotContains(t, string(content), "non-existing-2")
		assert.NotContains(t, string(content), "non-existing-3")
		assert.NotContains(t, string(content), "non-existing-4")
	})

	t.Run("Create catalog - test converged cloud plan render if correct region in path", func(t *testing.T) {
		// when
		resp := suite.CallAPI("GET", fmt.Sprintf("oauth/cf-eu20-staging/v2/catalog"), ``)

		content, err := io.ReadAll(resp.Body)
		assert.NoError(t, err)
		defer resp.Body.Close()

		// then
		assert.Contains(t, string(content), "sap-converged-cloud")
		assert.Contains(t, string(content), "non-existing-1")
		assert.Contains(t, string(content), "non-existing-2")
		assert.NotContains(t, string(content), "non-existing-3")
		assert.NotContains(t, string(content), "non-existing-4")
	})

}
