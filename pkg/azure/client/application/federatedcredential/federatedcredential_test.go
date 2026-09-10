package federatedcredential

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	msgraph "github.com/nais/msgraph.go/v1.0"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	azureconfig "github.com/nais/azureator/pkg/config"
	"github.com/nais/azureator/pkg/transaction"
	v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
)

func TestProcess(t *testing.T) {
	t.Run("reconciles credentials", func(t *testing.T) {
		existing := []msgraph.FederatedIdentityCredential{
			credential("kept", "kept-id", "audience", "issuer", "subject"),
			credential("first", "first-id", "first-audience", "first-issuer", "first-subject"),
			credential("second", "second-id", "second-audience", "second-issuer", "second-subject"),
			credential("deleted", "deleted-id", "audience", "issuer", "subject"),
		}
		desired := []v1.AzureAdFederatedCredential{
			{Name: "kept", Audience: "audience", Issuer: "issuer", Subject: "subject"},
			{Name: "first", Audience: "new-first-audience", Issuer: "new-first-issuer", Subject: "new-first-subject"},
			{Name: "second", Audience: "new-second-audience", Issuer: "new-second-issuer", Subject: "new-second-subject"},
			{Name: "created", Audience: "audience", Issuer: "issuer", Subject: "subject"},
		}
		requests, err := processRequests(t, existing, desired)

		require.NoError(t, err)
		assertRequests(t, requests,
			deleteRequest("deleted-id"),
			patchRequest("first-id", map[string]any{
				"audiences": []any{"new-first-audience"}, "issuer": "new-first-issuer", "subject": "new-first-subject",
			}),
			patchRequest("second-id", map[string]any{
				"audiences": []any{"new-second-audience"}, "issuer": "new-second-issuer", "subject": "new-second-subject",
			}),
			postRequest(map[string]any{
				"name": "created", "audiences": []any{"audience"}, "issuer": "issuer", "subject": "subject",
			}),
		)
	})

	t.Run("deletes credentials before recreating swapped issuer and subject pairs", func(t *testing.T) {
		existing := []msgraph.FederatedIdentityCredential{
			credential("first", "first-id", "audience", "first-issuer", "first-subject"),
			credential("second", "second-id", "audience", "second-issuer", "second-subject"),
		}
		desired := []v1.AzureAdFederatedCredential{
			{Name: "first", Audience: "audience", Issuer: "second-issuer", Subject: "second-subject"},
			{Name: "second", Audience: "audience", Issuer: "first-issuer", Subject: "first-subject"},
		}
		requests, err := processRequests(t, existing, desired)

		require.NoError(t, err)
		assertRequests(t, requests,
			deleteRequest("first-id"),
			deleteRequest("second-id"),
			postRequest(map[string]any{
				"name": "first", "audiences": []any{"audience"}, "issuer": "second-issuer", "subject": "second-subject",
			}),
			postRequest(map[string]any{
				"name": "second", "audiences": []any{"audience"}, "issuer": "first-issuer", "subject": "first-subject",
			}),
		)
	})
}

func TestDiff(t *testing.T) {
	t.Run("replaces credentials when issuer and subject pairs are swapped", func(t *testing.T) {
		existing := []msgraph.FederatedIdentityCredential{
			credential("second", "second-id", "audience", "second-issuer", "second-subject"),
			credential("first", "first-id", "audience", "first-issuer", "first-subject"),
		}
		desired := []v1.AzureAdFederatedCredential{
			{Name: "first", Audience: "audience", Issuer: "second-issuer", Subject: "second-subject"},
			{Name: "second", Audience: "audience", Issuer: "first-issuer", Subject: "first-subject"},
		}
		actual, err := diff(existing, desired)

		require.NoError(t, err)
		assert.Equal(t, credentialOperations{
			toDelete: []msgraph.FederatedIdentityCredential{existing[1], existing[0]},
			toCreate: desired,
		}, actual)
	})

	t.Run("deletes an obsolete conflict before updating", func(t *testing.T) {
		existing := []msgraph.FederatedIdentityCredential{
			credential("kept", "kept-id", "audience", "old-issuer", "old-subject"),
			credential("obsolete", "obsolete-id", "audience", "new-issuer", "new-subject"),
		}
		desired := []v1.AzureAdFederatedCredential{
			{Name: "kept", Audience: "audience", Issuer: "new-issuer", Subject: "new-subject"},
		}
		actual, err := diff(existing, desired)

		require.NoError(t, err)
		assert.Equal(t, credentialOperations{
			toDelete: existing[1:],
			toUpdate: []credentialUpdate{{id: "kept-id", AzureAdFederatedCredential: desired[0]}},
		}, actual)
	})

	t.Run("recreates a credential whose desired pair is held by another desired credential", func(t *testing.T) {
		existing := []msgraph.FederatedIdentityCredential{
			credential("first", "first-id", "audience", "first-issuer", "first-subject"),
			credential("second", "second-id", "audience", "second-issuer", "second-subject"),
		}
		desired := []v1.AzureAdFederatedCredential{
			{Name: "first", Audience: "audience", Issuer: "third-issuer", Subject: "third-subject"},
			{Name: "second", Audience: "audience", Issuer: "first-issuer", Subject: "first-subject"},
		}
		actual, err := diff(existing, desired)

		require.NoError(t, err)
		assert.Equal(t, credentialOperations{
			toCreate: desired[1:],
			toUpdate: []credentialUpdate{{id: "first-id", AzureAdFederatedCredential: desired[0]}},
			toDelete: existing[1:],
		}, actual)
	})

	t.Run("empty spec deletes all existing credentials", func(t *testing.T) {
		existing := []msgraph.FederatedIdentityCredential{
			credential("first", "first-id", "audience", "issuer", "subject"),
			credential("second", "second-id", "audience", "issuer", "subject"),
		}
		actual, err := diff(existing, nil)

		require.NoError(t, err)
		assert.Equal(t, credentialOperations{toDelete: existing}, actual)
	})

	t.Run("updates a credential with incomplete Graph fields", func(t *testing.T) {
		existing := []msgraph.FederatedIdentityCredential{{
			ID:   new("existing-id"),
			Name: new("existing"),
		}}
		desired := []v1.AzureAdFederatedCredential{
			{Name: "existing", Audience: "audience", Issuer: "issuer", Subject: "subject"},
		}
		actual, err := diff(existing, desired)

		require.NoError(t, err)
		assert.Equal(t, credentialOperations{toUpdate: []credentialUpdate{{id: "existing-id", AzureAdFederatedCredential: desired[0]}}}, actual)
	})
}

