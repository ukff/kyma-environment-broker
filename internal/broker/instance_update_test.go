package broker

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/broker/automock"
	"github.com/kyma-project/kyma-environment-broker/internal/k8s"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/stretchr/testify/mock"

	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"
	"github.com/kyma-project/kyma-environment-broker/internal/dashboard"

	"github.com/kyma-project/kyma-environment-broker/internal/fixture"
	kcMock "github.com/kyma-project/kyma-environment-broker/internal/kubeconfig/automock"
	"github.com/kyma-project/kyma-environment-broker/internal/ptr"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"github.com/pivotal-cf/brokerapi/v8/domain"
	"github.com/pivotal-cf/brokerapi/v8/domain/apiresponses"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var dashboardConfig = dashboard.Config{LandscapeURL: "https://dashboard.example.com"}

type handler struct {
	Instance   internal.Instance
	ersContext internal.ERSContext
}

func (h *handler) Handle(inst *internal.Instance, ers internal.ERSContext) (bool, error) {
	h.Instance = *inst
	h.ersContext = ers
	return false, nil
}

func TestUpdateEndpoint_UpdateSuspension(t *testing.T) {
	// given
	instance := internal.Instance{
		InstanceID:    instanceID,
		ServicePlanID: TrialPlanID,
		Parameters: internal.ProvisioningParameters{
			PlanID: TrialPlanID,
			ErsContext: internal.ERSContext{
				TenantID:        "",
				SubAccountID:    "",
				GlobalAccountID: "",
				Active:          nil,
			},
		},
	}
	st := storage.NewMemoryStorage()
	err := st.Instances().Insert(instance)
	require.NoError(t, err)
	err = st.Operations().InsertProvisioningOperation(fixProvisioningOperation("01"))
	require.NoError(t, err)
	err = st.Operations().InsertDeprovisioningOperation(fixSuspensionOperation())
	require.NoError(t, err)
	err = st.Operations().InsertProvisioningOperation(fixProvisioningOperation("02"))
	require.NoError(t, err)

	handler := &handler{}
	q := &automock.Queue{}
	q.On("Add", mock.AnythingOfType("string"))
	planDefaults := func(planID string, platformProvider internal.CloudProvider, provider *internal.CloudProvider) (*gqlschema.ClusterConfigInput, error) {
		return &gqlschema.ClusterConfigInput{}, nil
	}
	kcBuilder := &kcMock.KcBuilder{}
	svc := NewUpdate(
		Config{},
		st.Instances(),
		st.RuntimeStates(),
		st.Operations(),
		handler,
		true,
		false,
		true,
		q,
		PlansConfig{},
		planDefaults,
		logrus.New(),
		dashboardConfig,
		kcBuilder,
		&OneForAllConvergedCloudRegionsProvider{})

	// when
	response, err := svc.Update(context.Background(), instanceID, domain.UpdateDetails{
		ServiceID:       "",
		PlanID:          TrialPlanID,
		RawParameters:   nil,
		PreviousValues:  domain.PreviousValues{},
		RawContext:      json.RawMessage("{\"active\":false}"),
		MaintenanceInfo: nil,
	}, true)
	require.NoError(t, err)

	// then

	assert.Equal(t, internal.ERSContext{
		Active: ptr.Bool(false),
	}, handler.ersContext)

	require.NotNil(t, handler.Instance.Parameters.ErsContext.Active)
	assert.True(t, *handler.Instance.Parameters.ErsContext.Active)
	assert.Len(t, response.Metadata.Labels, 1)

	inst, err := st.Instances().GetByID(instanceID)
	assert.False(t, *inst.Parameters.ErsContext.Active)
}

