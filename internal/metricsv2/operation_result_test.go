package metricsv2

import (
	"context"
	"math/rand"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"github.com/pivotal-cf/brokerapi/v8/domain"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

const (
	tries = 1000000
)

func randomState() domain.LastOperationState {
	return domain.LastOperationState(opStates[rand.Intn(len(opStates))])
}

func randomType() internal.OperationType {
	return opTypes[rand.Intn(len(opTypes))]
}

func randomPlanId() string {
	return string(plans[rand.Intn(len(plans))])
}

func randomCreatedAt() time.Time {
	return time.Now().Add(-time.Duration(rand.Intn(1)) * time.Hour)
}

func randomUpdatedAtAfterCreatedAt() time.Time {
	return randomCreatedAt().Add(time.Duration(rand.Intn(10)) * time.Minute)
}


func TestOperationsResult(t *testing.T) {
	var ops []internal.Operation
	operations := storage.NewMemoryStorage().Operations()
	operationResult := NewOperationResult(
		context.Background(), operations, Config{
			Enabled: true, OperationResultPoolingInterval: 1 * time.Second,
			OperationStatsPoolingInterval: 1 * time.Second, OperationResultRetentionPeriod: 1 * time.Hour,
		}, logrus.New(),
	)
	
	t.Run("1000 ops", func(t *testing.T) {
			for i := 0; i < tries; i++ {
				o := internal.Operation{
					ID:         uuid.New().String(),
					InstanceID: uuid.New().String(),
					ProvisioningParameters: internal.ProvisioningParameters{
						PlanID: randomPlanId(),
					},
					CreatedAt: randomCreatedAt(),
					UpdatedAt: randomUpdatedAtAfterCreatedAt(),
					Type:      randomType(),
					State:     randomState(),
				}
				err := operations.InsertOperation(o)
				ops = append(ops, o)
				assert.NoError(t, err)
			}
			
			// wait for job on start to finish
			time.Sleep(100 * time.Second)
			
			// all ops should be proccessed and published with 1
			for _, op := range ops {
				assert.Equal(
					t, float64(1), testutil.ToFloat64(
						operationResult.metrics.With(getLabels(op)),
					),
				)
			}
			
			// job working in time windows
			
			// simulate new op
			newOp := getRandomOp()
			err := operations.InsertOperation(newOp)
			
			// wait for job
			time.Sleep(1 * time.Second)
			
			assert.NoError(t, err)
			assert.Equal(t, float64(1), testutil.ToFloat64(operationResult.metrics.With(getLabels(newOp))))
			
			// simulate new op updated
			newOp.State = randomState()
			newOp.UpdatedAt = time.Now().UTC()
			_, err = operations.UpdateOperation(newOp)
			assert.NoError(t, err)
			nonExistingOp1 := getRandomOp()
			getLabels(nonExistingOp1)
			nonExistingOp2 := getRandomOp()
			getLabels(nonExistingOp2)
			
			// wait for job
			time.Sleep(1 * time.Second)
			assert.Equal(t, float64(1), testutil.ToFloat64(operationResult.metrics.With(getLabels(newOp))))
			assert.Equal(t, float64(0), testutil.ToFloat64(operationResult.metrics.With(getLabels(nonExistingOp2))))
	})
}


func getRandomOp() internal.Operation {
	return internal.Operation{
		ID:         uuid.New().String(),
		InstanceID: uuid.New().String(),
		ProvisioningParameters: internal.ProvisioningParameters{
			PlanID: randomPlanId(),
		},
		CreatedAt: randomCreatedAt(),
		UpdatedAt: randomUpdatedAtAfterCreatedAt(),
		Type:      randomType(),
		State:     randomState(),
	}
}