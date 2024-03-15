package main

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/pivotal-cf/brokerapi/v8/domain"
	"github.com/stretchr/testify/assert"
)

func TestMetrics(t *testing.T) {
	t.Run("AssertCorrectMetricValue", func(t *testing.T) {
		// given
		suite := NewBrokerSuitTestWithMetrics(t)
		defer suite.TearDown()
		iid := uuid.New().String()
		plan := "361c511f-f939-4621-b228-d0fb79a1fe15"

		// when
		resp := suite.CallAPI("PUT", fmt.Sprintf("oauth/v2/service_instances/%s?accepts_incomplete=true", iid),
			`{
					"service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
					"plan_id": "361c511f-f939-4621-b228-d0fb79a1fe15",
					"context": {
						"globalaccount_id": "g-account-id",
						"subaccount_id": "sub-id",
						"user_id": "john.smith@email.com"
					},
					"parameters": {
						"name": "testing-cluster",
						"region": "eu-central-1"
					}
		}`)
		assert.Equal(t, http.StatusAccepted, resp.StatusCode)
		opID := suite.DecodeOperationID(resp)
		suite.processProvisioningByOperationID(opID)
		suite.WaitForOperationState(opID, domain.Succeeded)
		suite.AssertCorrectMetricValueT("provisioning_duration_seconds", plan, 1)

		resp = suite.CallAPI("PATCH", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true", iid), `
		{
			"service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
			"context": {
				"license_type": "CUSTOMER"
			}
		}`)

		assert.Equal(t, http.StatusAccepted, resp.StatusCode)
		opID = suite.DecodeOperationID(resp)
		suite.FinishUpdatingOperationByProvisioner(opID)
		suite.FinishUpdatingOperationByReconciler(opID)
		suite.WaitForOperationState(opID, domain.Succeeded)

		resp = suite.CallAPI("DELETE", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true&plan_id=361c511f-f939-4621-b228-d0fb79a1fe15&service_id=47c9dcbf-ff30-448e-ab36-d3bad66ba281", iid),
			``)
		assert.Equal(t, http.StatusAccepted, resp.StatusCode)
		opID = suite.DecodeOperationID(resp)
		suite.FinishDeprovisioningOperationByProvisioner(opID)
		suite.WaitForOperationsNotExists(iid) // deprovisioning completed, no operations in the DB
	})
}
