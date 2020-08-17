package client

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRequiredResourceAccess_microsoftGraph(t *testing.T) {
	a := client{}.requiredResourceAccess().microsoftGraph()
	j, _ := json.Marshal(a)

	assert.JSONEq(t, `
{
   "resourceAppId": "00000003-0000-0000-c000-000000000000",
   "resourceAccess": [
      {
         "id": "e1fe6dd8-ba31-4d61-89e7-88639da4683d",
         "type": "Scope"
      },
      {
         "id": "37f7f235-527c-4136-accd-4a02d197296e",
         "type": "Scope"
      },
      {
         "id": "bc024368-1153-4739-b217-4326f2e966d0",
         "type": "Scope"
      }
   ]
}
`, string(j))
}