const collectionPath = "/v1.0/applications/application-id/federatedIdentityCredentials"

type request struct {
	Method string
	Path   string
	Body   map[string]any
}

func processRequests(t *testing.T, existing []msgraph.FederatedIdentityCredential, desired []v1.AzureAdFederatedCredential) ([]request, error) {
	t.Helper()
	var requests []request
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rawBody, err := io.ReadAll(r.Body)
		if !assert.NoError(t, err) {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		var body map[string]any
		if len(rawBody) > 0 && !assert.NoError(t, json.Unmarshal(rawBody, &body)) {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		requests = append(requests, request{Method: r.Method, Path: r.URL.Path, Body: body})

		if r.Method == http.MethodGet {
			w.Header().Set("Content-Type", "application/json")
			assert.NoError(t, json.NewEncoder(w).Encode(map[string]any{"value": existing}))
			return
		}
		if r.Method == http.MethodPost {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			assert.NoError(t, json.NewEncoder(w).Encode(msgraph.FederatedIdentityCredential{}))
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	baseURL, err := url.Parse(server.URL)
	require.NoError(t, err)
	httpClient := &http.Client{Transport: rewriteTransport{baseURL: baseURL, next: http.DefaultTransport}}
	runtime := testRuntimeClient{graph: msgraph.NewClient(httpClient)}
	instance := &v1.AzureAdApplication{Status: v1.AzureAdApplicationStatus{ObjectId: "application-id"}, Spec: v1.AzureAdApplicationSpec{FederatedCredentials: desired}}
	tx := transaction.Transaction{Ctx: t.Context(), Instance: instance, Logger: *log.NewEntry(log.New())}
	err = NewFederatedCredential(runtime).Process(tx)
	return requests, err
}

func assertRequests(t *testing.T, actual []request, expected ...request) {
	t.Helper()
	expected = append([]request{{Method: http.MethodGet, Path: collectionPath}}, expected...)
	assert.Equal(t, expected, actual)
}

func deleteRequest(id string) request {
	return request{Method: http.MethodDelete, Path: collectionPath + "/" + id}
}

func patchRequest(id string, body map[string]any) request {
	return request{Method: http.MethodPatch, Path: collectionPath + "/" + id, Body: body}
}

func postRequest(body map[string]any) request {
	return request{Method: http.MethodPost, Path: collectionPath, Body: body}
}

func credential(name, id, audience, issuer, subject string) msgraph.FederatedIdentityCredential {
	return msgraph.FederatedIdentityCredential{ID: new(id), Name: new(name), Audiences: []string{audience}, Issuer: new(issuer), Subject: new(subject)}
}

type rewriteTransport struct {
	baseURL *url.URL
	next    http.RoundTripper
}

func (t rewriteTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	r.URL.Scheme = t.baseURL.Scheme
	r.URL.Host = t.baseURL.Host
	return t.next.RoundTrip(r)
}

type testRuntimeClient struct {
	graph *msgraph.GraphServiceRequestBuilder
}

func (t testRuntimeClient) Config() *azureconfig.AzureConfig                 { return &azureconfig.AzureConfig{} }
func (t testRuntimeClient) GraphClient() *msgraph.GraphServiceRequestBuilder { return t.graph }
func (t testRuntimeClient) HttpClient() *http.Client                         { return http.DefaultClient }
func (t testRuntimeClient) DelayIntervalBetweenModifications() time.Duration { return 0 }
func (t testRuntimeClient) MaxNumberOfPagesToFetch() int                     { return 1 }
