package config

import (
	"sort"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/nais/azureator/pkg/azure"
	log "github.com/sirupsen/logrus"
	flag "github.com/spf13/pflag"
	"github.com/spf13/viper"
)

type Config struct {
	AzureAd              azure.Config `json:"azure"`
	MetricsAddr          string       `json:"metrics-address"`
	EnableLeaderElection bool         `json:"enable-leader-election"`
	ClusterName          string       `json:"cluster-name"`
}

const (
	MetricsAddress       = "metrics-address"
	EnableLeaderElection = "enable-leader-election"
	ClusterName          = "cluster-name"
)

func init() {
	// Automatically read configuration options from environment variables.
	// e.g. --azure.client.id will be configurable using AZURERATOR_AZURE_CLIENT_ID.
	viper.SetEnvPrefix("AZURERATOR")
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_", ".", "_"))

	// Read configuration file from working directory and/or /etc.
	// File formats supported include JSON, TOML, YAML, HCL, envfile and Java properties config files
	viper.SetConfigName("azurerator")
	viper.AddConfigPath(".")
	viper.AddConfigPath("/etc")

	azure.SetupFlags()

	flag.String(MetricsAddress, ":8080", "The address the metric endpoint binds to.")
	flag.Bool(EnableLeaderElection, false, "Enable leader election for controller manager. Enabling this will ensure there is only one active controller manager.")
	flag.String(ClusterName, "cluster-name-not-set", "The cluster in which this application should run")
}

// Print out all configuration options except secret stuff.
func Print(redacted []string) {
	ok := func(key string) bool {
		for _, forbiddenKey := range redacted {
			if forbiddenKey == key {
				return false
			}
		}
		return true
	}

	var keys sort.StringSlice = viper.AllKeys()

	keys.Sort()
	for _, key := range keys {
		if ok(key) {
			log.Printf("%s: %s", key, viper.GetString(key))
		} else {
			log.Printf("%s: ***REDACTED***", key)
		}
	}
}

func decoderHook(dc *mapstructure.DecoderConfig) {
	dc.TagName = "json"
	dc.ErrorUnused = true
}

func New() (*Config, error) {
	var err error
	var cfg Config

	err = viper.ReadInConfig()
	if err != nil {
		if err.(viper.ConfigFileNotFoundError) != err {
			return nil, err
		}
	}

	flag.Parse()

	err = viper.BindPFlags(flag.CommandLine)
	if err != nil {
		return nil, err
	}

	err = viper.Unmarshal(&cfg, decoderHook)
	if err != nil {
		return nil, err
	}

	return &cfg, nil
}
