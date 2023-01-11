package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"time"

	"github.com/go-logr/zapr"
	"github.com/nais/liberator/pkg/tlsutil"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/metrics"

	"github.com/nais/azureator/controllers/azureadapplication"
	"github.com/nais/azureator/pkg/azure/client"
	"github.com/nais/azureator/pkg/config"
	"github.com/nais/azureator/pkg/kafka"
	"github.com/nais/azureator/pkg/logger"
	azureMetrics "github.com/nais/azureator/pkg/metrics"
	"github.com/nais/azureator/pkg/synchronizer"

	naisiov1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	// +kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	metrics.Registry.MustRegister(azureMetrics.AllMetrics...)
	logger.SetupLogrus()

	_ = clientgoscheme.AddToScheme(scheme)
	_ = naisiov1.AddToScheme(scheme)
	// +kubebuilder:scaffold:scheme
}

func main() {
	err := run()
	if err != nil {
		log.Fatalf("Run loop errored: %+v", err)
	}

	setupLog.Info("Manager shutting down")
}

func run() error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	zapLogger, err := logger.ZapLogger()
	if err != nil {
		return err
	}
	ctrl.SetLogger(zapr.NewLogger(zapLogger))

	cfg, err := config.DefaultConfig()
	if err != nil {
		return err
	}

	leaseDuration := 25 * time.Second
	renewDeadline := 20 * time.Second

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                     scheme,
		MetricsBindAddress:         cfg.MetricsAddr,
		LeaderElection:             cfg.LeaderElection.Enabled,
		LeaderElectionID:           fmt.Sprintf("azurerator.nais.io-%s", cfg.Azure.Tenant.Id),
		LeaderElectionNamespace:    cfg.LeaderElection.Namespace,
		LeaderElectionResourceLock: resourcelock.LeasesResourceLock,
		LeaseDuration:              &leaseDuration,
		RenewDeadline:              &renewDeadline,
	})
	if err != nil {
		return fmt.Errorf("unable to start manager: %w", err)
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

	var kafkaProducer kafka.Producer
	if cfg.Kafka.Enabled {
		kafkaLogger := logrus.StandardLogger()
		var tlsConfig *tls.Config

		if cfg.Kafka.TLS.Enabled {
			tlsConfig, err = tlsutil.TLSConfigFromFiles(cfg.Kafka.TLS.CertificatePath, cfg.Kafka.TLS.PrivateKeyPath, cfg.Kafka.TLS.CAPath)
			if err != nil {
				return fmt.Errorf("loading Kafka TLS credentials: %w", err)
			}
		}

		kafkaProducer, err = kafka.NewProducer(*cfg, tlsConfig, kafkaLogger)
		if err != nil {
			return fmt.Errorf("setting up kafka producer: %w", err)
		}

		callback := synchronizer.NewSynchronizer(*cfg, mgr.GetClient(), mgr.GetAPIReader()).Callback()

		_, err = kafka.NewConsumer(*cfg, tlsConfig, kafkaLogger, callback)
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
		Recorder:          mgr.GetEventRecorderFor("azurerator"),
		AzureOpenIDConfig: *azureOpenIDConfig,
		KafkaProducer:     kafkaProducer,
	}).SetupWithManager(mgr); err != nil {
		return fmt.Errorf("unable to create controller: %w", err)
	}

	// +kubebuilder:scaffold:builder

	setupLog.Info("starting metrics refresh goroutine")
	clusterMetrics := azureMetrics.New(mgr.GetClient())
	go clusterMetrics.Refresh(ctx)

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		return fmt.Errorf("problem running manager: %w", err)
	}

	return nil
}