func TestUpdateEndpoint_UpdateOfExpiredTrial(t *testing.T) {
	// given
	instance := internal.Instance{
		InstanceID:    instanceID,
		ServicePlanID: TrialPlanID,
		Parameters: internal.ProvisioningParameters{
			PlanID: TrialPlanID,
			ErsContext: internal.ERSContext{
				TenantID:        "",
				SubAccountID:    "",
				GlobalAccountID: "",
				Active:          ptr.Bool(false),
			},
		},
		ExpiredAt: ptr.Time(time.Now()),
	}
	st := storage.NewMemoryStorage()
	err := st.Instances().Insert(instance)
	require.NoError(t, err)
	err = st.Operations().InsertProvisioningOperation(fixProvisioningOperation("01"))
	require.NoError(t, err)

	handler := &handler{}
	q := &automock.Queue{}
	q.On("Add", mock.AnythingOfType("string"))
	planDefaults := func(planID string, platformProvider internal.CloudProvider, provider *internal.CloudProvider) (*gqlschema.ClusterConfigInput, error) {
		return &gqlschema.ClusterConfigInput{}, nil
	}
	kcBuilder := &kcMock.KcBuilder{}
	svc := NewUpdate(Config{}, st.Instances(), st.RuntimeStates(), st.Operations(), handler, true, false, true, q, PlansConfig{},
		planDefaults, logrus.New(), dashboardConfig, kcBuilder, &OneForAllConvergedCloudRegionsProvider{})

	// when
	response, err := svc.Update(context.Background(), instanceID, domain.UpdateDetails{
		ServiceID:       "",
		PlanID:          TrialPlanID,
		RawParameters:   json.RawMessage(`{"autoScalerMin": 3}`),
		PreviousValues:  domain.PreviousValues{},
		RawContext:      json.RawMessage("{\"active\":false}"),
		MaintenanceInfo: nil,
	}, true)

	// then
	assert.Error(t, err)
	assert.ErrorContains(t, err, "cannot update an expired instance")
	assert.IsType(t, err, &apiresponses.FailureResponse{}, "Updating returned error of unexpected type")
	apierr := err.(*apiresponses.FailureResponse)
	assert.Equal(t, apierr.ValidatedStatusCode(nil), http.StatusBadRequest, "Updating status code not matching")
	assert.False(t, response.IsAsync)
}

func TestUpdateEndpoint_UpdateAutoscalerParams(t *testing.T) {
	// given
	instance := internal.Instance{
		InstanceID:    instanceID,
		ServicePlanID: AWSPlanID,
		Parameters: internal.ProvisioningParameters{
			PlanID: AWSPlanID,
			ErsContext: internal.ERSContext{
				TenantID:        "",
				SubAccountID:    "",
				GlobalAccountID: "",
				Active:          ptr.Bool(false),
			},
		},
	}
	st := storage.NewMemoryStorage()
	err := st.Instances().Insert(instance)
	require.NoError(t, err)
	err = st.Operations().InsertProvisioningOperation(fixProvisioningOperation("01"))
	require.NoError(t, err)

	handler := &handler{}
	q := &automock.Queue{}
	q.On("Add", mock.AnythingOfType("string"))
	planDefaults := func(planID string, platformProvider internal.CloudProvider, provider *internal.CloudProvider) (*gqlschema.ClusterConfigInput, error) {
		return &gqlschema.ClusterConfigInput{}, nil
	}
	kcBuilder := &kcMock.KcBuilder{}
	svc := NewUpdate(Config{}, st.Instances(), st.RuntimeStates(), st.Operations(), handler, true, false, true, q, PlansConfig{},
		planDefaults, logrus.New(), dashboardConfig, kcBuilder, &OneForAllConvergedCloudRegionsProvider{})

	t.Run("Should fail on invalid (too low) autoScalerMin and autoScalerMax", func(t *testing.T) {

		// when
		response, err := svc.Update(context.Background(), instanceID, domain.UpdateDetails{
			ServiceID:       "",
			PlanID:          AWSPlanID,
			RawParameters:   json.RawMessage(`{"autoScalerMin": 1, "autoScalerMax": 1}`),
			PreviousValues:  domain.PreviousValues{},
			RawContext:      json.RawMessage("{\"active\":false}"),
			MaintenanceInfo: nil,
		}, true)

		// then
		assert.ErrorContains(t, err, "while validating update parameters:")
		assert.IsType(t, err, &apiresponses.FailureResponse{}, "Updating returned error of unexpected type")
		apierr := err.(*apiresponses.FailureResponse)
		assert.Equal(t, apierr.ValidatedStatusCode(nil), http.StatusBadRequest, "Updating status code not matching")
		assert.False(t, response.IsAsync)
	})

	t.Run("Should fail on invalid autoScalerMin and autoScalerMax", func(t *testing.T) {

		// when
		response, err := svc.Update(context.Background(), instanceID, domain.UpdateDetails{
			ServiceID:       "",
			PlanID:          AWSPlanID,
			RawParameters:   json.RawMessage(`{"autoScalerMin": 4, "autoScalerMax": 3}`),
			PreviousValues:  domain.PreviousValues{},
			RawContext:      json.RawMessage("{\"active\":false}"),
			MaintenanceInfo: nil,
		}, true)

		// then
		assert.ErrorContains(t, err, "AutoScalerMax 3 should be larger than AutoScalerMin 4")
		assert.IsType(t, err, &apiresponses.FailureResponse{}, "Updating returned error of unexpected type")
		apierr := err.(*apiresponses.FailureResponse)
		assert.Equal(t, apierr.ValidatedStatusCode(nil), http.StatusBadRequest, "Updating status code not matching")
		assert.False(t, response.IsAsync)
	})

	t.Run("Should fail on invalid autoScalerMin and autoScalerMax and JSON validation should precede", func(t *testing.T) {

		// when
		response, err := svc.Update(context.Background(), instanceID, domain.UpdateDetails{
			ServiceID:       "",
			PlanID:          AWSPlanID,
			RawParameters:   json.RawMessage(`{"autoScalerMin": 2, "autoScalerMax": 1}`),
			PreviousValues:  domain.PreviousValues{},
			RawContext:      json.RawMessage("{\"active\":false}"),
			MaintenanceInfo: nil,
		}, true)

		// then
		assert.ErrorContains(t, err, "while validating update parameters:")
		assert.IsType(t, err, &apiresponses.FailureResponse{}, "Updating returned error of unexpected type")
		apierr := err.(*apiresponses.FailureResponse)
		assert.Equal(t, apierr.ValidatedStatusCode(nil), http.StatusBadRequest, "Updating status code not matching")
		assert.False(t, response.IsAsync)
	})
}

