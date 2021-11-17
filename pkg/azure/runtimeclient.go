package azure

import (
	"net/http"
	"time"

	msgraph "github.com/nais/msgraph.go/v1.0"

	"github.com/nais/azureator/pkg/config"
)

type RuntimeClient interface {
	Config() *config.AzureConfig
	GraphClient() *msgraph.GraphServiceRequestBuilder
	HttpClient() *http.Client

	DelayIntervalBetweenModifications() time.Duration
	MaxNumberOfPagesToFetch() int
}
