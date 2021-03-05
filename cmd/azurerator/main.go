package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/pprof"
	"os"
	"time"

	"github.com/go-logr/zapr"
	"github.com/nais/azureator/controllers/azureadapplication"
	"github.com/nais/azureator/pkg/azure/client"
	"github.com/nais/azureator/pkg/config"
	azureMetrics "github.com/nais/azureator/pkg/metrics"
	log "github.com/sirupsen/logrus"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/metrics"

	naisiov1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	// +kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	metrics.Registry.MustRegister(azureMetrics.AllMetrics...)

	formatter := log.JSONFormatter{
		TimestampFormat: time.RFC3339Nano,
	}
	log.SetFormatter(&formatter)
	log.SetLevel(log.DebugLevel)

	_ = clientgoscheme.AddToScheme(scheme)
	_ = naisiov1.AddToScheme(scheme)
	// +kubebuilder:scaffold:scheme
}

func main() {
	err := run()

	if err != nil {
		setupLog.Error(err, "Run loop errored")
		os.Exit(1)
	}

	setupLog.Info("Manager shutting down")
}

func run() error {
	zapLogger, err := setupZapLogger()
	if err != nil {
		return err
	}
	ctrl.SetLogger(zapr.NewLogger(zapLogger))

	ctx := context.Background()

	cfg, err := setupConfig()
	if err != nil {
		return err
	}

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:             scheme,
		MetricsBindAddress: cfg.MetricsAddr,
		LeaderElection:     false,
	})
	if err != nil {
		return fmt.Errorf("unable to start manager: %w", err)
	}

	if cfg.Debug {
		if err := addDebugHandler(mgr); err != nil {
			return fmt.Errorf("unable to register debug handler: %w", err)
		}
	}

	azureClient, err := client.New(ctx, &cfg.Azure)
	if err != nil {
		return fmt.Errorf("unable to create Azure client: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, 1*time.Minute)
	defer cancel()
	azureOpenIDConfig, err := config.NewAzureOpenIdConfig(ctx, cfg.Azure.Tenant)
	if err != nil {
		return fmt.Errorf("fetching Azure OpenID Configuration: %w", err)
	}

	if err = (&azureadapplication.Reconciler{
		Client:            mgr.GetClient(),
		Reader:            mgr.GetAPIReader(),
		Scheme:            mgr.GetScheme(),
		AzureClient:       azureClient,
		Config:            cfg,
		Recorder:          mgr.GetEventRecorderFor("azurerator"),
		AzureOpenIDConfig: *azureOpenIDConfig,
	}).SetupWithManager(mgr); err != nil {
		return fmt.Errorf("unable to create controller: %w", err)
	}

	// +kubebuilder:scaffold:builder

	setupLog.Info("starting metrics refresh goroutine")
	clusterMetrics := azureMetrics.New(mgr.GetAPIReader())
	go clusterMetrics.Refresh(context.Background())

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		return fmt.Errorf("problem running manager: %w", err)
	}

	return nil
}

func setupZapLogger() (*zap.Logger, error) {
	loggerConfig := zap.NewProductionConfig()
	loggerConfig.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
	loggerConfig.EncoderConfig.TimeKey = "timestamp"
	loggerConfig.EncoderConfig.EncodeTime = zapcore.RFC3339NanoTimeEncoder
	return loggerConfig.Build()
}

func setupConfig() (*config.Config, error) {
	cfg, err := config.New()
	if err != nil {
		return nil, err
	}

	cfg.Print([]string{
		config.AzureClientSecret,
	})

	if err = cfg.Validate([]string{
		config.AzureTenantId,
		config.AzureClientId,
		config.AzureClientSecret,
		config.AzurePermissionGrantResourceId,
		config.ClusterName,
	}); err != nil {
		return nil, err
	}
	return cfg, nil
}

func addDebugHandler(mgr ctrl.Manager) error {
	log.Info("registering debug handler")
	mux := http.NewServeMux()
	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)

	if err := mgr.AddMetricsExtraHandler("/debug/pprof/", mux); err != nil {
		return err
	}
	return nil
}
