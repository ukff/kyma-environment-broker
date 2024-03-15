package main

import (
	`fmt`
	`net/http`
	`testing`
	
	`github.com/google/uuid`
	reconcilerApi `github.com/kyma-incubator/reconciler/pkg/keb`
	`github.com/kyma-project/kyma-environment-broker/internal`
	`github.com/kyma-project/kyma-environment-broker/internal/broker`
	`github.com/pivotal-cf/brokerapi/v8/domain`
	`github.com/stretchr/testify/assert`
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

// (trial, own, azure, gcp, aws) (trial, own, aws)
// (5 success) (3 fail)
// (3 update success) (2 update fail)
// 5 deprovisoning


func TestMetrics(t *testing.T) {
	
	suite := NewBrokerSuitTestWithMetrics(t)
	defer suite.TearDown()
	
	provisionReq := func(iid, plan string) string {
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
		return suite.DecodeOperationID(resp)
	}
	
	updateReq := func(iid string) string {
		resp := suite.CallAPI(
			"PATCH", fmt.Sprintf("oauth/v2/service_instances/%s?accepts_incomplete=true", iid), `
		{
			"service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
			"context": {
				"license_type": "CUSTOMER"
			}
		}`)
		assert.Equal(t, http.StatusAccepted, resp.StatusCode)
		return suite.DecodeOperationID(resp)
	}
	
	deleteReq := func(iid string) string {
		resp := suite.CallAPI(
			"DELETE", fmt.Sprintf(
				"oauth/v2/service_instances/%s?accepts_incomplete=true&plan_id=361c511f-f939-4621-b228-d0fb79a1fe15&service_id=47c9dcbf-ff30-448e-ab36-d3bad66ba281", iid), ``)
		assert.Equal(t, http.StatusAccepted, resp.StatusCode)
		return suite.DecodeOperationID(resp)
	}
	
	t.Run("AssertCorrectMetricValue", func(t *testing.T) {
		depSuc := uuid.New().String()
		provFail := uuid.New().String()
		updateSuccess := uuid.New().String()
		depFail := uuid.New().String()
		provSuc := uuid.New().String()
		
		opID := provisionReq(depSuc, broker.GCPPlanID)
		suite.processProvisioningByOperationID(opID)
		suite.WaitForOperationState(opID, domain.Succeeded)
		
		opID = provisionReq(provFail, broker.GCPPlanID)
		suite.failProvisioningByOperationID(opID)
		suite.WaitForOperationState(opID, domain.Failed)
		
		opID = provisionReq(updateSuccess, broker.GCPPlanID)
		suite.processProvisioningByOperationID(opID)
		suite.WaitForOperationState(opID, domain.Succeeded)
		
		opID = provisionReq(depFail, broker.GCPPlanID)
		suite.processProvisioningByOperationID(opID)
		suite.WaitForOperationState(opID, domain.Succeeded)
		
		opID = provisionReq(provSuc, broker.GCPPlanID)
		suite.processProvisioningByOperationID(opID)
		suite.WaitForOperationState(opID, domain.Succeeded)
		
		opID = updateReq(updateSuccess)
		suite.FinishUpdatingOperationByProvisioner(opID)
		suite.FinishUpdatingOperationByReconciler(opID)
		suite.WaitForOperationState(opID, domain.Succeeded)
		
		suite.SetReconcilerResponseStatus(reconcilerApi.StatusDeleted)
		opID = deleteReq(depSuc)
		suite.FinishDeprovisioningByReconciler(opID)
		suite.FinishDeprovisioningOperationByProvisioner(opID)
		suite.WaitForInstanceArchivedCreated(depSuc)
		suite.WaitFor(
			func() bool {
				resp := suite.CallAPI("GET", fmt.Sprintf("oauth/v2/service_instances/%s/last_operation", depSuc), ``)
				defer resp.Body.Close()
				data := suite.ParseLastOperationResponse(resp)
				return resp.StatusCode == http.StatusOK && data.State == domain.Succeeded
			})
		suite.WaitForOperationsNotExists(depSuc)
		
		suite.SetReconcilerResponseStatus(reconcilerApi.StatusError)
		opID = deleteReq(depFail)
		suite.FailDeprovisioningOperationByProvisioner(opID)
		suite.WaitForOperationState(opID, domain.Failed)
		
		/*
		suite.SetReconcilerResponseStatus(reconcilerApi.StatusDeleteError)
		opID = deleteReq(updateSuccess)
		suite.FailDeprovisioningByReconciler(opID)
		suite.WaitForOperationState(opID, domain.Failed)*/
		
		suite.AssertMetric(internal.OperationTypeProvision, domain.Succeeded, "361c511f-f939-4621-b228-d0fb79a1fe15", 4)
		suite.AssertMetric(internal.OperationTypeProvision, domain.Failed, "361c511f-f939-4621-b228-d0fb79a1fe15", 1)
		suite.AssertMetric(internal.OperationTypeUpdate, domain.Succeeded, "361c511f-f939-4621-b228-d0fb79a1fe15", 1)
		suite.AssertMetric(internal.OperationTypeUpdate, domain.Failed, "361c511f-f939-4621-b228-d0fb79a1fe15", 0)
		suite.AssertMetric(internal.OperationTypeDeprovision, domain.Succeeded, "361c511f-f939-4621-b228-d0fb79a1fe15", 1)
		suite.AssertMetric(internal.OperationTypeDeprovision, domain.Failed, "361c511f-f939-4621-b228-d0fb79a1fe15", 1)
	})
}