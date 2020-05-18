package metrics

import (
	"context"
	"time"

	"github.com/go-logr/logr"
	"github.com/nais/azureator/api/v1alpha1"
	"github.com/nais/azureator/pkg/resourcecreator"
	"github.com/prometheus/client_golang/prometheus"
	v1 "k8s.io/api/core/v1"
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
	AzureAppConfigMapsTotal = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "azureadapp_configmaps_total",
			Help: "Total number of azureadapp configmaps",
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
	Refresh(ctx context.Context) error
}

type metrics struct {
	log logr.Logger
	cli client.Client
}

func New(cli client.Client, log logr.Logger) Metrics {
	return metrics{
		log: log,
		cli: cli,
	}
}

func (m metrics) Refresh(ctx context.Context) error {
	var err error
	exp := 10 * time.Second

	var mLabels = client.MatchingLabels{}
	mLabels["type"] = resourcecreator.LabelType

	var secretList v1.SecretList
	var configMapList v1.ConfigMapList

	var azureAdAppList v1alpha1.AzureAdApplicationList

	t := time.NewTicker(exp)
	for range t.C {
		m.log.Info("Refreshing metrics from cluster")

		if err = m.cli.List(ctx, &secretList, mLabels); err != nil {
			return err
		}
		AzureAppSecretsTotal.Set(float64(len(secretList.Items)))

		if err = m.cli.List(ctx, &configMapList, mLabels); err != nil {
			return err
		}
		AzureAppConfigMapsTotal.Set(float64(len(configMapList.Items)))

		if err = m.cli.List(ctx, &azureAdAppList); err != nil {
			return err
		}
		AzureAppsTotal.Set(float64(len(azureAdAppList.Items)))
	}
	return nil
}