func TestUpdateEndpoint_UpdateUnsuspension(t *testing.T) {
	// given
	instance := internal.Instance{
		InstanceID:    instanceID,
		ServicePlanID: TrialPlanID,
		Parameters: internal.ProvisioningParameters{
			PlanID: TrialPlanID,
			ErsContext: internal.ERSContext{
				TenantID:        "",
				SubAccountID:    "",
				GlobalAccountID: "",
				Active:          nil,
			},
		},
	}
	st := storage.NewMemoryStorage()
	err := st.Instances().Insert(instance)
	require.NoError(t, err)
	err = st.Operations().InsertProvisioningOperation(fixProvisioningOperation("01"))
	require.NoError(t, err)
	err = st.Operations().InsertDeprovisioningOperation(fixSuspensionOperation())
	require.NoError(t, err)

	handler := &handler{}
	q := &automock.Queue{}
	q.On("Add", mock.AnythingOfType("string"))
	planDefaults := func(planID string, platformProvider internal.CloudProvider, provider *internal.CloudProvider) (*gqlschema.ClusterConfigInput, error) {
		return &gqlschema.ClusterConfigInput{}, nil
	}
	kcBuilder := &kcMock.KcBuilder{}
	svc := NewUpdate(Config{}, st.Instances(), st.RuntimeStates(), st.Operations(), handler, true, false, true, q, PlansConfig{},
		planDefaults, logrus.New(), dashboardConfig, kcBuilder, &OneForAllConvergedCloudRegionsProvider{})

	// when
	_, err = svc.Update(context.Background(), instanceID, domain.UpdateDetails{
		ServiceID:       "",
		PlanID:          TrialPlanID,
		RawParameters:   nil,
		PreviousValues:  domain.PreviousValues{},
		RawContext:      json.RawMessage("{\"active\":true}"),
		MaintenanceInfo: nil,
	}, true)
	require.NoError(t, err)

	// then

	assert.Equal(t, internal.ERSContext{
		Active: ptr.Bool(true),
	}, handler.ersContext)

	require.NotNil(t, handler.Instance.Parameters.ErsContext.Active)
	assert.False(t, *handler.Instance.Parameters.ErsContext.Active)
}

func TestUpdateEndpoint_UpdateInstanceWithWrongActiveValue(t *testing.T) {
	// given
	instance := internal.Instance{
		InstanceID:    instanceID,
		ServicePlanID: TrialPlanID,
		Parameters: internal.ProvisioningParameters{
			PlanID: TrialPlanID,
			ErsContext: internal.ERSContext{
				TenantID:        "",
				SubAccountID:    "",
				GlobalAccountID: "",
				Active:          ptr.Bool(false),
			},
		},
	}
	st := storage.NewMemoryStorage()
	err := st.Instances().Insert(instance)
	require.NoError(t, err)
	err = st.Operations().InsertProvisioningOperation(fixProvisioningOperation("01"))
	require.NoError(t, err)
	handler := &handler{}
	q := &automock.Queue{}
	q.On("Add", mock.AnythingOfType("string"))
	planDefaults := func(planID string, platformProvider internal.CloudProvider, provider *internal.CloudProvider) (*gqlschema.ClusterConfigInput, error) {
		return &gqlschema.ClusterConfigInput{}, nil
	}
	kcBuilder := &kcMock.KcBuilder{}
	svc := NewUpdate(Config{}, st.Instances(), st.RuntimeStates(), st.Operations(), handler, true, false, true, q, PlansConfig{},
		planDefaults, logrus.New(), dashboardConfig, kcBuilder, &OneForAllConvergedCloudRegionsProvider{})

	// when
	_, err = svc.Update(context.Background(), instanceID, domain.UpdateDetails{
		ServiceID:       "",
		PlanID:          TrialPlanID,
		RawParameters:   nil,
		PreviousValues:  domain.PreviousValues{},
		RawContext:      json.RawMessage("{\"active\":false}"),
		MaintenanceInfo: nil,
	}, true)
	require.NoError(t, err)

	// then
	assert.Equal(t, internal.ERSContext{
		Active: ptr.Bool(false),
	}, handler.ersContext)

	assert.True(t, *handler.Instance.Parameters.ErsContext.Active)
}

