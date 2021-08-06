package application_test

import (
	"encoding/json"
	"testing"

	"github.com/nais/azureator/pkg/azure/client/application"
	v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	"github.com/nais/msgraph.go/ptr"
	"github.com/stretchr/testify/assert"
)

func TestRedirectUriApp(t *testing.T) {
	t.Run("web application, default", func(t *testing.T) {
		app := azureAdApp()
		a := application.RedirectUriApp(app)
		expected := `
{
  "web": {
    "redirectUris": [
      "https://test.host/callback"
    ]
  },
  "spa": {
    "redirectUris": []
  }
}
`
		assertJson(t, a, expected)
	})

	t.Run("web application, empty urls", func(t *testing.T) {
		app := azureAdApp()
		app.Spec.ReplyUrls = make([]v1.AzureAdReplyUrl, 0)

		a := application.RedirectUriApp(app)
		expected := `
{
  "web": {
    "redirectUris": []
  },
  "spa": {
    "redirectUris": []
  }
}
`
		assertJson(t, a, expected)
	})

	t.Run("single-page application", func(t *testing.T) {
		app := azureAdApp()
		app.Spec.SinglePageApplication = (*v1.AzureAdSinglePageApplication)(ptr.Bool(true))

		a := application.RedirectUriApp(app)
		expected := `
{
  "web": {
    "redirectUris": []
  },
  "spa": {
    "redirectUris": [
      "https://test.host/callback"
    ]
  }
}
`
		assertJson(t, a, expected)
	})

	t.Run("single-page application, empty urls", func(t *testing.T) {
		app := azureAdApp()
		app.Spec.SinglePageApplication = (*v1.AzureAdSinglePageApplication)(ptr.Bool(true))
		app.Spec.ReplyUrls = make([]v1.AzureAdReplyUrl, 0)

		a := application.RedirectUriApp(app)
		expected := `
{
  "web": {
    "redirectUris": []
  },
  "spa": {
    "redirectUris": []
  }
}
`
		assertJson(t, a, expected)
	})
}

func assertJson(t *testing.T, input interface{}, expected string) {
	j, _ := json.Marshal(input)
	assert.JSONEq(t, expected, string(j))
}

func azureAdApp() v1.AzureAdApplication {
	url := "https://test.host/callback"
	return v1.AzureAdApplication{
		Spec: v1.AzureAdApplicationSpec{
			ReplyUrls: []v1.AzureAdReplyUrl{
				{
					Url: url,
				},
			},
		},
	}
}
