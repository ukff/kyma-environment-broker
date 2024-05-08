package process

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/kyma-project/kyma-environment-broker/internal/broker"

	"github.com/kyma-project/kyma-environment-broker/internal/storage/dberr"

	"github.com/pkg/errors"

	"github.com/kyma-project/kyma-environment-broker/internal"
	kebError "github.com/kyma-project/kyma-environment-broker/internal/error"
	"github.com/kyma-project/kyma-environment-broker/internal/event"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"

	"github.com/pivotal-cf/brokerapi/v8/domain"
	"github.com/sirupsen/logrus"
)

type StagedManager struct {
	log              logrus.FieldLogger
	operationStorage storage.Operations
	publisher        event.Publisher

	stages           []*stage
	operationTimeout time.Duration

	mu sync.RWMutex

	speedFactor int64
	cfg         StagedManagerConfiguration
}

type StagedManagerConfiguration struct {
	// Max time of processing step by a worker without returning to the queue
	MaxStepProcessingTime time.Duration `envconfig:"default=2m"`
	WorkersAmount         int           `envconfig:"default=20"`
}

func (c StagedManagerConfiguration) String() string {
	return fmt.Sprintf("(MaxStepProcessingTime=%s; WorkersAmount=%d)", c.MaxStepProcessingTime, c.WorkersAmount)
}

type Step interface {
	Name() string
	Run(operation internal.Operation, logger logrus.FieldLogger) (internal.Operation, time.Duration, error)
}

type StepCondition func(operation internal.Operation) bool

type StepWithCondition struct {
	Step
	condition StepCondition
}

type stage struct {
	name  string
	steps []StepWithCondition
}

func (s *stage) AddStep(step Step, cnd StepCondition) {
	s.steps = append(s.steps, StepWithCondition{
		Step:      step,
		condition: cnd,
	})
}

func NewStagedManager(storage storage.Operations, pub event.Publisher, operationTimeout time.Duration, cfg StagedManagerConfiguration, logger logrus.FieldLogger) *StagedManager {
	return &StagedManager{
		log:              logger,
		operationStorage: storage,
		publisher:        pub,
		operationTimeout: operationTimeout,
		speedFactor:      1,
		cfg:              cfg,
	}
}

// SpeedUp changes speedFactor parameter to reduce the sleep time if a step needs a retry.
// This method should only be used for testing purposes
func (m *StagedManager) SpeedUp(speedFactor int64) {
	m.speedFactor = speedFactor
}

func (m *StagedManager) DefineStages(names []string) {
	m.stages = make([]*stage, len(names))
	for i, n := range names {
		m.stages[i] = &stage{name: n, steps: []StepWithCondition{}}
	}
}

func (m *StagedManager) AddStep(stageName string, step Step, cnd StepCondition) error {
	for _, s := range m.stages {
		if s.name == stageName {
			s.AddStep(step, cnd)
			return nil
		}
	}
	return fmt.Errorf("stage %s not defined", stageName)
}

func (m *StagedManager) GetAllStages() []string {
	var all []string
	for _, s := range m.stages {
		all = append(all, s.name)
	}
	return all
}

func (m *StagedManager) Execute(operationID string) (time.Duration, error) {

	operation, err := m.operationStorage.GetOperationByID(operationID)
	if err != nil {
		m.log.Errorf("Cannot fetch operation from storage: %s", err)
		return 3 * time.Second, nil
	}

	logOperation := m.log.WithFields(logrus.Fields{"operation": operationID, "instanceID": operation.InstanceID, "planID": operation.ProvisioningParameters.PlanID})
	logOperation.Infof("Start process operation steps for GlobalAccount=%s, ", operation.ProvisioningParameters.ErsContext.GlobalAccountID)
	if time.Since(operation.CreatedAt) > m.operationTimeout {
		timeoutErr := kebError.TimeoutError("operation has reached the time limit")
		operation.LastError = timeoutErr
		defer m.publishEventOnFail(operation, err)
		logOperation.Infof("operation has reached the time limit: operation was created at: %s", operation.CreatedAt)
		operation.State = domain.Failed
		_, err = m.operationStorage.UpdateOperation(*operation)
		if err != nil {
			logOperation.Infof("Unable to save operation with finished the provisioning process")
			timeoutErr = timeoutErr.SetMessage(fmt.Sprintf("%s and %s", timeoutErr.Error(), err.Error()))
			operation.LastError = timeoutErr
			return time.Second, timeoutErr
		}

		return 0, timeoutErr
	}

	var when time.Duration
	processedOperation := *operation

	for _, stage := range m.stages {
		if processedOperation.IsStageFinished(stage.name) {
			continue
		}

		for _, step := range stage.steps {
			logStep := logOperation.WithField("step", step.Name()).
				WithField("stage", stage.name)
			if step.condition != nil && !step.condition(processedOperation) {
				logStep.Debugf("Skipping")
				continue
			}
			operation.EventInfof("processing step: %v", step.Name())

			processedOperation, when, err = m.runStep(step, processedOperation, logStep)
			if err != nil {
				logStep.Errorf("Process operation failed: %s", err)
				operation.EventErrorf(err, "step %v processing returned error", step.Name())
				return 0, err
			}
			if processedOperation.State == domain.Failed || processedOperation.State == domain.Succeeded {
				logStep.Infof("Operation %q got status %s. Process finished.", operation.ID, processedOperation.State)
				operation.EventInfof("operation processing %v", processedOperation.State)
				m.publishOperationFinishedEvent(processedOperation)
				m.publishDeprovisioningSucceeded(&processedOperation)
				return 0, nil
			}

			// the step needs a retry
			if when > 0 {
				logStep.Warnf("retrying step by restarting the operation in %d s", int64(when.Seconds()))
				return when, nil
			}
		}

		processedOperation, err = m.saveFinishedStage(processedOperation, stage, logOperation)

		// it is ok, when operation deos not exists in the DB - it can happen at the end of a deprovisioning process
		if err != nil && !dberr.IsNotFound(err) {
			return time.Second, nil
		}
	}

	logOperation.Infof("Operation succeeded")

	processedOperation.State = domain.Succeeded
	processedOperation.Description = "Processing finished"

	m.publishEventOnSuccess(&processedOperation)

	_, err = m.operationStorage.UpdateOperation(processedOperation)
	// it is ok, when operation deos not exists in the DB - it can happen at the end of a deprovisioning process
	if err != nil && !dberr.IsNotFound(err) {
		logOperation.Infof("Unable to save operation with finished the provisioning process")
		return time.Second, err
	}

	return 0, nil
}