func TestUpdateEndpoint_UpdateNonExistingInstance(t *testing.T) {
	// given
	st := storage.NewMemoryStorage()
	handler := &handler{}
	q := &automock.Queue{}
	q.On("Add", mock.AnythingOfType("string"))
	planDefaults := func(planID string, platformProvider internal.CloudProvider, provider *internal.CloudProvider) (*gqlschema.ClusterConfigInput, error) {
		return &gqlschema.ClusterConfigInput{}, nil
	}
	kcBuilder := &kcMock.KcBuilder{}
	svc := NewUpdate(Config{}, st.Instances(), st.RuntimeStates(), st.Operations(), handler, true, false, true, q, PlansConfig{},
		planDefaults, logrus.New(), dashboardConfig, kcBuilder, &OneForAllConvergedCloudRegionsProvider{})

	// when
	_, err := svc.Update(context.Background(), instanceID, domain.UpdateDetails{
		ServiceID:       "",
		PlanID:          TrialPlanID,
		RawParameters:   nil,
		PreviousValues:  domain.PreviousValues{},
		RawContext:      json.RawMessage("{\"active\":false}"),
		MaintenanceInfo: nil,
	}, true)

	// then
	assert.IsType(t, err, &apiresponses.FailureResponse{}, "Updating returned error of unexpected type")
	apierr := err.(*apiresponses.FailureResponse)
	assert.Equal(t, apierr.ValidatedStatusCode(nil), http.StatusNotFound, "Updating status code not matching")
}

func fixProvisioningOperation(id string) internal.ProvisioningOperation {
	provisioningOperation := fixture.FixProvisioningOperation(id, instanceID)

	return internal.ProvisioningOperation{Operation: provisioningOperation}
}

func fixSuspensionOperation() internal.DeprovisioningOperation {
	deprovisioningOperation := fixture.FixDeprovisioningOperation("id", instanceID)
	deprovisioningOperation.Temporary = true

	return deprovisioningOperation
}

func TestUpdateEndpoint_UpdateGlobalAccountID(t *testing.T) {
	// given
	instance := internal.Instance{
		InstanceID:      instanceID,
		ServicePlanID:   TrialPlanID,
		GlobalAccountID: "origin-account-id",
		Parameters: internal.ProvisioningParameters{
			PlanID: TrialPlanID,
			ErsContext: internal.ERSContext{
				TenantID:        "",
				SubAccountID:    "",
				GlobalAccountID: "",
				Active:          nil,
			},
		},
	}
	newGlobalAccountID := "updated-account-id"
	st := storage.NewMemoryStorage()
	err := st.Instances().Insert(instance)
	require.NoError(t, err)
	err = st.Operations().InsertProvisioningOperation(fixProvisioningOperation("01"))
	require.NoError(t, err)
	err = st.Operations().InsertDeprovisioningOperation(fixSuspensionOperation())
	require.NoError(t, err)
	err = st.Operations().InsertProvisioningOperation(fixProvisioningOperation("02"))
	require.NoError(t, err)

	handler := &handler{}
	q := &automock.Queue{}
	q.On("Add", mock.AnythingOfType("string"))
	planDefaults := func(planID string, platformProvider internal.CloudProvider, provider *internal.CloudProvider) (*gqlschema.ClusterConfigInput, error) {
		return &gqlschema.ClusterConfigInput{}, nil
	}
	kcBuilder := &kcMock.KcBuilder{}
	svc := NewUpdate(Config{}, st.Instances(), st.RuntimeStates(), st.Operations(), handler, true, true, true, q, PlansConfig{},
		planDefaults, logrus.New(), dashboardConfig, kcBuilder, &OneForAllConvergedCloudRegionsProvider{})

	// when
	response, err := svc.Update(context.Background(), instanceID, domain.UpdateDetails{
		ServiceID:       "",
		PlanID:          TrialPlanID,
		RawParameters:   nil,
		PreviousValues:  domain.PreviousValues{},
		RawContext:      json.RawMessage("{\"globalaccount_id\":\"" + newGlobalAccountID + "\", \"active\":true}"),
		MaintenanceInfo: nil,
	}, true)
	require.NoError(t, err)

	// then
	inst, err := st.Instances().GetByID(instanceID)
	require.NoError(t, err)
	// Check if SubscriptionGlobalAccountID is not empty
	assert.NotEmpty(t, inst.SubscriptionGlobalAccountID)

	// Check if SubscriptionGlobalAccountID is now the same as GlobalAccountID
	assert.Equal(t, inst.GlobalAccountID, newGlobalAccountID)

	require.NotNil(t, handler.Instance.Parameters.ErsContext.Active)
	assert.True(t, *handler.Instance.Parameters.ErsContext.Active)
	assert.Len(t, response.Metadata.Labels, 1)
}

