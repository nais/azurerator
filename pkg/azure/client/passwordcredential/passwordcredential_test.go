package passwordcredential

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"

	v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	msgraph "github.com/nais/msgraph.go/v1.0"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"

	"github.com/nais/azureator/pkg/azure/credentials"
	fakemsgraph "github.com/nais/azureator/pkg/azure/fake/msgraph"
	"github.com/nais/azureator/pkg/transaction"
)

func TestAddRetriesWithSamePayload(t *testing.T) {
	var bodies [][]byte
	attempt := 0
	runtime := fakemsgraph.NewRuntime(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/v1.0/applications/application-id/addPassword", r.URL.Path)
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		bodies = append(bodies, body)
		attempt++
		if attempt == 1 {
			writeConcurrencyError(w)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"keyId":"result-id","secretText":"secret"}`))
	}))

	tx := passwordCredentialTransaction(t)
	result, err := NewPasswordCredential(runtime).Add(tx)
	require.NoError(t, err)
	require.Equal(t, "result-id", string(*result.KeyID))
	require.Len(t, bodies, 2)
	require.JSONEq(t, string(bodies[0]), string(bodies[1]))
	var payload msgraph.ApplicationAddPasswordRequestParameter
	require.NoError(t, json.Unmarshal(bodies[0], &payload))
	require.NotNil(t, payload.PasswordCredential.KeyID)
}

func TestRemoveVerifiesCredentialAfterMutationError(t *testing.T) {
	for _, test := range []struct {
		name        string
		credentials []msgraph.PasswordCredential
		getStatus   int
		wantErr     bool
	}{
		{name: "still present", credentials: []msgraph.PasswordCredential{password("key-id")}, wantErr: true},
		{name: "already removed"},
		{name: "verification read fails", getStatus: http.StatusForbidden, wantErr: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			methods := make([]string, 0)
			runtime := fakemsgraph.NewRuntime(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				methods = append(methods, r.Method)
				w.Header().Set("Content-Type", "application/json")
				switch {
				case r.Method == http.MethodPost:
					w.WriteHeader(http.StatusInternalServerError)
					_, _ = w.Write([]byte(`{"error":{"code":"InternalError","message":"mutation failed"}}`))
				case test.getStatus != 0:
					w.WriteHeader(test.getStatus)
					_, _ = w.Write([]byte(`{"error":{"code":"Forbidden","message":"read failed"}}`))
				default:
					appID := "client-id"
					app := msgraph.Application{AppID: &appID, PasswordCredentials: test.credentials}
					_ = json.NewEncoder(w).Encode(map[string]any{"value": []msgraph.Application{app}})
				}
			}))

			tx := passwordCredentialTransaction(t)
			keyID := msgraph.UUID("key-id")
			err := passwordCredential{Client: runtime}.remove(tx, "application-id", &keyID)
			require.Equal(t, []string{http.MethodPost, http.MethodGet}, methods)
			if !test.wantErr {
				require.NoError(t, err)
				return
			}
			var graphErr *msgraph.ErrorResponse
			require.ErrorAs(t, err, &graphErr)
			require.Equal(t, "mutation failed", graphErr.ErrorObject.Message)
		})
	}
}

func writeConcurrencyError(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusInternalServerError)
	_, _ = w.Write([]byte(`{"error":{"code":"Request_BadRequest","message":"` + credentials.ConcurrentRequestMessage + `"}}`))
}

func password(keyID string) msgraph.PasswordCredential {
	id := msgraph.UUID(keyID)
	return msgraph.PasswordCredential{KeyID: &id}
}

func passwordCredentialTransaction(t *testing.T) transaction.Transaction {
	t.Helper()
	return transaction.Transaction{
		Ctx:      t.Context(),
		Instance: &v1.AzureAdApplication{Status: v1.AzureAdApplicationStatus{ObjectId: "application-id"}},
		Logger:   *log.NewEntry(log.New()),
	}
}
