package kymacustomresource

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/kyma-project/kyma-environment-broker/internal/syncqueues"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
)

const (
	namespace               = "kcp-system"
	subaccountIdLabelKey    = "kyma-project.io/subaccount-id"
	subaccountIdLabelFormat = "kyma-project.io/subaccount-id=%s"
	k8sRequestInterval      = 5 * time.Second
)

type Updater struct {
	k8sClient     dynamic.Interface
	queue         syncqueues.MultiConsumerPriorityQueue
	kymaGVR       schema.GroupVersionResource
	sleepDuration time.Duration
	labelKey      string
	ctx           context.Context
	logger        *slog.Logger
}

func NewUpdater(k8sClient dynamic.Interface,
	queue syncqueues.MultiConsumerPriorityQueue,
	gvr schema.GroupVersionResource,
	sleepDuration time.Duration,
	labelKey string,
	ctx context.Context,
	logger *slog.Logger) (*Updater, error) {

	logger.Info(fmt.Sprintf("Creating Kyma CR updater for label: %s", labelKey))

	return &Updater{
		k8sClient:     k8sClient,
		queue:         queue,
		kymaGVR:       gvr,
		logger:        logger,
		sleepDuration: sleepDuration,
		labelKey:      labelKey,
		ctx:           ctx,
	}, nil
}

func (u *Updater) Run() error {
	for {
		u.logger.Debug("Checking if there is an item to process in the queue")
		item, ok := u.queue.Extract()
		if !ok {
			time.Sleep(u.sleepDuration)
			continue
		}
		u.logger.Debug(fmt.Sprintf("Item dequeued - subaccountID: %s, betaEnabled %s", item.SubaccountID, item.BetaEnabled))

		ctxWithTimeout, cancel := context.WithTimeout(u.ctx, k8sRequestInterval)
		defer cancel()

		unstructuredList, err := u.k8sClient.Resource(u.kymaGVR).Namespace(namespace).List(ctxWithTimeout, metav1.ListOptions{
			LabelSelector: fmt.Sprintf(subaccountIdLabelFormat, item.SubaccountID),
		})
		if err != nil {
			u.logger.Warn("while listing Kyma CRs: " + err.Error() + "adding item back to the queue")
			u.queue.Insert(item)
			continue
		}
		if len(unstructuredList.Items) == 0 {
			u.logger.Info("no Kyma CRs found for subaccount" + item.SubaccountID)
			continue
		}
		retryRequired := false
		u.logger.Debug(fmt.Sprintf("found %d Kyma CRs for subaccount", len(unstructuredList.Items)))
		for _, kymaCrUnstructured := range unstructuredList.Items {
			if err := u.updateBetaEnabledLabel(kymaCrUnstructured, item.BetaEnabled, ctxWithTimeout); err != nil {
				u.logger.Warn("while updating Kyma CR: " + err.Error() + "item will be added back to the queue")
				retryRequired = true
			}
		}
		if retryRequired {
			u.logger.Info(fmt.Sprintf("Requeue item for subaccount: %s", item.SubaccountID))
			u.queue.Insert(item)
		}
	}
}

func (u *Updater) updateBetaEnabledLabel(un unstructured.Unstructured, betaEnabled string, ctx context.Context) error {
	labels := un.GetLabels()
	if labels == nil {
		labels = make(map[string]string)
	}
	labels[u.labelKey] = betaEnabled
	un.SetLabels(labels)
	_, err := u.k8sClient.Resource(u.kymaGVR).Namespace(namespace).Update(ctx, &un, metav1.UpdateOptions{})
	return err
}