func TestUpdateEndpoint_UpdateParameters(t *testing.T) {
	// given
	instance := fixture.FixInstance(instanceID)
	st := storage.NewMemoryStorage()
	err := st.Instances().Insert(instance)
	require.NoError(t, err)
	err = st.Operations().InsertProvisioningOperation(fixProvisioningOperation("provisioning01"))
	require.NoError(t, err)

	handler := &handler{}
	q := &automock.Queue{}
	q.On("Add", mock.AnythingOfType("string"))
	planDefaults := func(planID string, platformProvider internal.CloudProvider, provider *internal.CloudProvider) (*gqlschema.ClusterConfigInput, error) {
		return &gqlschema.ClusterConfigInput{}, nil
	}
	kcBuilder := &kcMock.KcBuilder{}
	svc := NewUpdate(Config{}, st.Instances(), st.RuntimeStates(), st.Operations(), handler, true, true, true, q, PlansConfig{},
		planDefaults, logrus.New(), dashboardConfig, kcBuilder, &OneForAllConvergedCloudRegionsProvider{})

	t.Run("Should fail on invalid OIDC params", func(t *testing.T) {
		// given
		oidcParams := `"clientID":"{clientID}","groupsClaim":"groups","issuerURL":"{issuerURL}","signingAlgs":["RS256"],"usernameClaim":"email","usernamePrefix":"-"`
		errMsg := fmt.Errorf("issuerURL must be a valid URL, issuerURL must have https scheme")
		expectedErr := apiresponses.NewFailureResponse(errMsg, http.StatusUnprocessableEntity, errMsg.Error())

		// when
		_, err := svc.Update(context.Background(), instanceID, domain.UpdateDetails{
			ServiceID:       "",
			PlanID:          AzurePlanID,
			RawParameters:   json.RawMessage("{\"oidc\":{" + oidcParams + "}}"),
			PreviousValues:  domain.PreviousValues{},
			RawContext:      json.RawMessage("{\"globalaccount_id\":\"globalaccount_id_1\", \"active\":true}"),
			MaintenanceInfo: nil,
		}, true)

		// then
		require.Error(t, err)
		assert.IsType(t, &apiresponses.FailureResponse{}, err)
		apierr := err.(*apiresponses.FailureResponse)
		assert.Equal(t, expectedErr.ValidatedStatusCode(nil), apierr.ValidatedStatusCode(nil))
		assert.Equal(t, expectedErr.LoggerAction(), apierr.LoggerAction())
	})

	t.Run("Should fail on insufficient OIDC params (missing issuerURL)", func(t *testing.T) {
		// given
		oidcParams := `"clientID":"client-id"`
		errMsg := fmt.Errorf("issuerURL must not be empty")
		expectedErr := apiresponses.NewFailureResponse(errMsg, http.StatusUnprocessableEntity, errMsg.Error())

		// when
		_, err := svc.Update(context.Background(), instanceID, domain.UpdateDetails{
			ServiceID:       "",
			PlanID:          AzurePlanID,
			RawParameters:   json.RawMessage("{\"oidc\":{" + oidcParams + "}}"),
			PreviousValues:  domain.PreviousValues{},
			RawContext:      json.RawMessage("{\"globalaccount_id\":\"globalaccount_id_1\", \"active\":true}"),
			MaintenanceInfo: nil,
		}, true)

		// then
		require.Error(t, err)
		assert.IsType(t, &apiresponses.FailureResponse{}, err)
		apierr := err.(*apiresponses.FailureResponse)
		assert.Equal(t, expectedErr.ValidatedStatusCode(nil), apierr.ValidatedStatusCode(nil))
		assert.Equal(t, expectedErr.LoggerAction(), apierr.LoggerAction())
	})

	t.Run("Should fail on insufficient OIDC params (missing clientID)", func(t *testing.T) {
		// given
		oidcParams := `"issuerURL":"https://test.local"`
		errMsg := fmt.Errorf("clientID must not be empty")
		expectedErr := apiresponses.NewFailureResponse(errMsg, http.StatusUnprocessableEntity, errMsg.Error())

		// when
		_, err := svc.Update(context.Background(), instanceID, domain.UpdateDetails{
			ServiceID:       "",
			PlanID:          AzurePlanID,
			RawParameters:   json.RawMessage("{\"oidc\":{" + oidcParams + "}}"),
			PreviousValues:  domain.PreviousValues{},
			RawContext:      json.RawMessage("{\"globalaccount_id\":\"globalaccount_id_1\", \"active\":true}"),
			MaintenanceInfo: nil,
		}, true)

		// then
		require.Error(t, err)
		assert.IsType(t, &apiresponses.FailureResponse{}, err)
		apierr := err.(*apiresponses.FailureResponse)
		assert.Equal(t, expectedErr.ValidatedStatusCode(nil), apierr.ValidatedStatusCode(nil))
		assert.Equal(t, expectedErr.LoggerAction(), apierr.LoggerAction())
	})

	t.Run("Should fail on invalid OIDC signingAlgs param", func(t *testing.T) {
		// given
		oidcParams := `"clientID":"client-id","issuerURL":"https://test.local","signingAlgs":["RS256","notValid"]`
		errMsg := fmt.Errorf("signingAlgs must contain valid signing algorithm(s)")
		expectedErr := apiresponses.NewFailureResponse(errMsg, http.StatusUnprocessableEntity, errMsg.Error())

		// when
		_, err := svc.Update(context.Background(), instanceID, domain.UpdateDetails{
			ServiceID:       "",
			PlanID:          AzurePlanID,
			RawParameters:   json.RawMessage("{\"oidc\":{" + oidcParams + "}}"),
			PreviousValues:  domain.PreviousValues{},
			RawContext:      json.RawMessage("{\"globalaccount_id\":\"globalaccount_id_1\", \"active\":true}"),
			MaintenanceInfo: nil,
		}, true)

		// then
		require.Error(t, err)
		assert.IsType(t, &apiresponses.FailureResponse{}, err)
		apierr := err.(*apiresponses.FailureResponse)
		assert.Equal(t, expectedErr.ValidatedStatusCode(nil), apierr.ValidatedStatusCode(nil))
		assert.Equal(t, expectedErr.LoggerAction(), apierr.LoggerAction())
	})
}

