package deprovisioning

import (
	"time"

	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/archive"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"github.com/kyma-project/kyma-environment-broker/internal/storage/dberr"
	"github.com/sirupsen/logrus"
)

const (
	dbRetryBackoff = 2 * time.Second
)

type ArchivingStep struct {
	operations        storage.Operations
	instances         storage.Instances
	instancesArchived storage.InstancesArchived
	dryRun            bool
}

func NewArchivingStep(operations storage.Operations, instances storage.Instances, archived storage.InstancesArchived, dryRun bool) *ArchivingStep {
	return &ArchivingStep{
		operations:        operations,
		instances:         instances,
		instancesArchived: archived,
		dryRun:            dryRun,
	}
}

func (s *ArchivingStep) Name() string {
	return "Archiving"
}

func (s *ArchivingStep) Run(operation internal.Operation, logger logrus.FieldLogger) (internal.Operation, time.Duration, error) {
	if operation.Temporary {
		logger.Infof("suspension operation, skipping")
		return operation, 0, nil
	}
	if operation.ExcutedButNotCompleted != nil && len(operation.ExcutedButNotCompleted) > 0 {
		logger.Infof("operation needs a retrigger, skipping")
		return operation, 0, nil
	}

	// check if the archived instance already exists
	_, err := s.instancesArchived.GetByInstanceID(operation.InstanceID)
	switch {
	case err == nil:
		logger.Warn("archived instance already existis, skipping")
		return operation, 0, nil
	case !dberr.IsNotFound(err):
		logger.Error("Unable to get archived instance")
		return operation, dbRetryBackoff, nil
	}

	instance, err := s.instances.GetByID(operation.InstanceID)
	switch {
	case dberr.IsNotFound(err):
		logger.Error("Instance not found")
	case err != nil:
		return operation, dbRetryBackoff, nil
	}

	operations, err := s.operations.ListOperationsByInstanceID(operation.InstanceID)
	if err != nil {
		logger.Error("unable to get operations for given instance")
		return operation, dbRetryBackoff, nil
	}

	var archived internal.InstanceArchived
	if instance != nil {
		logger.Infof("Creating instance archived from the instance and %d operations", len(operations))
		archived, err = archive.NewInstanceArchivedFromOperationsAndInstance(*instance, operations)
		if err != nil {
			logger.Errorf("Unable to create instance archived: %s", err.Error())
			return operation, 0, nil
		}
	} else {
		logger.Infof("Creating instance archived from %d operations", len(operations))
		archived, err = archive.NewInstanceArchivedFromOperations(operations)
		if err != nil {
			logger.Errorf("Unable to create instance archived: %s", err.Error())
			return operation, 0, nil
		}
	}

	if s.dryRun {
		logger.Infof("DryRun enabled, skipping insert of archived instance")
		logger.Infof("Archived instance: %+v", archived)
		return operation, 0, nil
	}

	err = s.instancesArchived.Insert(archived)
	if err != nil {
		logger.Errorf("unable to insert archived instance: %s", err.Error())
		return operation, dbRetryBackoff, nil
	}

	return operation, 0, nil
}
