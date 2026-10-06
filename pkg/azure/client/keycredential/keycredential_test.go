package keycredential

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	msgraph "github.com/nais/msgraph.go/v1.0"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nais/azureator/pkg/azure/credentials"
	fakemsgraph "github.com/nais/azureator/pkg/azure/fake/msgraph"
	"github.com/nais/azureator/pkg/transaction"
	transactionsecrets "github.com/nais/azureator/pkg/transaction/secrets"
)

var (
	pem        = []byte("secret-pem")
	fromSecret = base64.StdEncoding.EncodeToString(pem)
)

func TestDeleteExpired(t *testing.T) {
	inSecrets := map[string][]byte{"current-id": pem, "undated-id": pem}
	for _, test := range []struct {
		name        string
		keys        []msgraph.KeyCredential
		wantMethods []string
		// wantPatch maps the key IDs sent in the PATCH to their key bytes.
		wantPatch map[string]string
	}{
		{
			name: "no-op", keys: []msgraph.KeyCredential{key("current", "current-id", time.Now().Add(time.Hour))},
			wantMethods: []string{http.MethodGet},
		},
		{
			name: "expired key removed", keys: []msgraph.KeyCredential{
				key("current", "current-id", time.Now().Add(time.Hour)),
				key("expired", "expired-id", time.Now().Add(-time.Hour)),
			}, wantMethods: []string{http.MethodGet, http.MethodPatch},
			wantPatch: map[string]string{"current-id": fromSecret},
		},
		{
			name: "key that no secret holds is dropped", keys: []msgraph.KeyCredential{
				key("current", "current-id", time.Now().Add(time.Hour)),
				key("orphan", "orphan-id", time.Now().Add(time.Hour)),
				key("expired", "expired-id", time.Now().Add(-time.Hour)),
			}, wantMethods: []string{http.MethodGet, http.MethodPatch},
			wantPatch: map[string]string{"current-id": fromSecret},
		},
		{
			name: "nil end date retained", keys: []msgraph.KeyCredential{
				func() msgraph.KeyCredential {
					credential := key("undated", "undated-id", time.Time{})
					credential.EndDateTime = nil
					return credential
				}(),
				key("expired", "expired-id", time.Now().Add(-time.Hour)),
			}, wantMethods: []string{http.MethodGet, http.MethodPatch},
			wantPatch: map[string]string{"undated-id": fromSecret},
		},
		{name: "all expired sends empty array", keys: []msgraph.KeyCredential{
			key("expired", "expired-id", time.Now().Add(-time.Hour)),
		}, wantMethods: []string{http.MethodGet, http.MethodPatch}, wantPatch: map[string]string{}},
	} {
		t.Run(test.name, func(t *testing.T) {
			requests, err := keyCredentialRequests(t, test.keys, inSecrets, func(client KeyCredential, tx transaction.Transaction) error {
				return client.DeleteExpired(tx)
			})
			require.NoError(t, err)
			require.Equal(t, test.wantMethods, requestMethods(requests))
			if test.wantPatch != nil {
				assert.Equal(t, test.wantPatch, patchedKeys(t, requests[1]))
			}
		})
	}
}

func TestDeleteUnusedSkipsPatchWhenNoKeysRevoked(t *testing.T) {
	requests, err := keyCredentialRequests(t, []msgraph.KeyCredential{
		key("azurerator-current", "current-id", time.Now().Add(time.Hour)),
		key("azurerator-next", "next-id", time.Now().Add(2*time.Hour)),
	}, nil, deleteUnused)
	require.NoError(t, err)
	assert.Equal(t, []string{http.MethodGet}, requestMethods(requests))
}

func TestDeleteUnusedKeepsOnlyKeysInUse(t *testing.T) {
	requests, err := keyCredentialRequests(t, []msgraph.KeyCredential{
		key("azurerator-current", "current-id", time.Now().Add(time.Hour)),
		key("azurerator-next", "next-id", time.Now().Add(2*time.Hour)),
		key("azurerator-unused", "unused-id", time.Now().Add(3*time.Hour)),
		key("azurerator-newest", "newest-id", time.Now().Add(4*time.Hour)),
	}, map[string][]byte{"current-id": pem, "next-id": pem, "unused-id": pem}, deleteUnused)
	require.NoError(t, err)
	require.Equal(t, []string{http.MethodGet, http.MethodPatch}, requestMethods(requests))
	assert.Equal(t, map[string]string{"current-id": fromSecret, "next-id": fromSecret}, patchedKeys(t, requests[1]))
}