func TestUpdateEndpoint_UpdateWithEnabledDashboard(t *testing.T) {
	// given
	instance := internal.Instance{
		InstanceID:    instanceID,
		ServicePlanID: TrialPlanID,
		Parameters: internal.ProvisioningParameters{
			PlanID: TrialPlanID,
			ErsContext: internal.ERSContext{
				TenantID:        "",
				SubAccountID:    "",
				GlobalAccountID: "",
				Active:          nil,
			},
		},
		DashboardURL: "https://console.cd6e47b.example.com",
	}
	st := storage.NewMemoryStorage()
	err := st.Instances().Insert(instance)
	require.NoError(t, err)
	err = st.Operations().InsertProvisioningOperation(fixProvisioningOperation("01"))
	require.NoError(t, err)
	// st.Operations().InsertDeprovisioningOperation(fixSuspensionOperation())
	// st.Operations().InsertProvisioningOperation(fixProvisioningOperation("02"))

	handler := &handler{}
	q := &automock.Queue{}
	q.On("Add", mock.AnythingOfType("string"))
	planDefaults := func(planID string, platformProvider internal.CloudProvider, provider *internal.CloudProvider) (*gqlschema.ClusterConfigInput, error) {
		return &gqlschema.ClusterConfigInput{}, nil
	}
	kcBuilder := &kcMock.KcBuilder{}
	svc := NewUpdate(Config{AllowUpdateExpiredInstanceWithContext: true}, st.Instances(), st.RuntimeStates(), st.Operations(), handler, true, false, true, q, PlansConfig{},
		planDefaults, logrus.New(), dashboardConfig, kcBuilder, &OneForAllConvergedCloudRegionsProvider{})

	// when
	response, err := svc.Update(context.Background(), instanceID, domain.UpdateDetails{
		ServiceID:       "",
		PlanID:          TrialPlanID,
		RawParameters:   nil,
		PreviousValues:  domain.PreviousValues{},
		RawContext:      json.RawMessage("{\"active\":false}"),
		MaintenanceInfo: nil,
	}, true)
	require.NoError(t, err)

	// then
	inst, err := st.Instances().GetByID(instanceID)
	require.NoError(t, err)

	// check if the instance is updated successfully
	assert.Regexp(t, `^https:\/\/dashboard\.example\.com\/\?kubeconfigID=`, inst.DashboardURL)
	// check if the API response is correct
	assert.Regexp(t, `^https:\/\/dashboard\.example\.com\/\?kubeconfigID=`, response.DashboardURL)
}

