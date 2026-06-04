package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"time"

	"github.com/IBM/sarama"
	"github.com/go-logr/logr"
	"github.com/nais/liberator/pkg/logrus2logr"
	"github.com/nais/liberator/pkg/tlsutil"
	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"

	"github.com/nais/azureator/controllers/azureadapplication"
	"github.com/nais/azureator/pkg/azure/client"
	"github.com/nais/azureator/pkg/config"
	"github.com/nais/azureator/pkg/kafka"
	azureMetrics "github.com/nais/azureator/pkg/metrics"
	"github.com/nais/azureator/pkg/synchronizer"

	naisiov1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	// +kubebuilder:scaffold:imports
)

var scheme = runtime.NewScheme()

func init() {
	metrics.Registry.MustRegister(azureMetrics.AllMetrics...)

	formatter := &log.JSONFormatter{
		TimestampFormat: time.RFC3339Nano,
	}
	log.SetFormatter(formatter)
	log.SetLevel(log.DebugLevel)

	ctrllog := log.New()
	ctrllog.Formatter = formatter
	ctrllog.Level = log.InfoLevel
	ctrl.SetLogger(logr.New(&logrus2logr.Logrus2Logr{Logger: ctrllog}))

	_ = clientgoscheme.AddToScheme(scheme)
	_ = naisiov1.AddToScheme(scheme)
	// +kubebuilder:scaffold:scheme
}

func main() {
	err := run()
	if err != nil {
		log.Fatalf("Run loop errored: %+v", err)
	}
}

func run() error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	setupLog := ctrl.Log.WithName("setup")

	cfg, err := config.DefaultConfig()
	if err != nil {
		return err
	}

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme: scheme,
		Metrics: metricsserver.Options{
			BindAddress: cfg.MetricsAddr,
		},
		HealthProbeBindAddress:     cfg.ProbesAddr,
		LivenessEndpointName:       "/healthz",
		ReadinessEndpointName:      "/readyz",
		LeaderElection:             cfg.LeaderElection.Enabled,
		LeaderElectionID:           fmt.Sprintf("azurerator.nais.io-%s", cfg.Azure.Tenant.Id),
		LeaderElectionNamespace:    cfg.LeaderElection.Namespace,
		LeaderElectionResourceLock: resourcelock.LeasesResourceLock,
		LeaseDuration:              new(25 * time.Second),
		RenewDeadline:              new(20 * time.Second),
	})
	if err != nil {
		return fmt.Errorf("unable to start manager: %w", err)
	}

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		return fmt.Errorf("unable to set up health check: %w", err)
	}

	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		return fmt.Errorf("unable to set up ready check: %w", err)
	}

	azureClient, err := client.New(ctx, &cfg.Azure)
	if err != nil {
		return fmt.Errorf("instantiating Azure client: %w", err)
	}

	azureCtx, cancel := context.WithTimeout(ctx, 1*time.Minute)
	defer cancel()

	azureOpenIDConfig, err := config.NewAzureOpenIdConfig(azureCtx, cfg.Azure.Tenant)
	if err != nil {
		return fmt.Errorf("fetching Azure OpenID Configuration: %w", err)
	}

	syncer := synchronizer.New(cfg.ClusterName, mgr.GetClient(), mgr.GetAPIReader())

	var kafkaProducer *kafka.Producer
	if cfg.Kafka.Enabled {
		sarama.Logger = log.StandardLogger().WithField("subsystem", "kafka")

		var tlsConfig *tls.Config

		if cfg.Kafka.TLS.Enabled {
			tlsConfig, err = tlsutil.TLSConfigFromFiles(cfg.Kafka.TLS.CertificatePath, cfg.Kafka.TLS.PrivateKeyPath, cfg.Kafka.TLS.CAPath)
			if err != nil {
				return fmt.Errorf("loading Kafka TLS credentials: %w", err)
			}
		}

		kafkaProducer, err = kafka.NewProducer(*cfg, tlsConfig)
		if err != nil {
			return fmt.Errorf("setting up kafka producer: %w", err)
		}

		_, err = kafka.NewConsumer(ctx, *cfg, tlsConfig, syncer.Kafka())
		if err != nil {
			return fmt.Errorf("setting up kafka consumer: %w", err)
		}
	}

	if err = (&azureadapplication.Reconciler{
		Client:            mgr.GetClient(),
		Reader:            mgr.GetAPIReader(),
		Scheme:            mgr.GetScheme(),
		AzureClient:       azureClient,
		Config:            cfg,
		Recorder:          mgr.GetEventRecorder("azurerator"),
		AzureOpenIDConfig: *azureOpenIDConfig,
		KafkaProducer:     kafkaProducer,
		Synchronizer:      syncer,
	}).SetupWithManager(mgr); err != nil {
		return fmt.Errorf("unable to create controller: %w", err)
	}

	// +kubebuilder:scaffold:builder

	setupLog.Info("starting metrics refresh goroutine")
	clusterMetrics := azureMetrics.New(mgr.GetClient())
	go clusterMetrics.Refresh(ctx)

	setupLog.Info("registering synchronizer periodic sweep runnable")
	if err := mgr.Add(synchronizer.NewSweeper(
		cfg.ClusterName,
		mgr.GetClient(),
		mgr.GetAPIReader(),
		azureClient,
		cfg.Azure.Tenant.Id,
		cfg.Controller.SweepInterval,
	)); err != nil {
		return fmt.Errorf("registering synchronizer periodic sweep runnable: %w", err)
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		return fmt.Errorf("problem running manager: %w", err)
	}

	setupLog.Info("Manager shutting down")
	return nil
}
