package client

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	msgraph "github.com/nais/msgraph.go/v1.0"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nais/azureator/pkg/config"
	"github.com/nais/azureator/pkg/transaction"
)

func TestDelete(t *testing.T) {
	const matchingApp = `{"value":[{"id":"object-id","appId":"client-id"}]}`
	tests := []struct {
		name         string
		listStatus   int
		listBody     string
		deleteStatus int
		deleteCode   string
		wantError    bool
	}{
		{"deletes a matching application", 200, matchingApp, 204, "", false},
		{"ignores a stale matching application", 200, matchingApp, 404, "Request_ResourceNotFound", false},
		{"ignores an empty list", 200, `{"value":[]}`, 0, "", false},
		{"returns lookup failures", 500, `{"error":{"code":"InternalServerError","message":"test error"}}`, 0, "", true},
		{"returns unrelated not found delete failures", 200, matchingApp, 404, "SomeOtherNotFound", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var requests []string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests = append(requests, r.Method+" "+r.URL.Path)
				w.Header().Set("Content-Type", "application/json")
				switch r.Method {
				case http.MethodGet:
					assert.Equal(t, "displayName eq 'regression-delete-app'", r.URL.Query().Get("$filter"))
					w.WriteHeader(tt.listStatus)
					_, err := w.Write([]byte(tt.listBody))
					assert.NoError(t, err)
				case http.MethodDelete:
					if tt.deleteStatus == 0 {
						w.WriteHeader(http.StatusInternalServerError)
						return
					}
					w.WriteHeader(tt.deleteStatus)
					if tt.deleteStatus != http.StatusNoContent {
						assert.NoError(t, json.NewEncoder(w).Encode(map[string]any{
							"error": map[string]string{"code": tt.deleteCode, "message": "test error"},
						}))
					}
				default:
					w.WriteHeader(http.StatusInternalServerError)
				}
			}))
			defer server.Close()

			baseURL, err := url.Parse(server.URL)
			require.NoError(t, err)
			httpClient := &http.Client{Transport: deleteTestTransport{baseURL: baseURL, next: http.DefaultTransport}}
			c := Client{
				config:      &config.AzureConfig{Pagination: config.AzurePagination{MaxPages: 1}},
				graphClient: msgraph.NewClient(httpClient),
			}
			tx := transaction.Transaction{
				Ctx:           t.Context(),
				ExistsInAzure: true,
				Instance: &v1.AzureAdApplication{
					Status: v1.AzureAdApplicationStatus{ClientId: "client-id", ObjectId: "object-id"},
				},
				UniformResourceName: "regression-delete-app",
				Logger:              *log.NewEntry(log.New()),
			}

			err = c.Delete(tx)
			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			expected := []string{"GET /v1.0/applications"}
			if tt.deleteStatus != 0 {
				expected = append(expected, "DELETE /v1.0/applications/object-id")
			}
			assert.Equal(t, expected, requests)
		})
	}
}

type deleteTestTransport struct {
	baseURL *url.URL
	next    http.RoundTripper
}

func (t deleteTestTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	r = r.Clone(r.Context())
	r.URL.Scheme = t.baseURL.Scheme
	r.URL.Host = t.baseURL.Host
	return t.next.RoundTrip(r)
}
