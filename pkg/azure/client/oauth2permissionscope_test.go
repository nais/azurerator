package client

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOAuth2PermissionScope_defaultScopes(t *testing.T) {
	a := application{}.oAuth2PermissionScopes().defaultScopes()
	j, _ := json.Marshal(a)

	assert.JSONEq(t, `
[
  {
    "adminConsentDescription": "Gives adminconsent for scope defaultaccess",
    "adminConsentDisplayName": "Adminconsent for scope defaultaccess",
    "id": "00000000-1337-d34d-b33f-000000000000",
    "isEnabled": true,
    "type": "User",
    "value": "defaultaccess"
  }
]
`, string(j))
}
