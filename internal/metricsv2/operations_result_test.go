package metricsv2

import (
	"context"
	"math/rand"
	"testing"
	"time"
	
	"github.com/google/uuid"
	"github.com/kyma-project/kyma-environment-broker/internal"
	`github.com/kyma-project/kyma-environment-broker/internal/event`
	`github.com/kyma-project/kyma-environment-broker/internal/process`
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"github.com/pivotal-cf/brokerapi/v8/domain"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

const (
	tries = 100000
)

func TestOperationsResult(t *testing.T) {
	t.Run("1000000 metrics should be published with 1 or 0", func(t *testing.T) {
		operations := storage.NewMemoryStorage().Operations()
		for i := 0; i < tries; i++ {
			op := internal.Operation{
				ID:        uuid.New().String(),
				InstanceID: uuid.New().String(),
				ProvisioningParameters: internal.ProvisioningParameters{
					PlanID: randomPlanId(),
				},
				CreatedAt: randomCreatedAt(),
				UpdatedAt: randomUpdatedAtAfterCreatedAt(),
				Type:      randomType(),
				State:     randomState(),
			}
			err := operations.InsertOperation(op)
			assert.NoError(t, err)
		}
		
		operationResult := NewOperationResult(
			context.Background(), operations, Config{
				Enabled: true, OperationResultPoolingInterval: 1 * time.Second,
				OperationStatsPoolingInterval: 1 * time.Second, OperationResultRetentionPeriod: 24 * time.Hour,
			}, logrus.New(),
		)
		
		eventBroker := event.NewPubSub(logrus.New())
		eventBroker.Subscribe(process.OperationFinished{}, operationResult.Handler)
		
		time.Sleep(1 * time.Second)
		
		ops, err := operations.GetAllOperations()
		assert.NoError(t, err)
		assert.Equal(t, tries, len(ops))
		
		for _, op := range ops {
			assert.Equal(
				t, float64(1), testutil.ToFloat64(
					operationResult.metrics.With(getLabels(op)),
				),
			)
		}

		newOp := getRandomOp(time.Now().UTC(), domain.InProgress)
 		err = operations.InsertOperation(newOp)
		time.Sleep(1 * time.Second)
		
		assert.NoError(t, err)
		assert.Equal(t, float64(1), testutil.ToFloat64(operationResult.metrics.With(getLabels(newOp))))

		newOp.State = domain.InProgress
		newOp.UpdatedAt = time.Now().UTC().Add(1*time.Second)
		_, err = operations.UpdateOperation(newOp)
		assert.NoError(t, err)
		assert.Equal(t, float64(1), testutil.ToFloat64(operationResult.metrics.With(getLabels(newOp))))
		
		opEvent := getRandomOp(randomCreatedAt(), domain.InProgress)
		eventBroker.Publish(context.Background(), process.OperationFinished{Operation: opEvent})
		time.Sleep(1 * time.Second)
		assert.Equal(t, float64(1), testutil.ToFloat64(operationResult.metrics.With(getLabels(opEvent))))
		
		nonExistingOp1 := getRandomOp(randomCreatedAt(), domain.InProgress)
		nonExistingOp2 := getRandomOp(randomCreatedAt(), domain.Failed)
		time.Sleep(1 * time.Second)

		assert.Equal(t, float64(0), testutil.ToFloat64(operationResult.metrics.With(getLabels(nonExistingOp1))))
		assert.Equal(t, float64(0), testutil.ToFloat64(operationResult.metrics.With(getLabels(nonExistingOp2))))
		
		existingOp1 := getRandomOp(time.Now().UTC(), domain.InProgress)
		operations.InsertOperation(existingOp1)
		
		existingOp2 := getRandomOp(time.Now().UTC(), domain.Succeeded)
		operations.InsertOperation(existingOp2)
		
		existingOp3 := getRandomOp(time.Now().UTC(), domain.InProgress)
		operations.InsertOperation(existingOp3)
		
		existingOp4 := getRandomOp(time.Now().UTC(),domain.Failed)
 		operations.InsertOperation(existingOp4)
		
		time.Sleep(1 * time.Second)
		
		assert.Equal(t, float64(1), testutil.ToFloat64(operationResult.metrics.With(getLabels(existingOp1))))
		assert.Equal(t, float64(1), testutil.ToFloat64(operationResult.metrics.With(getLabels(existingOp2))))
		assert.Equal(t, float64(1), testutil.ToFloat64(operationResult.metrics.With(getLabels(existingOp4))))
		assert.Equal(t, float64(1), testutil.ToFloat64(operationResult.metrics.With(getLabels(existingOp3))))
	})
}

func getRandomOp(createdAt time.Time, state domain.LastOperationState) internal.Operation {
	return internal.Operation{
		ID:         uuid.New().String(),
		InstanceID: uuid.New().String(),
		ProvisioningParameters: internal.ProvisioningParameters{
			PlanID: randomPlanId(),
		},
		CreatedAt: createdAt,
		UpdatedAt: randomUpdatedAtAfterCreatedAt(),
		Type:      randomType(),
		State:     state,
	}
}

func randomState() domain.LastOperationState {
	return opStates[rand.Intn(len(opStates))]
}

func randomType() internal.OperationType {
	return opTypes[rand.Intn(len(opTypes))]
}

func randomPlanId() string {
	return string(plans[rand.Intn(len(plans))])
}

func randomCreatedAt() time.Time {
	return time.Now().UTC().Add(-time.Duration(rand.Intn(60)) * time.Minute)
}

func randomUpdatedAtAfterCreatedAt() time.Time {
	return randomCreatedAt().Add(time.Duration(rand.Intn(10)) * time.Minute)
}