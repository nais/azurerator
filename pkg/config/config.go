package config

import (
	"errors"
	"sort"
	"strings"

	"github.com/mitchellh/mapstructure"
	azure "github.com/nais/azureator/pkg/azure/config"
	log "github.com/sirupsen/logrus"
	flag "github.com/spf13/pflag"
	"github.com/spf13/viper"
)

type Config struct {
	AzureAd     azure.Config `json:"azure"`
	MetricsAddr string       `json:"metrics-address"`
	ClusterName string       `json:"cluster-name"`
	Debug       bool         `json:"debug"`
	Annotations Annotations  `json:"annotations"`
}

type Annotations struct {
	Tenant Annotation `json:"tenant"`
}

type Annotation struct {
	Required bool `json:"required"`
}

const (
	MetricsAddress            = "metrics-address"
	ClusterName               = "cluster-name"
	Debug                     = "debug"
	AnnotationsTenantRequired = "annotations.tenant.required"
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
	flag.String(ClusterName, "", "The cluster in which this application should run")
	flag.Bool(Debug, false, "Debug mode toggle")
	flag.Bool(AnnotationsTenantRequired, false, "If true, will only process resources that have a tenant annotation")
}

// Print out all configuration options except secret stuff.
func (c Config) Print(redacted []string) {
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

func (c Config) Validate(required []string) error {
	present := func(key string) bool {
		for _, requiredKey := range required {
			if requiredKey == key {
				return len(viper.GetString(requiredKey)) > 0
			}
		}
		return true
	}
	var keys sort.StringSlice = viper.AllKeys()
	errs := make([]string, 0)

	keys.Sort()
	for _, key := range keys {
		if !present(key) {
			errs = append(errs, key)
		}
	}
	for _, key := range errs {
		log.Printf("required key '%s' not configured", key)
	}
	if len(errs) > 0 {
		return errors.New("missing configuration values")
	}
	return nil
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
