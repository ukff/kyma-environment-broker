package main

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/google/uuid"
	reconcilerApi `github.com/kyma-incubator/reconciler/pkg/keb`
	`github.com/kyma-project/kyma-environment-broker/internal`
	"github.com/pivotal-cf/brokerapi/v8/domain"
	"github.com/stretchr/testify/assert"
)

// - kcp_keb_v2_operations_provisioning_failed_total
// - kcp_keb_v2_operations_provisioning_in_progress_total
// - kcp_keb_v2_operations_provisioning_succeeded_total
// - kcp_keb_v2_operations_deprovisioning_failed_total
// - kcp_keb_v2_operations_deprovisioning_in_progress_total
// - kcp_keb_v2_operations_deprovisioning_succeeded_total
// - kcp_keb_v2_operations_updating_update_failed_total
// - kcp_keb_v2_operations_updating_update_in_progress_total
// - kcp_keb_v2_operations_updating_update_succeeded_total

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
		suite.AssertCorrectMetricValueT2(internal.OperationTypeProvision, domain.Succeeded, plan, 1)

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
		suite.AssertCorrectMetricValueT2(internal.OperationTypeUpdate, domain.Succeeded, plan, 1)
		
		suite.SetReconcilerResponseStatus(reconcilerApi.StatusDeleted)
		resp = suite.CallAPI("DELETE", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true&plan_id=361c511f-f939-4621-b228-d0fb79a1fe15&service_id=47c9dcbf-ff30-448e-ab36-d3bad66ba281", iid), ``)
		assert.Equal(t, http.StatusAccepted, resp.StatusCode)
		opID = suite.DecodeOperationID(resp)
		suite.FinishDeprovisioningByReconciler(opID)
		suite.FinishDeprovisioningOperationByProvisioner(opID)
		suite.WaitForInstanceArchivedCreated(iid)
		suite.WaitFor(func() bool {
			resp := suite.CallAPI("GET", fmt.Sprintf("oauth/v2/service_instances/%s/last_operation", iid), ``)
			defer resp.Body.Close()
			data := suite.ParseLastOperationResponse(resp)
			return resp.StatusCode == http.StatusOK && data.State == domain.Succeeded
		})
		suite.WaitForOperationsNotExists(iid) // deprovisioning completed, no operations in the DB
		suite.AssertCorrectMetricValueT2(internal.OperationTypeDeprovision, domain.Succeeded, plan, 1)
		
		suite.AssertCorrectMetricValueT2(internal.OperationTypeProvision, domain.Failed, plan, 0)
		suite.AssertCorrectMetricValueT2(internal.OperationTypeDeprovision, domain.Failed, plan, 0)
		suite.AssertCorrectMetricValueT2(internal.OperationTypeUpdate, domain.Failed, plan, 0)
	})
}