func TestAddDropsKeysThatNoSecretHolds(t *testing.T) {
	var added *credentials.AddedKeyCredentialSet
	requests, err := keyCredentialRequests(t, []msgraph.KeyCredential{
		key("azurerator-managed", "managed-id", time.Now().Add(time.Hour)),
		key("azurerator-orphan", "orphan-id", time.Now().Add(2*time.Hour)),
	}, map[string][]byte{"managed-id": pem}, func(client KeyCredential, tx transaction.Transaction) error {
		var err error
		added, err = client.Add(tx)
		return err
	})
	require.NoError(t, err)
	require.Equal(t, []string{http.MethodGet, http.MethodPatch}, requestMethods(requests))
	patched := patchedKeys(t, requests[1])
	assert.Len(t, patched, 3)
	assert.Equal(t, fromSecret, patched["managed-id"])
	assert.NotContains(t, patched, "orphan-id")
	assert.Contains(t, patched, string(*added.Current.KeyCredential.KeyID))
	assert.Contains(t, patched, string(*added.Next.KeyCredential.KeyID))
}

func deleteUnused(client KeyCredential, tx transaction.Transaction) error {
	tx.Secrets.LatestCredentials.Set = &credentials.Set{
		Current: credentials.Credentials{Certificate: credentials.Certificate{KeyId: "current-id"}},
		Next:    credentials.Credentials{Certificate: credentials.Certificate{KeyId: "next-id"}},
	}
	return client.DeleteUnused(tx)
}

type keyCredentialRequest struct {
	Method string
	Body   map[string]any
}

func keyCredentialRequests(t *testing.T, existing []msgraph.KeyCredential, certificates map[string][]byte, run func(KeyCredential, transaction.Transaction) error) ([]keyCredentialRequest, error) {
	t.Helper()
	var requests []keyCredentialRequest
	runtime := fakemsgraph.NewRuntime(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if r.Method == http.MethodPatch {
			assert.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		}
		requests = append(requests, keyCredentialRequest{Method: r.Method, Body: body})
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		// Graph omits key bytes unless the request selects keyCredentials
		withoutKeys := append([]msgraph.KeyCredential(nil), existing...)
		for i := range withoutKeys {
			withoutKeys[i].Key = nil
		}
		appID := "client-id"
		app := msgraph.Application{AppID: &appID, KeyCredentials: withoutKeys}
		w.Header().Set("Content-Type", "application/json")
		assert.NoError(t, json.NewEncoder(w).Encode(map[string]any{"value": []msgraph.Application{app}}))
	}))
	instance := &v1.AzureAdApplication{Status: v1.AzureAdApplicationStatus{ObjectId: "application-id"}}
	instance.SetName("test-app")
	instance.SetNamespace("test")
	tx := transaction.Transaction{
		Ctx:         t.Context(),
		ClusterName: "test-cluster",
		Instance:    instance,
		Logger:      *log.NewEntry(log.New()),
		Secrets: transactionsecrets.Secrets{
			Certificates: certificates,
			KeyIDs:       credentials.KeyIDs{Used: credentials.KeyID{Certificate: []string{}}},
		},
	}
	err := run(NewKeyCredential(runtime), tx)
	return requests, err
}

func patchedKeys(t *testing.T, request keyCredentialRequest) map[string]string {
	t.Helper()
	require.Equal(t, http.MethodPatch, request.Method)
	keys, ok := request.Body["keyCredentials"].([]any)
	require.True(t, ok)
	patched := make(map[string]string, len(keys))
	for _, item := range keys {
		credential := item.(map[string]any)
		patched[credential["keyId"].(string)] = credential["key"].(string)
	}
	return patched
}

func requestMethods(requests []keyCredentialRequest) []string {
	methods := make([]string, len(requests))
	for i, request := range requests {
		methods[i] = request.Method
	}
	return methods
}

func key(name, id string, end time.Time) msgraph.KeyCredential {
	keyID := msgraph.UUID(id)
	keyBytes := msgraph.Binary("certificate-bytes")
	return msgraph.KeyCredential{DisplayName: &name, KeyID: &keyID, EndDateTime: &end, Key: &keyBytes}
}
