package main

import (
	"fmt"
	"net/http"
	"testing"
	"time"
	
	"github.com/google/uuid"
	reconcilerApi "github.com/kyma-incubator/reconciler/pkg/keb"
	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/pivotal-cf/brokerapi/v8/domain"
	"github.com/stretchr/testify/assert"
)

func TestMetrics(t *testing.T) {

	plan := "361c511f-f939-4621-b228-d0fb79a1fe15"
	suite := NewBrokerSuitTestWithMetrics(t)
	defer suite.TearDown()

	provisionReq := func(iid, plan string) string {
		body := fmt.Sprintf(`{
					"service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
					"plan_id": "%s",
					"context": {
						"globalaccount_id": "g-account-id",
						"subaccount_id": "sub-id",
						"user_id": "john.smith@email.com"
					},
					"parameters": {
						"name": "testing-cluster",
						"region": "eu-central-1"
					}
		}`, plan)
		resp := suite.CallAPI("PUT", fmt.Sprintf("oauth/v2/service_instances/%s?accepts_incomplete=true", iid), body)
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

		opID := provisionReq(depSuc, plan)
		suite.processProvisioningByOperationID(opID)
		suite.WaitForOperationState(opID, domain.Succeeded)

		opID = provisionReq(provFail, plan)
		suite.failProvisioningByOperationID(opID)
		suite.WaitForOperationState(opID, domain.Failed)

		opID = provisionReq(updateSuccess, plan)
		suite.processProvisioningByOperationID(opID)
		suite.WaitForOperationState(opID, domain.Succeeded)

		opID = provisionReq(depFail,plan)
		suite.processProvisioningByOperationID(opID)
		suite.WaitForOperationState(opID, domain.Succeeded)

		opID = provisionReq(provSuc, plan)
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

		time.Sleep(1 * time.Second)
		suite.AssertMetric(internal.OperationTypeProvision, domain.Succeeded, plan, 4)
		suite.AssertMetric(internal.OperationTypeProvision, domain.Failed, plan, 1)
		suite.AssertMetric(internal.OperationTypeUpdate, domain.Succeeded, plan, 1)
		suite.AssertMetric(internal.OperationTypeUpdate, domain.Failed, plan, 0)
		suite.AssertMetric(internal.OperationTypeDeprovision, domain.Succeeded, plan, 1)
		suite.AssertMetric(internal.OperationTypeDeprovision, domain.Failed, plan, 1)
	})
}