func TestUpdateExpiredInstance(t *testing.T) {
	instance := internal.Instance{
		InstanceID:      instanceID,
		ServicePlanID:   TrialPlanID,
		GlobalAccountID: "globalaccount_id_init",
		Parameters: internal.ProvisioningParameters{
			PlanID:     TrialPlanID,
			ErsContext: internal.ERSContext{},
		},
	}
	expireTime := instance.CreatedAt.Add(time.Hour * 24 * 14)
	instance.ExpiredAt = &expireTime

	storage := storage.NewMemoryStorage()
	err := storage.Instances().Insert(instance)
	require.NoError(t, err)

	err = storage.Operations().InsertProvisioningOperation(fixProvisioningOperation("01"))
	require.NoError(t, err)

	kcBuilder := &kcMock.KcBuilder{}

	handler := &handler{}

	queue := &automock.Queue{}
	queue.On("Add", mock.AnythingOfType("string"))

	planDefaults := func(planID string, platformProvider internal.CloudProvider, provider *internal.CloudProvider) (*gqlschema.ClusterConfigInput, error) {
		return &gqlschema.ClusterConfigInput{}, nil
	}

	svc := NewUpdate(Config{AllowUpdateExpiredInstanceWithContext: true}, storage.Instances(), storage.RuntimeStates(), storage.Operations(), handler, true, false, true, queue, PlansConfig{},
		planDefaults, logrus.New(), dashboardConfig, kcBuilder, &OneForAllConvergedCloudRegionsProvider{})

	t.Run("should reject change GA - it is same as previous", func(t *testing.T) {
		_, err = svc.Update(context.Background(), instanceID, domain.UpdateDetails{
			ServiceID:       KymaServiceID,
			PlanID:          TrialPlanID,
			RawParameters:   nil,
			PreviousValues:  domain.PreviousValues{},
			RawContext:      json.RawMessage("{\"globalaccount_id\":\"globalaccount_id_init\"}"),
			MaintenanceInfo: nil,
		}, true)
		require.Error(t, err)
	})

	t.Run("should accept change GA", func(t *testing.T) {
		_, err = svc.Update(context.Background(), instanceID, domain.UpdateDetails{
			ServiceID:       KymaServiceID,
			PlanID:          TrialPlanID,
			RawParameters:   nil,
			PreviousValues:  domain.PreviousValues{},
			RawContext:      json.RawMessage("{\"globalaccount_id\":\"globalaccount_id_new\"}"),
			MaintenanceInfo: nil,
		}, true)
		require.NoError(t, err)
	})

	t.Run("should accept change GA, with params", func(t *testing.T) {
		_, err = svc.Update(context.Background(), instanceID, domain.UpdateDetails{
			ServiceID:       KymaServiceID,
			PlanID:          TrialPlanID,
			PreviousValues:  domain.PreviousValues{},
			RawContext:      json.RawMessage("{\"globalaccount_id\":\"globalaccount_id_new_2\", \"active\":true}"),
			RawParameters:   json.RawMessage(`{"autoScalerMin": 4, "autoScalerMax": 3}`),
			MaintenanceInfo: nil,
		}, true)
		require.NoError(t, err)
	})

	t.Run("should fail as not global account passed", func(t *testing.T) {
		_, err = svc.Update(context.Background(), instanceID, domain.UpdateDetails{
			ServiceID:       KymaServiceID,
			PlanID:          TrialPlanID,
			PreviousValues:  domain.PreviousValues{},
			RawContext:      json.RawMessage("{\"x\":\"y\", \"active\":true}"),
			RawParameters:   json.RawMessage(`{"autoScalerMin": 4, "autoScalerMax": 3}`),
			MaintenanceInfo: nil,
		}, true)
		require.Error(t, err)
	})
}

