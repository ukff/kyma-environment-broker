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
	tries = 5
)

func TestOperationsResult(t *testing.T) {
	operations := storage.NewMemoryStorage().Operations()
	operationResult := NewOperationResult(
		context.Background(), operations, Config{
			Enabled: true, OperationResultPoolingInterval: 100 * time.Minute,
			OperationStatsPoolingInterval: 100 * time.Minute, OperationResultRetentionPeriod: 24 * time.Hour,
		}, logrus.New(),
	)
	t.Run("1000 ops", func(t *testing.T) {
		var ops []internal.Operation
		var iids []string
		for i := 0; i < tries; i++ {
			iid := uuid.New().String()
			iids = append(iids, iid)
			o := internal.Operation{
				ID:        iid,
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
			assert.NoError(t, err)
			ops = append(ops, o)
		}

		// wait for job on start to finish
		for operationResult.jobRunning{}

		// all ops should be processed and published with 1
		for _, op := range ops {
			assert.Equal(
				t, float64(1), testutil.ToFloat64(
					operationResult.metrics.With(getLabels(op)),
				),
			)
		}

		// job seeking now in time windows

		// simulate new op
		newOp := getRandomOp(domain.InProgress)
		err := operations.InsertOperation(newOp)

		// wait for job
		for operationResult.jobRunning {}

		assert.NoError(t, err)
		assert.Equal(t, float64(1), testutil.ToFloat64(operationResult.metrics.With(getLabels(newOp))))

		// simulate new op updated
		newOp.State = domain.InProgress
		newOp.UpdatedAt = time.Now().UTC().Add(1*time.Second)
		_, err = operations.UpdateOperation(newOp)
		assert.NoError(t, err)
		
		
		nonExistingOp1 := getRandomOp(domain.InProgress)
		getLabels(nonExistingOp1)
		nonExistingOp2 := getRandomOp(domain.Failed)
		getLabels(nonExistingOp2)

		// wait for job
		for operationResult.jobRunning {}
		assert.Equal(t, float64(0), testutil.ToFloat64(operationResult.metrics.With(getLabels(nonExistingOp1))))
		assert.Equal(t, float64(0), testutil.ToFloat64(operationResult.metrics.With(getLabels(nonExistingOp2))))
		
		existingOp1 := getRandomOp(domain.InProgress)
		getLabels(existingOp1)
		existingOp2 := getRandomOp(domain.Succeeded)
		getLabels(existingOp2)
		
		existingOp4 := getRandomOp(domain.InProgress)
		getLabels(existingOp4)
		existingOp3 := getRandomOp(domain.Failed)
		getLabels(existingOp3)
		
		// wait for job
		for operationResult.jobRunning {}
		assert.Equal(t, float64(1), testutil.ToFloat64(operationResult.metrics.With(getLabels(existingOp1))))
		assert.Equal(t, float64(1), testutil.ToFloat64(operationResult.metrics.With(getLabels(existingOp2))))
		assert.Equal(t, float64(1), testutil.ToFloat64(operationResult.metrics.With(getLabels(existingOp3))))
		assert.Equal(t, float64(1), testutil.ToFloat64(operationResult.metrics.With(getLabels(existingOp3))))
	})
}

func getRandomOp(op domain.LastOperationState) internal.Operation {
	return internal.Operation{
		ID:         uuid.New().String(),
		InstanceID: uuid.New().String(),
		ProvisioningParameters: internal.ProvisioningParameters{
			PlanID: randomPlanId(),
		},
		CreatedAt: randomCreatedAt(),
		UpdatedAt: randomUpdatedAtAfterCreatedAt(),
		Type:      randomType(),
		State:     op,
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