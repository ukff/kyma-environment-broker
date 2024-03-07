package metricsv2

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/kyma-environment-broker/internal/process"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"github.com/pivotal-cf/brokerapi/v8/domain"
	"github.com/prometheus/client_golang/prometheus/testutil"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestOperationsCounter(t *testing.T) {
	var ctr *operationStats

	opType1 := internal.OperationTypeProvision
	opState1 := domain.Succeeded
	opPlan1 := broker.AzurePlanID
	eventsCount1 := 5
	key1, err := ctr.buildKeyFor(opType1, opState1, broker.PlanID(opPlan1))
	assert.NoError(t, err)

	opType2 := internal.OperationTypeUpdate
	opState2 := domain.Failed
	opPlan2 := broker.AWSPlanID
	key2, err := ctr.buildKeyFor(opType2, opState2, broker.PlanID(opPlan2))
	eventsCount2 := 1
	assert.NoError(t, err)

	opType3 := internal.OperationTypeDeprovision
	opState3 := domain.Failed
	opPlan3 := broker.GCPPlanID
	eventsCount3 := 3
	key3, err := ctr.buildKeyFor(opType3, opState3, broker.PlanID(opPlan3))
	assert.NoError(t, err)

	opType4 := internal.OperationTypeDeprovision
	opState4 := domain.InProgress
	opPlan4 := broker.GCPPlanID
	key4, err := ctr.buildKeyFor(opType4, opState4, broker.PlanID(opPlan4))
	assert.NoError(t, err)

	operations := storage.NewMemoryStorage().Operations()
	opType5 := internal.OperationTypeProvision
	opState5 := domain.InProgress
	opPlan5 := broker.AzurePlanID
	key5, err := ctr.buildKeyFor(opType5, opState5, broker.PlanID(opPlan5))
	assert.NoError(t, err)

	t.Run("create counter key", func(t *testing.T) {
		ctr, err = NewOperationsCounters(context.TODO(), operations, 1*time.Millisecond, log.WithField("metrics", "test"))
		assert.NoError(t, err)
		//ctr.MustRegister()
	})

	t.Run("gauge in_progress operations test", func(t *testing.T) {
		op := internal.Operation{
			State: opState5,
			Type:  opType5,
			ProvisioningParameters: internal.ProvisioningParameters{
				PlanID: opPlan5,
			},
		}
		err := operations.InsertOperation(op)
		assert.NoError(t, err)
	})

	t.Run("should increase all counter", func(t *testing.T) {
		t.Run("should increase counter", func(t *testing.T) {
			t.Parallel()
			wg := sync.WaitGroup{}
			for i := 0; i < eventsCount1; i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					err := ctr.Handler(context.TODO(), process.OperationCounting{
						OpId:    "test1",
						PlanID:  opPlan1,
						OpState: string(opState1),
						OpType:  string(opType1),
					})
					assert.NoError(t, err)
				}()
			}
			wg.Wait()
		})

		t.Run("should increase counter", func(t *testing.T) {
			t.Parallel()
			wg := sync.WaitGroup{}
			for i := 0; i < eventsCount2; i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					err := ctr.Handler(context.TODO(), process.OperationCounting{
						OpId:    "test2",
						PlanID:  opPlan2,
						OpState: string(opState2),
						OpType:  string(opType2),
					})
					assert.NoError(t, err)
				}()
			}
			wg.Wait()
		})

		t.Run("should increase counter", func(t *testing.T) {
			t.Parallel()
			wg := sync.WaitGroup{}
			for i := 0; i < eventsCount3; i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					err := ctr.Handler(context.TODO(), process.OperationCounting{
						OpId:    "test3",
						PlanID:  opPlan3,
						OpState: string(opState3),
						OpType:  string(opType3),
					})
					assert.NoError(t, err)
				}()
			}
			wg.Wait()
		})
	})

	t.Run("should get correct number of metrics", func(t *testing.T) {
		time.Sleep(1 * time.Second)
		assert.Equal(t, float64(eventsCount1), testutil.ToFloat64(ctr.counters[key1]))
		assert.Equal(t, float64(eventsCount2), testutil.ToFloat64(ctr.counters[key2]))
		assert.Equal(t, float64(eventsCount3), testutil.ToFloat64(ctr.counters[key3]))
		assert.Equal(t, float64(0), testutil.ToFloat64(ctr.gauges[key4]))
		assert.True(t, testutil.ToFloat64(ctr.gauges[key5]) > float64(0))
	})

	t.Cleanup(func() {
		ctr = nil
	})
}