func (m *StagedManager) saveFinishedStage(operation internal.Operation, s *stage, log logrus.FieldLogger) (internal.Operation, error) {
	operation.FinishStage(s.name)
	op, err := m.operationStorage.UpdateOperation(operation)
	// it is ok, when operation deos not exists in the DB - it can happen at the end of a deprovisioning process
	if err != nil && !dberr.IsNotFound(err) {
		log.Infof("Unable to save operation with finished stage %s: %s", s.name, err.Error())
		return operation, err
	}
	log.Infof("Finished stage %s", s.name)
	return *op, nil
}

func (m *StagedManager) runStep(step Step, operation internal.Operation, logger logrus.FieldLogger) (processedOperation internal.Operation, backoff time.Duration, err error) {
	var start time.Time
	defer func() {
		if pErr := recover(); pErr != nil {
			logger.Println("panic in RunStep in staged manager: ", pErr)
			err = errors.New(fmt.Sprintf("%v", pErr))
			om := NewOperationManager(m.operationStorage)
			processedOperation, _, _ = om.OperationFailed(operation, "recovered from panic", err, m.log)
		}
	}()

	processedOperation = operation
	begin := time.Now()
	for {
		start = time.Now()
		logger.Infof("Start step")
		stepLogger := logger.WithFields(logrus.Fields{"step": step.Name(), "operation": processedOperation.ID})
		processedOperation, backoff, err = step.Run(processedOperation, stepLogger)
		if err != nil {
			processedOperation.LastError = kebError.ReasonForError(err)
			logOperation := stepLogger.WithFields(logrus.Fields{"error_component": processedOperation.LastError.Component(), "error_reason": processedOperation.LastError.Reason()})
			logOperation.Warnf("Last error from step: %s", processedOperation.LastError.Error())
			// only save to storage, skip for alerting if error
			_, err = m.operationStorage.UpdateOperation(processedOperation)
			if err != nil {
				logOperation.Errorf("unable to save operation with resolved last error from step, additionally, see previous logs for ealier errors")
			}
		}

		m.publisher.Publish(context.TODO(), OperationStepProcessed{
			StepProcessed: StepProcessed{
				StepName: step.Name(),
				Duration: time.Since(start),
				When:     backoff,
				Error:    err,
			},
			Operation:    processedOperation,
			OldOperation: operation,
		})

		// break the loop if:
		// - the step does not need a retry
		// - step returns an error
		// - the loop takes too much time (to not block the worker too long)
		if backoff == 0 || err != nil || time.Since(begin) > m.cfg.MaxStepProcessingTime {
			if err != nil {
				logOperation := m.log.WithFields(logrus.Fields{"step": step.Name(), "operation": processedOperation.ID, "error_component": processedOperation.LastError.Component(), "error_reason": processedOperation.LastError.Reason()})
				logOperation.Errorf("Last Error that terminated the step: %s", processedOperation.LastError.Error())
			}
			return processedOperation, backoff, err
		}
		operation.EventInfof("step %v sleeping for %v", step.Name(), backoff)
		time.Sleep(backoff / time.Duration(m.speedFactor))
	}
}

func (m *StagedManager) publishEventOnFail(operation *internal.Operation, err error) {
	logOperation := m.log.WithFields(logrus.Fields{"operation": operation.ID, "error_component": operation.LastError.Component(), "error_reason": operation.LastError.Reason()})
	logOperation.Errorf("Last error: %s", operation.LastError.Error())

	m.publishOperationFinishedEvent(*operation)

	m.publisher.Publish(context.TODO(), OperationStepProcessed{
		StepProcessed: StepProcessed{
			Duration: time.Since(operation.CreatedAt),
			Error:    err,
		},
		OldOperation: *operation,
		Operation:    *operation,
	})
}

func (m *StagedManager) publishEventOnSuccess(operation *internal.Operation) {
	m.publisher.Publish(context.TODO(), OperationSucceeded{
		Operation: *operation,
	})

	m.publishOperationFinishedEvent(*operation)

	m.publishDeprovisioningSucceeded(operation)
}

func (m *StagedManager) publishOperationFinishedEvent(operation internal.Operation) {
	m.publisher.Publish(context.TODO(), OperationFinished{
		Operation: operation,
		PlanID:    broker.PlanID(operation.ProvisioningParameters.PlanID),
	})
}

func (m *StagedManager) publishDeprovisioningSucceeded(operation *internal.Operation) {
	if operation.State == domain.Succeeded && operation.Type == internal.OperationTypeDeprovision {
		m.publisher.Publish(
			context.TODO(), DeprovisioningSucceeded{
				Operation: internal.DeprovisioningOperation{Operation: *operation},
			},
		)
	}
}