func TestSubaccountMovement(t *testing.T) {
	instance := internal.Instance{
		InstanceID:      instanceID,
		ServicePlanID:   TrialPlanID,
		GlobalAccountID: "InitialGlobalAccountID",
		Parameters: internal.ProvisioningParameters{
			PlanID:     TrialPlanID,
			ErsContext: internal.ERSContext{},
		},
	}

	storage := storage.NewMemoryStorage()
	err := storage.Instances().Insert(instance)
	require.NoError(t, err)

	err = storage.Operations().InsertProvisioningOperation(fixProvisioningOperation("01"))
	require.NoError(t, err)

	kcBuilder := &kcMock.KcBuilder{}

	handler := &handler{}

	queue := &automock.Queue{}
	queue.On("Add", mock.AnythingOfType("string"))

	planDefaults := func(planID string, platformProvider internal.CloudProvider, provider *internal.CloudProvider) (*gqlschema.ClusterConfigInput, error) {
		return &gqlschema.ClusterConfigInput{}, nil
	}

	svc := NewUpdate(Config{SubaccountMovementEnabled: true}, storage.Instances(), storage.RuntimeStates(), storage.Operations(), handler, true, true, true, queue, PlansConfig{},
		planDefaults, logrus.New(), dashboardConfig, kcBuilder, &OneForAllConvergedCloudRegionsProvider{})

	t.Run("no move performed so subscription should be empty", func(t *testing.T) {
		_, err = svc.Update(context.Background(), instanceID, domain.UpdateDetails{
			ServiceID:       KymaServiceID,
			PlanID:          TrialPlanID,
			PreviousValues:  domain.PreviousValues{},
			RawContext:      json.RawMessage("{\"globalaccount_id\":\"InitialGlobalAccountID\", \"active\":true}"),
			RawParameters:   json.RawMessage("{\"name\":\"test\"}"),
			MaintenanceInfo: nil,
		}, true)
		require.NoError(t, err)
		instance, err := storage.Instances().GetByID(instanceID)
		require.NoError(t, err)
		assert.Equal(t, "", instance.SubscriptionGlobalAccountID)
		assert.Equal(t, "InitialGlobalAccountID", instance.GlobalAccountID)
	})

	t.Run("move subaccount first time", func(t *testing.T) {
		_, err = svc.Update(context.Background(), instanceID, domain.UpdateDetails{
			ServiceID:       KymaServiceID,
			PlanID:          TrialPlanID,
			PreviousValues:  domain.PreviousValues{},
			RawContext:      json.RawMessage("{\"globalaccount_id\":\"newGlobalAccountID-v1\"}"),
			MaintenanceInfo: nil,
		}, true)
		require.NoError(t, err)
		instance, err := storage.Instances().GetByID(instanceID)
		require.NoError(t, err)
		assert.Equal(t, "InitialGlobalAccountID", instance.SubscriptionGlobalAccountID)
		assert.Equal(t, "newGlobalAccountID-v1", instance.GlobalAccountID)
	})

	t.Run("move subaccount second time", func(t *testing.T) {
		_, err = svc.Update(context.Background(), instanceID, domain.UpdateDetails{
			ServiceID:       KymaServiceID,
			PlanID:          TrialPlanID,
			PreviousValues:  domain.PreviousValues{},
			RawContext:      json.RawMessage("{\"globalaccount_id\":\"newGlobalAccountID-v2\"}"),
			MaintenanceInfo: nil,
		}, true)
		require.NoError(t, err)
		instance, err := storage.Instances().GetByID(instanceID)
		require.NoError(t, err)
		assert.Equal(t, "InitialGlobalAccountID", instance.SubscriptionGlobalAccountID)
		assert.Equal(t, "newGlobalAccountID-v2", instance.GlobalAccountID)
	})
}

func TestLabelChangeWhenMovingSubaccount(t *testing.T) {
	const (
		oldGlobalAccountId = "old-global-account-id"
		newGlobalAccountId = "changed-global-account-id"
	)

	iid := uuid.New().String()

	tFunc := func(t *testing.T, name, crName string) {
		gvk, err := k8s.GvkByName(crName)
		require.NoError(t, err)

		cr := &unstructured.Unstructured{}
		cr.SetGroupVersionKind(gvk)
		cr.SetName(name)
		cr.SetNamespace(KymaNamespace)

		labels := cr.GetLabels()
		assert.Empty(t, labels)

		existingLabels := make(map[string]string)
		existingLabels[k8s.GlobalAccountIdLabel] = oldGlobalAccountId
		existingLabels["foo"] = "bar"
		cr.SetLabels(existingLabels)

		labels = cr.GetLabels()
		assert.Len(t, cr.GetLabels(), 2)
		assert.Equal(t, oldGlobalAccountId, labels[k8s.GlobalAccountIdLabel])
		assert.Equal(t, "bar", labels["foo"])

		// update CR with new global account id
		err = k8s.AddOrOverrideMetadata(cr, k8s.GlobalAccountIdLabel, newGlobalAccountId)
		require.NoError(t, err)

		labels = cr.GetLabels()
		assert.Len(t, labels, 2)
		assert.Equal(t, newGlobalAccountId, labels[k8s.GlobalAccountIdLabel])
		assert.Equal(t, "bar", labels["foo"])
	}

	t.Run("KymaCr should have correct and new global account id", func(t *testing.T) {
		tFunc(t, iid, k8s.KymaCr)
	})

	t.Run("GardenerClusterCr should have correct and new global account id", func(t *testing.T) {
		tFunc(t, iid, k8s.GardenerClusterCr)
	})

	t.Run("RuntimeCr hould have correct and new global account id", func(t *testing.T) {
		tFunc(t, iid, k8s.RuntimeCr)
	})
}
