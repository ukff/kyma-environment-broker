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
	for i := 0; i < 1000; i++ {
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

	operationResult := NewOperationResult(context.Background(), operations, Config{Enabled: true, OperationResultPoolingInterval: 1 * time.Second, OperationStatsPoolingInterval: 1 * time.Second, OperationResultRetentionPeriod: 1 * time.Hour}, logrus.New())

	time.Sleep(15 * time.Second)

	for _, op := range ops {
		l := getLabels(op)
		assert.Equal(
			t, float64(1), testutil.ToFloat64(
				operationResult.metrics.With(l),
			))
	}

	newOp := internal.Operation{
		ID:         uuid.New().String(),
		InstanceID: uuid.New().String(),
		ProvisioningParameters: internal.ProvisioningParameters{
			PlanID: randomPlanId(),
		},
		CreatedAt: time.Now().UTC(),
		Type:      randomType(),
		State:     randomState(),
	}
	err := operations.InsertOperation(newOp)
	time.Sleep(15 * time.Second)
	assert.NoError(t, err)
	assert.Equal(t, float64(1), testutil.ToFloat64(operationResult.metrics.With(getLabels(newOp))))
	newOp.State = domain.Failed
	newOp.UpdatedAt = time.Now().UTC()
	_, err = operations.UpdateOperation(newOp)
	assert.NoError(t, err)
	time.Sleep(15 * time.Second)
	assert.Equal(t, float64(1), testutil.ToFloat64(operationResult.metrics.With(getLabels(newOp))))
}
