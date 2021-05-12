package metrics

import (
	"context"
	"time"

	v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	"github.com/nais/liberator/pkg/kubernetes"
	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/nais/azureator/pkg/labels"
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
)

var AllMetrics = []prometheus.Collector{
	AzureAppsTotal,
	AzureAppSecretsTotal,
	AzureAppsProcessedCount,
	AzureAppsFailedProcessingCount,
	AzureAppsCreatedCount,
	AzureAppsUpdatedCount,
	AzureAppsRotatedCount,
	AzureAppsDeletedCount,
	AzureAppsSkippedCount,
}

var AllCounters = []*prometheus.CounterVec{
	AzureAppsProcessedCount,
	AzureAppsFailedProcessingCount,
	AzureAppsCreatedCount,
	AzureAppsUpdatedCount,
	AzureAppsRotatedCount,
	AzureAppsDeletedCount,
	AzureAppsSkippedCount,
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
	ns, err := kubernetes.ListNamespaces(context.Background(), m.reader)
	if err != nil {
		log.Errorf("failed to list namespaces: %v", err)
	}
	for _, n := range ns.Items {
		for _, c := range AllCounters {
			c.WithLabelValues(n.Name).Add(0)
		}
	}
}

func (m metrics) Refresh(ctx context.Context) {
	var err error
	exp := 1 * time.Minute

	mLabels := client.MatchingLabels{
		labels.TypeLabelKey: labels.TypeLabelValue,
	}

	var secretList corev1.SecretList
	var azureAdAppList v1.AzureAdApplicationList

	m.InitWithNamespaceLabels()

	t := time.NewTicker(exp)
	for range t.C {
		log.Debug("Refreshing metrics from cluster")
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
