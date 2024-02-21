package metrics

import (
	"fmt"

	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

// InstancesStatsGetter provides number of all instances failed, succeeded or orphaned
//
//	(instance exists but the cluster was removed manually from the gardener):
//
// - compass_keb_instances_total - total number of all instances
// - compass_keb_global_account_id_instances_total - total number of all instances per global account
// - compass_keb_ers_context_license_type_total - count of instances grouped by license types

type InstancesStatsGetter interface {
	GetInstanceStats() (internal.InstanceStats, error)
	GetERSContextStats() (internal.ERSContextStats, error)
}

type InstancesCollector struct {
	statsGetter InstancesStatsGetter

	instancesDesc        *prometheus.Desc
	instancesPerGAIDDesc *prometheus.Desc
	licenseTypeDesc      *prometheus.Desc
}

func NewInstancesCollector(statsGetter InstancesStatsGetter) *InstancesCollector {
	return &InstancesCollector{
		statsGetter: statsGetter,

		instancesDesc: prometheus.NewDesc(
			prometheus.BuildFQName(prometheusNamespace, prometheusSubsystem, "instances_total"),
			"The total number of instances",
			[]string{},
			nil),
		instancesPerGAIDDesc: prometheus.NewDesc(
			prometheus.BuildFQName(prometheusNamespace, prometheusSubsystem, "global_account_id_instances_total"),
			"The total number of instances by Global Account ID",
			[]string{"global_account_id"},
			nil),
		licenseTypeDesc: prometheus.NewDesc(
			prometheus.BuildFQName(prometheusNamespace, prometheusSubsystem, "ers_context_license_type_total"),
			"count of instances grouped by license types",
			[]string{"license_type"},
			nil),
	}
}

func (c *InstancesCollector) Describe(ch chan<- *prometheus.Desc) {
	fmt.Println("Describe InstancesCollector called")
	ch <- c.instancesDesc
	ch <- c.instancesPerGAIDDesc
	ch <- c.licenseTypeDesc
}

// Collect implements the prometheus.Collector interface.
func (c *InstancesCollector) Collect(prometheusChannel chan<- prometheus.Metric) {
	fmt.Println("Collector InstancesCollector called")
	// SQL CALL
	instanceStats, err := c.statsGetter.GetInstanceStats()
	if err != nil {
		logrus.Error(err)
	} else {
		collect(prometheusChannel, c.instancesDesc, instanceStats.TotalNumberOfInstances)

		for globalAccountID, value := range instanceStats.PerGlobalAccountID {
			collect(prometheusChannel, c.instancesPerGAIDDesc, value, globalAccountID)
		}
	}

	// SQL CALL
	ersStats, err := c.statsGetter.GetERSContextStats()
	if err != nil {
		logrus.Error(err)
		return
	}
	for key, value := range ersStats.LicenseType {
		collect(prometheusChannel, c.licenseTypeDesc, value, key)
	}
}
