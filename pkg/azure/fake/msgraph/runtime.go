package msgraph

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	graphsdk "github.com/nais/msgraph.go/v1.0"

	"github.com/nais/azureator/pkg/azure"
	"github.com/nais/azureator/pkg/azure/client/application"
	"github.com/nais/azureator/pkg/config"
)

type Runtime interface {
	azure.RuntimeClient
	Application() application.Application
}

type runtime struct {
	graph *graphsdk.GraphServiceRequestBuilder
}

func NewRuntime(t testing.TB, handler http.Handler) Runtime {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	graph := graphsdk.NewClient(server.Client())
	graph.SetURL(server.URL + "/v1.0")
	return runtime{graph: graph}
}

func (r runtime) Config() *config.AzureConfig                       { return &config.AzureConfig{} }
func (r runtime) GraphClient() *graphsdk.GraphServiceRequestBuilder { return r.graph }
func (r runtime) HttpClient() *http.Client                          { return http.DefaultClient }
func (r runtime) DelayIntervalBetweenModifications() time.Duration  { return 0 }
func (r runtime) MaxNumberOfPagesToFetch() int                      { return 1 }
func (r runtime) Application() application.Application              { return application.NewApplication(r) }
