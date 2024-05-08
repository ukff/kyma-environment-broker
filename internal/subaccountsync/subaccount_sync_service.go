package subaccountsync

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/kyma-project/kyma-environment-broker/internal/kymacustomresource"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	queues "github.com/kyma-project/kyma-environment-broker/internal/syncqueues"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	subaccountIDLabel     = "kyma-project.io/subaccount-id"
	runtimeIDLabel        = "kyma-project.io/runtime-id"
	betaEnabledLabel      = "operator.kyma-project.io/beta"
	eventServicePath      = "%s/events/v1/events/central"
	subaccountServicePath = "%s/accounts/v1/technical/subaccounts/%s"
	eventType             = "Subaccount_Creation,Subaccount_Update"
)

type (
	subaccountIDType string
	runtimeIDType    string
	runtimeStateType struct {
		betaEnabled string
	}
	subaccountRuntimesType map[runtimeIDType]runtimeStateType
	statesFromCisType      map[subaccountIDType]CisStateType
	subaccountsSetType     map[subaccountIDType]struct{}
	subaccountStateType    struct {
		cisState       CisStateType
		resourcesState subaccountRuntimesType
		pendingDelete  bool
	}
	inMemoryStateType   map[subaccountIDType]subaccountStateType
	stateReconcilerType struct {
		inMemoryState    inMemoryStateType
		mutex            sync.Mutex
		eventsClient     *RateLimitedCisClient
		accountsClient   *RateLimitedCisClient
		kcpK8sClient     *client.Client
		dynamicK8sClient *dynamic.Interface
		db               storage.BrokerStorage
		syncQueue        queues.MultiConsumerPriorityQueue
		logger           *slog.Logger
		updater          *kymacustomresource.Updater
		metrics          *Metrics
		eventWindow      *EventWindow
	}
)

type SyncService struct {
	appName   string
	ctx       context.Context
	cfg       Config
	kymaGVR   schema.GroupVersionResource
	db        storage.BrokerStorage
	k8sClient dynamic.Interface
}

func NewSyncService(appName string, ctx context.Context, cfg Config, kymaGVR schema.GroupVersionResource, db storage.BrokerStorage, dynamicClient dynamic.Interface) *SyncService {
	return &SyncService{
		appName:   appName,
		ctx:       ctx,
		cfg:       cfg,
		kymaGVR:   kymaGVR,
		db:        db,
		k8sClient: dynamicClient,
	}
}

func (s *SyncService) Run() {
	logger := slog.Default()
	logger.Info(fmt.Sprintf("%s service started", s.appName))

	// create CIS clients
	eventsClient := CreateEventsClient(s.ctx, s.cfg.CisEvents, logger)
	accountsClient := CreateAccountsClient(s.ctx, s.cfg.CisAccounts, logger)

	// create and register metrics
	metricsRegistry := prometheus.NewRegistry()
	metricsRegistry.MustRegister(collectors.NewGoCollector())

	metrics := NewMetrics(metricsRegistry, s.appName)
	promHandler := promhttp.HandlerFor(metricsRegistry, promhttp.HandlerOpts{Registry: metricsRegistry})
	http.Handle("/metrics", promHandler)

	go func() {
		address := fmt.Sprintf(":%s", s.cfg.MetricsPort)
		err := http.ListenAndServe(address, nil)
		if err != nil {
			logger.Error(fmt.Sprintf("while serving metrics: %s", err))
		}
	}()

	// create priority queue
	priorityQueue := queues.NewPriorityQueueWithCallbacks(logger, &queues.EventHandler{
		OnInsert: func(queueSize int) {
			metrics.queue.Set(float64(queueSize))
			metrics.queueOps.With(prometheus.Labels{"operation": "insert"}).Inc()
		},
		OnExtract: func(queueSize int, timeEnqueuedNano int64) {
			metrics.queue.Set(float64(queueSize))
			metrics.queueOps.With(prometheus.Labels{"operation": "extract"}).Inc()
			timeEnqueuedMillis := timeEnqueuedNano / int64(time.Millisecond)
			logger.Debug(fmt.Sprintf("Item extracted from the queue after %d ms", timeEnqueuedMillis))
			metrics.timeInQueue.Set(float64(timeEnqueuedMillis))
		},
	})

	// create updater if needed
	var updater *kymacustomresource.Updater
	var err error
	if s.cfg.UpdateResources {
		logger.Debug("Resource update is enabled, creating updater")
		updater, err = kymacustomresource.NewUpdater(
			s.k8sClient,
			priorityQueue,
			s.kymaGVR,
			s.cfg.SyncQueueSleepInterval,
			betaEnabledLabel,
			s.ctx,
			logger.With("component", "updater"))
		fatalOnError(err)
	}

	// create state reconciler
	stateReconciler := stateReconcilerType{
		inMemoryState:    make(inMemoryStateType),
		mutex:            sync.Mutex{},
		eventsClient:     eventsClient,
		accountsClient:   accountsClient,
		dynamicK8sClient: &s.k8sClient,
		logger:           logger.With("component", "state-reconciler"),
		db:               s.db,
		updater:          updater,
		syncQueue:        priorityQueue,
		metrics:          metrics,
		eventWindow:      NewEventWindow(s.cfg.EventsWindowSize.Milliseconds(), epochInMillis),
	}

	stateReconciler.recreateStateFromDB()

	factory := dynamicinformer.NewFilteredDynamicSharedInformerFactory(s.k8sClient, time.Minute, "kcp-system", nil)
	informer := factory.ForResource(s.kymaGVR).Informer()

	configureInformer(&informer, &stateReconciler, logger.With("component", "informer"), metrics)

	go stateReconciler.runCronJobs(s.cfg, s.ctx)

	if s.cfg.UpdateResources && stateReconciler.updater != nil {
		logger.Info("Starting updater")

		go func() {
			err := stateReconciler.updater.Run()
			if err != nil {
				logger.Warn(fmt.Sprintf("while running updater: %s, cannot update", err))
			}
		}()
	} else {
		logger.Info("Resource update is disabled")
	}

	informer.Run(s.ctx.Done())
}

func CreateAccountsClient(ctx context.Context, accountsConfig CisEndpointConfig, logger *slog.Logger) *RateLimitedCisClient {
	accountsClient := NewRateLimitedCisClient(ctx, accountsConfig, logger.With("component", "CIS-Accounts-client"))
	return accountsClient
}

func CreateEventsClient(ctx context.Context, eventsConfig CisEndpointConfig, logger *slog.Logger) *RateLimitedCisClient {
	eventsClient := NewRateLimitedCisClient(ctx, eventsConfig, logger.With("component", "CIS-Events-client"))
	return eventsClient
}

func getDataFromLabels(u *unstructured.Unstructured) (subaccountID string, runtimeID string, betaEnabled string) {
	labels := u.GetLabels()
	subaccountID = labels[subaccountIDLabel]
	runtimeID = labels[runtimeIDLabel]
	betaEnabled = labels[betaEnabledLabel]
	return
}

func fatalOnError(err error) {
	if err != nil {
		panic(err)
	}
}
