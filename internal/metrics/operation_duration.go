package metrics

import (
	"context"
	"fmt"

	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/process"

	"github.com/pivotal-cf/brokerapi/v8/domain"
	"github.com/prometheus/client_golang/prometheus"
)

// OperationDurationCollector provides histograms which describes the time of provisioning/deprovisioning operations:
// - compass_keb_provisioning_duration_minutes
// - compass_keb_deprovisioning_duration_minutes
type OperationDurationCollector struct {
	provisioningHistogram   *prometheus.HistogramVec
	deprovisioningHistogram *prometheus.HistogramVec
}

func NewOperationDurationCollector() *OperationDurationCollector {
	return &OperationDurationCollector{
		provisioningHistogram: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: prometheusNamespace,
			Subsystem: prometheusSubsystem,
			Name:      "provisioning_duration_minutes",
			Help:      "The time of the provisioning process",
			Buckets:   prometheus.LinearBuckets(10, 2, 56),
		}, []string{"plan_id"}),
		deprovisioningHistogram: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: prometheusNamespace,
			Subsystem: prometheusSubsystem,
			Name:      "deprovisioning_duration_minutes",
			Help:      "The time of the deprovisioning process",
			Buckets:   prometheus.LinearBuckets(10, 2, 56),
		}, []string{"plan_id"}),
	}
}

func (c *OperationDurationCollector) Describe(ch chan<- *prometheus.Desc) {
	fmt.Println("Describe OperationDurationCollector called")
	c.provisioningHistogram.Describe(ch)
	c.deprovisioningHistogram.Describe(ch)
}

func (c *OperationDurationCollector) Collect(ch chan<- prometheus.Metric) {
	fmt.Println("Collector OperationDurationCollector called")
	c.provisioningHistogram.Collect(ch)
	c.deprovisioningHistogram.Collect(ch)
}

func (c *OperationDurationCollector) OnProvisioningSucceeded(ctx context.Context, ev interface{}) error {
	provision, ok := ev.(process.ProvisioningSucceeded)
	if !ok {
		return fmt.Errorf("expected process.ProvisioningSucceeded but got %+v", ev)
	}

	op := provision.Operation
	pp := op.ProvisioningParameters
	minutes := op.UpdatedAt.Sub(op.CreatedAt).Minutes()
	c.provisioningHistogram.
		WithLabelValues(pp.PlanID).Observe(minutes)

	return nil
}

func (c *OperationDurationCollector) OnDeprovisioningStepProcessed(ctx context.Context, ev interface{}) error {
	stepProcessed, ok := ev.(process.DeprovisioningStepProcessed)
	if !ok {
		return fmt.Errorf("expected process.DeprovisioningStepProcessed but got %+v", ev)
	}

	op := stepProcessed.Operation
	pp := op.ProvisioningParameters
	if stepProcessed.OldOperation.State == domain.InProgress && op.State == domain.Succeeded {
		minutes := op.UpdatedAt.Sub(op.CreatedAt).Minutes()
		c.deprovisioningHistogram.
			WithLabelValues(pp.PlanID).Observe(minutes)
	}

	return nil
}

func (c *OperationDurationCollector) OnOperationStepProcessed(ctx context.Context, ev interface{}) error {
	stepProcessed, ok := ev.(process.OperationStepProcessed)
	if !ok {
		return fmt.Errorf("expected process.OperationStepProcessed in OnOperationStepProcessed but got %+v", ev)
	}

	if stepProcessed.Operation.Type == internal.OperationTypeDeprovision {
		dsp := process.DeprovisioningStepProcessed{
			StepProcessed: stepProcessed.StepProcessed,
			OldOperation:  internal.DeprovisioningOperation{Operation: stepProcessed.OldOperation},
			Operation:     internal.DeprovisioningOperation{Operation: stepProcessed.Operation},
		}
		err := c.OnDeprovisioningStepProcessed(ctx, dsp)
		if err != nil {
			return fmt.Errorf("failed to handle OnDeprovisioningStepProcessed in OnOperationStepProcessed: %w", err)
		}
	}
	return nil
}

func (c *OperationDurationCollector) OnOperationSucceeded(ctx context.Context, ev interface{}) error {
	operationSucceeded, ok := ev.(process.OperationSucceeded)
	if !ok {
		return fmt.Errorf("expected OperationSucceeded but got %+v", ev)
	}

	switch operationSucceeded.Operation.Type {
	case internal.OperationTypeProvision:
		provisioningOperation := process.ProvisioningSucceeded{
			Operation: internal.ProvisioningOperation{Operation: operationSucceeded.Operation},
		}
		err := c.OnProvisioningSucceeded(ctx, provisioningOperation)
		if err != nil {
			return err
		}
	case internal.OperationTypeDeprovision:
		op := operationSucceeded.Operation
		pp := operationSucceeded.Operation.ProvisioningParameters
		minutes := op.UpdatedAt.Sub(op.CreatedAt).Minutes()
		c.deprovisioningHistogram.WithLabelValues(pp.PlanID).Observe(minutes)
	default:
		return fmt.Errorf("unsupported OperationStep %+v for OnOperationSucceeded handler", operationSucceeded.Operation.Type)
	}

	return nil
}
