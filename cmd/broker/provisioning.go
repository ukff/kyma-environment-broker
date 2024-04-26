package main

import (
	"context"

	"github.com/kyma-project/kyma-environment-broker/common/hyperscaler"
	"github.com/kyma-project/kyma-environment-broker/internal/avs"
	"github.com/kyma-project/kyma-environment-broker/internal/process"
	"github.com/kyma-project/kyma-environment-broker/internal/process/input"
	"github.com/kyma-project/kyma-environment-broker/internal/process/provisioning"
	"github.com/kyma-project/kyma-environment-broker/internal/process/steps"
	"github.com/kyma-project/kyma-environment-broker/internal/provisioner"
	"github.com/kyma-project/kyma-environment-broker/internal/reconciler"
	"github.com/kyma-project/kyma-environment-broker/internal/runtimeversion"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"github.com/sirupsen/logrus"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func NewProvisioningProcessingQueue(ctx context.Context, provisionManager *process.StagedManager, workersAmount int, cfg *Config,
	db storage.BrokerStorage, provisionerClient provisioner.Client, inputFactory input.CreatorForPlan, avsDel *avs.Delegator,
	internalEvalAssistant *avs.InternalEvalAssistant, externalEvalCreator *provisioning.ExternalEvalCreator,
	runtimeVerConfigurator *runtimeversion.RuntimeVersionConfigurator,
	runtimeOverrides provisioning.RuntimeOverridesAppender, edpClient provisioning.EDPClient, accountProvider hyperscaler.AccountProvider,
	reconcilerClient reconciler.Client, k8sClientProvider provisioning.K8sClientProvider, cli client.Client, logs logrus.FieldLogger) *process.Queue {

	const postActionsStageName = "post_actions"
	provisionManager.DefineStages([]string{startStageName, createRuntimeStageName,
		checkKymaStageName, createKymaResourceStageName, postActionsStageName})
	/*
			The provisioning process contains the following stages:
			1. "start" - changes the state from pending to in progress if no deprovisioning is ongoing.
			2. "create_runtime" - collects all information needed to make an input for the Provisioner request as overrides and labels.
			Those data is collected using an InputCreator which is not persisted. That's why all steps which prepares such data must be in the same stage as "create runtime step".
		    All steps which requires InputCreator must be run in this stage.
			3. "check_kyma" - checks if the Kyma is installed
			4. "post_actions" - all steps which must be executed after the runtime is provisioned

			Once the stage is done it will never be retried.
	*/

	provisioningSteps := []struct {
		disabled  bool
		stage     string
		step      process.Step
		condition process.StepCondition
	}{
		{
			stage: startStageName,
			step:  provisioning.NewStartStep(db.Operations(), db.Instances()),
		},
		{
			stage: createRuntimeStageName,
			step:  provisioning.NewInitialisationStep(db.Operations(), db.Instances(), inputFactory, runtimeVerConfigurator),
		},
		{
			stage: createRuntimeStageName,
			step:  steps.NewInitKymaTemplate(db.Operations()),
		},
		{
			stage: createRuntimeStageName,
			step:  provisioning.NewOverrideKymaModules(db.Operations()),
		},
		{
			stage:     createRuntimeStageName,
			step:      provisioning.NewResolveCredentialsStep(db.Operations(), accountProvider),
			condition: provisioning.SkipForOwnClusterPlan,
		},
		{
			stage:    createRuntimeStageName,
			step:     provisioning.NewInternalEvaluationStep(avsDel, internalEvalAssistant),
			disabled: cfg.Avs.Disabled,
		},
		{
			stage:     createRuntimeStageName,
			step:      provisioning.NewEDPRegistrationStep(db.Operations(), edpClient, cfg.EDP),
			disabled:  cfg.EDP.Disabled,
			condition: provisioning.SkipForOwnClusterPlan,
		},
		{
			stage: createRuntimeStageName,
			step:  provisioning.NewOverridesFromSecretsAndConfigStep(db.Operations(), runtimeOverrides, runtimeVerConfigurator),
			// Preview plan does not call Reconciler so it does not need overrides
			condition: skipForPreviewPlan,
		},
		{
			condition: provisioning.WhenBTPOperatorCredentialsProvided,
			stage:     createRuntimeStageName,
			step:      provisioning.NewBTPOperatorOverridesStep(db.Operations()),
		},
		{
			condition: provisioning.SkipForOwnClusterPlan,
			stage:     createRuntimeStageName,
			step:      provisioning.NewCreateRuntimeWithoutKymaStep(db.Operations(), db.RuntimeStates(), db.Instances(), provisionerClient),
		},
		{
			condition: provisioning.DoForOwnClusterPlanOnly,
			stage:     createRuntimeStageName,
			step:      provisioning.NewCreateRuntimeForOwnClusterStep(db.Operations(), db.Instances()),
		},
		{
			stage:     createRuntimeStageName,
			step:      provisioning.NewCheckRuntimeStep(db.Operations(), provisionerClient, cfg.Provisioner.ProvisioningTimeout),
			condition: provisioning.SkipForOwnClusterPlan,
		},
		{
			stage:     createRuntimeStageName,
			disabled:  cfg.InfrastructureManagerIntegrationDisabled,
			step:      steps.NewSyncGardenerCluster(db.Operations(), cli),
			condition: provisioning.SkipForOwnClusterPlan,
		},
		{
			stage:     createRuntimeStageName,
			disabled:  cfg.InfrastructureManagerIntegrationDisabled,
			step:      steps.NewCheckGardenerCluster(db.Operations(), cli, cfg.Provisioner.GardenerClusterStepTimeout),
			condition: provisioning.SkipForOwnClusterPlan,
		},
		{ // TODO: this step must be removed when kubeconfig is created by IM only
			stage: createRuntimeStageName,
			step:  provisioning.NewGetKubeconfigStep(db.Operations(), provisionerClient),
		},
		{ // TODO: this step must be removed when kubeconfig is created by IM and own_cluster plan is permanently removed
			disabled:  cfg.LifecycleManagerIntegrationDisabled,
			stage:     createRuntimeStageName,
			step:      steps.SyncKubeconfig(db.Operations(), cli),
			condition: provisioning.DoForOwnClusterPlanOnly,
		},
		{ // must be run after the secret with kubeconfig is created ("syncKubeconfig" or "checkGardenerCluster")
			condition: provisioning.WhenBTPOperatorCredentialsProvided,
			stage:     createRuntimeStageName,
			step:      provisioning.NewInjectBTPOperatorCredentialsStep(db.Operations(), k8sClientProvider),
		},
		{
			disabled:  cfg.ReconcilerIntegrationDisabled,
			stage:     createRuntimeStageName,
			step:      provisioning.NewCreateClusterConfiguration(db.Operations(), db.RuntimeStates(), reconcilerClient),
			condition: skipForPreviewPlan,
		},
		{
			disabled:  cfg.ReconcilerIntegrationDisabled,
			stage:     checkKymaStageName,
			step:      provisioning.NewCheckClusterConfigurationStep(db.Operations(), reconcilerClient, cfg.Reconciler.ProvisioningTimeout),
			condition: skipForPreviewPlan,
		},
		{
			disabled: cfg.LifecycleManagerIntegrationDisabled,
			stage:    createKymaResourceStageName,
			step:     provisioning.NewApplyKymaStep(db.Operations(), cli),
		},
		// post actions
		{
			stage: postActionsStageName,
			step:  provisioning.NewExternalEvalStep(externalEvalCreator),
		},
		{
			stage:    postActionsStageName,
			step:     provisioning.NewInternalEvaluationStep(avsDel, internalEvalAssistant),
			disabled: cfg.Avs.Disabled,
		},
	}
	for _, step := range provisioningSteps {
		if !step.disabled {
			err := provisionManager.AddStep(step.stage, step.step, step.condition)
			if err != nil {
				fatalOnError(err, logs)
			}
		}
	}

	queue := process.NewQueue(provisionManager, logs)
	queue.Run(ctx.Done(), workersAmount)

	return queue
}
