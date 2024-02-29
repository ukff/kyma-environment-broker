package metricsv2

import (
	"context"
	"sync"
	"testing"

	"github.com/kyma-project/kyma-environment-broker/common/orchestration"
	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/kyma-environment-broker/internal/process"
	"github.com/pivotal-cf/brokerapi/v8/domain"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestOperationsCounter(t *testing.T) {
	var ctr *operationsCounter

	opType := internal.OperationTypeProvision
	opState := domain.Succeeded
	opPlan := broker.AzurePlanID
	eventsCount1 := 5
	key1 := ctr.buildKeyFor(opType, opState, broker.PlanID(opPlan))

	opType2 := internal.OperationTypeUpdate
	opState2 := domain.InProgress
	opPlan2 := broker.AWSPlanID
	key2 := ctr.buildKeyFor(opType2, opState2, broker.PlanID(opPlan2))
	eventsCount2 := 1

	opType3 := internal.OperationTypeDeprovision
	opState3 := domain.Failed
	opPlan3 := broker.GCPPlanID
	eventsCount3 := 3
	key3 := ctr.buildKeyFor(opType3, opState3, broker.PlanID(opPlan3))

	opType4 := internal.OperationTypeDeprovision
	opState4 := domain.InProgress
	opPlan4 := broker.GCPPlanID
	eventsCount4 := 0
	key4 := ctr.buildKeyFor(opType4, opState4, broker.PlanID(opPlan4))

	t.Run("create counter key", func(t *testing.T) {
		ctr = NewOperationsCounters(logrus.WithField("metrics", "test"))
	})

	t.Run("should increase counter", func(t *testing.T) {
		t.Run("should increase counter", func(t *testing.T) {
			t.Parallel()
			wg := sync.WaitGroup{}
			for i := 0; i < eventsCount1; i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					ctr.handler(context.TODO(), process.ProvisioningFinished{Operation: internal.ProvisioningOperation{
						Operation: internal.Operation{
							ID:    "test",
							State: opState,
							Type:  opType,
							RuntimeOperation: orchestration.RuntimeOperation{
								Runtime: orchestration.Runtime{
									Plan: opPlan,
								},
							},
						},
					}})
				}()
			}
			wg.Wait()
		})

		t.Run("should increase counter", func(t *testing.T) {
			t.Parallel()
			ctr.handler(context.TODO(), process.UpdateFinished{Operation: internal.UpdatingOperation{
				Operation: internal.Operation{
					ID:    "test",
					State: opState2,
					Type:  opType2,
					RuntimeOperation: orchestration.RuntimeOperation{
						Runtime: orchestration.Runtime{
							Plan: opPlan2,
						},
					},
				},
			}})
		})

		t.Run("should increase counter", func(t *testing.T) {
			t.Parallel()
			wg := sync.WaitGroup{}
			for i := 0; i < eventsCount3; i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					ctr.handler(context.TODO(), process.DeprovisioningFinished{Operation: internal.DeprovisioningOperation{
						Operation: internal.Operation{
							ID:    "test",
							State: opState3,
							Type:  opType3,
							RuntimeOperation: orchestration.RuntimeOperation{
								Runtime: orchestration.Runtime{
									Plan: opPlan3,
								},
							},
						},
					}})
				}()
			}
			wg.Wait()
		})
	})

	t.Run("should get correct number of metrics", func(t *testing.T) {
		assert.Equal(t, float64(eventsCount1), testutil.ToFloat64(ctr.metrics[key1]))
		assert.Equal(t, float64(eventsCount2), testutil.ToFloat64(ctr.metrics[key2]))
		assert.Equal(t, float64(eventsCount3), testutil.ToFloat64(ctr.metrics[key3]))
		assert.Equal(t, float64(eventsCount4), testutil.ToFloat64(ctr.metrics[key4]))
	})

	t.Cleanup(func() {
		ctr = nil
	})
}
