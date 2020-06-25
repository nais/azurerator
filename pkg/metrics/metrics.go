package metrics

import (
	"context"
	"time"

	"github.com/nais/azureator/api/v1"
	"github.com/nais/azureator/pkg/resourcecreator"
	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
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
	AzureAppsProcessedCount = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "azureadapp_processed_count",
			Help: "Number of azureadapps processed",
		},
	)
)

type Metrics interface {
	Refresh(ctx context.Context)
}

type metrics struct {
	cli client.Client
}

func New(cli client.Client) Metrics {
	return metrics{
		cli: cli,
	}
}

func (m metrics) Refresh(ctx context.Context) {
	var err error
	exp := 10 * time.Second

	var mLabels = client.MatchingLabels{}
	mLabels[resourcecreator.TypeLabelKey] = resourcecreator.TypeLabelValue

	var secretList corev1.SecretList
	var azureAdAppList v1.AzureAdApplicationList

	t := time.NewTicker(exp)
	for range t.C {
		log.Debug("Refreshing metrics from cluster")
		if err = m.cli.List(ctx, &secretList, mLabels); err != nil {
			log.Errorf("failed to list secrets: %v", err)
		}
		AzureAppSecretsTotal.Set(float64(len(secretList.Items)))

		if err = m.cli.List(ctx, &azureAdAppList); err != nil {
			log.Errorf("failed to list azure apps: %v", err)
		}
		AzureAppsTotal.Set(float64(len(azureAdAppList.Items)))
	}
}
