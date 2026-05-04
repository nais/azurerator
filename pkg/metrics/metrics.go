package metrics

import (
	"context"
	"fmt"
	"time"

	v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	"github.com/nais/liberator/pkg/kubernetes"
	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/nais/azureator/pkg/labels"
	"github.com/nais/azureator/pkg/retry"
)

const (
	labelNamespace = "namespace"
)

var (
	AzureAppsTotal = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "azureadapp_total",
		})
	AzureAppSecretsTotal = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "azureadapp_secrets_total",
			Help: "Total number of azureadapp secrets",
		},
	)
	AzureAppOrphanedTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "azureadapp_orphaned_total",
			Help: "Number of orphaned azuread apps (exists in Azure AD without matching k8s resource)",
		},
		[]string{labelNamespace, "tenant"},
	)
	AzureAppOrphanedCleanedTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "azureadapp_orphaned_cleaned_total",
			Help: "Number of orphaned azuread apps successfully deleted from Azure AD.",
		},
		[]string{labelNamespace, "tenant"},
	)
	AzureAppsCreatedCount = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "azureadapp_created_count",
			Help: "Number of azureadapps created successfully",
		},
		[]string{labelNamespace},
	)
	AzureAppsUpdatedCount = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "azureadapp_updated_count",
			Help: "Number of azureadapps updated successfully",
		},
		[]string{labelNamespace},
	)
	AzureAppsRotatedCount = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "azureadapp_rotated_count",
			Help: "Number of azureadapps successfully rotated credentials",
		},
		[]string{labelNamespace},
	)
	AzureAppsProcessedCount = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "azureadapp_processed_count",
			Help: "Number of azureadapps processed successfully",
		},
		[]string{labelNamespace},
	)
	AzureAppsFailedProcessingCount = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "azureadapp_failed_processing_count",
			Help: "Number of azureadapps that failed processing",
		},
		[]string{labelNamespace},
	)
	AzureAppsDeletedCount = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "azureadapp_deleted_count",
			Help: "Number of azureadapps successfully deleted",
		},
		[]string{labelNamespace},
	)
	AzureAppsSkippedCount = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "azureadapp_skipped_count",
			Help: "Number of azureapps skipped due to certain conditions",
		},
		[]string{labelNamespace},
	)
	ResyncEventsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "azureadapp_resync_events_total",
			Help: "Number of resync events received by the synchronizer, by source/event/result.",
		},
		[]string{"source", "event", "result"},
	)
	ResyncCandidatesTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "azureadapp_resync_candidates_total",
			Help: "Number of dependent AzureAdApplications marked for resync.",
		},
		[]string{labelNamespace, "event"},
	)
	ResyncFailedTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "azureadapp_resync_failed_total",
			Help: "Number of resync attempts that failed to update the dependent AzureAdApplication.",
		},
		[]string{labelNamespace, "event"},
	)
	ResyncFanout = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "azureadapp_resync_fanout",
			Help:    "Number of dependent apps marked for resync per inbound event.",
			Buckets: []float64{0, 1, 2, 5, 10, 25, 50, 100},
		},
		[]string{"event"},
	)
)

var AllMetrics = []prometheus.Collector{
	AzureAppsTotal,
	AzureAppSecretsTotal,
	AzureAppOrphanedTotal,
	AzureAppOrphanedCleanedTotal,
	AzureAppsProcessedCount,
	AzureAppsFailedProcessingCount,
	AzureAppsCreatedCount,
	AzureAppsUpdatedCount,
	AzureAppsRotatedCount,
	AzureAppsDeletedCount,
	AzureAppsSkippedCount,
	ResyncEventsTotal,
	ResyncCandidatesTotal,
	ResyncFailedTotal,
	ResyncFanout,
}

var AllCounters = []*prometheus.CounterVec{
	AzureAppsProcessedCount,
	AzureAppsFailedProcessingCount,
	AzureAppsCreatedCount,
	AzureAppsUpdatedCount,
	AzureAppsRotatedCount,
	AzureAppsDeletedCount,
	AzureAppsSkippedCount,
	ResyncCandidatesTotal,
	ResyncFailedTotal,
}

func IncWithNamespaceLabel(metric *prometheus.CounterVec, namespace string) {
	metric.WithLabelValues(namespace).Inc()
}

type Metrics interface {
	Refresh(ctx context.Context)
}

type metrics struct {
	reader client.Reader
}

func New(reader client.Reader) Metrics {
	return metrics{
		reader: reader,
	}
}

func (m metrics) InitWithNamespaceLabels() {
	var ns corev1.NamespaceList
	var err error

	retryable := func(ctx context.Context) error {
		ns, err = kubernetes.ListNamespaces(context.Background(), m.reader)
		if err != nil {
			return retry.RetryableError(fmt.Errorf("listing namespaces: %w", err))
		}
		return nil
	}

	err = retry.Fibonacci(1*time.Second).
		WithMaxDuration(1*time.Minute).
		Do(context.Background(), retryable)
	if err != nil {
		log.Error(err)
	}

	for _, n := range ns.Items {
		for _, c := range AllCounters {
			c.WithLabelValues(n.Name).Add(0)
		}
	}

	log.Infof("metrics with namespace labels initialized")
}

func (m metrics) Refresh(ctx context.Context) {
	var err error
	exp := 10 * time.Second

	mLabels := client.MatchingLabels{
		labels.TypeLabelKey: labels.TypeLabelValue,
	}

	var secretList corev1.SecretList
	var azureAdAppList v1.AzureAdApplicationList

	m.InitWithNamespaceLabels()

	t := time.NewTicker(exp)
	for range t.C {
		if err = m.reader.List(ctx, &secretList, mLabels); err != nil {
			log.Errorf("failed to list secrets: %v", err)
		}
		AzureAppSecretsTotal.Set(float64(len(secretList.Items)))

		if err = m.reader.List(ctx, &azureAdAppList); err != nil {
			log.Errorf("failed to list azure apps: %v", err)
		}
		AzureAppsTotal.Set(float64(len(azureAdAppList.Items)))
	}
}
