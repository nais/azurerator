package redirecturi_test

import (
	"encoding/json"
	"testing"

	"github.com/nais/azureator/pkg/azure/client/application/redirecturi"

	v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	"github.com/nais/msgraph.go/ptr"
	"github.com/stretchr/testify/assert"
)

func TestRedirectUriApp(t *testing.T) {
	t.Run("web application, default", func(t *testing.T) {
		app := azureAdApp()
		a := redirecturi.App(app)
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

		a := redirecturi.App(app)
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
		app.Spec.SinglePageApplication = ptr.Bool(true)

		a := redirecturi.App(app)
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
		app.Spec.SinglePageApplication = ptr.Bool(true)
		app.Spec.ReplyUrls = make([]v1.AzureAdReplyUrl, 0)

		a := redirecturi.App(app)
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

func TestGetReplyUrlsStringSlice(t *testing.T) {
	t.Run("Empty Application should return empty slice of reply URLs", func(t *testing.T) {
		p := &v1.AzureAdApplication{}
		actual := redirecturi.ReplyUrlsToStringSlice(p)
		assert.Empty(t, actual)
	})

	t.Run("Application with reply URL should return equivalent string slice of reply URLs", func(t *testing.T) {
		url := "https://test.host/callback"
		p := &v1.AzureAdApplication{Spec: v1.AzureAdApplicationSpec{ReplyUrls: []v1.AzureAdReplyUrl{{Url: v1.AzureAdReplyUrlString(url)}}}}
		actual := redirecturi.ReplyUrlsToStringSlice(p)
		assert.NotEmpty(t, actual)
		assert.Len(t, actual, 1)
		assert.Contains(t, actual, url)
	})

	t.Run("Application with duplicate reply URLs should return set of reply URLs", func(t *testing.T) {
		p := &v1.AzureAdApplication{Spec: v1.AzureAdApplicationSpec{
			ReplyUrls: []v1.AzureAdReplyUrl{
				{Url: "https://test.host/callback"},
				{Url: "https://test.host/callback"},
				{Url: "https://test.host/other-callback"},
				{Url: "https://test.host/other-callback"},
			},
		}}
		actual := redirecturi.ReplyUrlsToStringSlice(p)
		assert.NotEmpty(t, actual)
		assert.Len(t, actual, 2)
		assert.ElementsMatch(t, actual, []string{"https://test.host/callback", "https://test.host/other-callback"})
	})

	t.Run("Application with invalid URLs should return only valid URLs", func(t *testing.T) {
		p := &v1.AzureAdApplication{Spec: v1.AzureAdApplicationSpec{
			ReplyUrls: []v1.AzureAdReplyUrl{
				{Url: "https://test.host/callback"},
				{Url: "https://test.host/oauth2/callback"},
				{Url: "http://localhost/oauth2/callback"},
				{Url: "http://localhost:8080/oauth2/callback"},
				{Url: "http://127.0.0.1/oauth2/callback"},
				{Url: "http://127.0.0.1:8080/oauth2/callback"},
				{Url: "https://https://test.host/callback"},
				{Url: `https://test."host/other-callback"`},
			},
		}}
		actual := redirecturi.ReplyUrlsToStringSlice(p)
		assert.NotEmpty(t, actual)
		assert.Len(t, actual, 6)
		assert.ElementsMatch(t, actual, []string{
			"https://test.host/callback",
			"https://test.host/oauth2/callback",
			"http://localhost/oauth2/callback",
			"http://localhost:8080/oauth2/callback",
			"http://127.0.0.1/oauth2/callback",
			"http://127.0.0.1:8080/oauth2/callback",
		})
	})
}

func assertJson(t *testing.T, input any, expected string) {
	j, _ := json.Marshal(input)
	assert.JSONEq(t, expected, string(j))
}

func azureAdApp() *v1.AzureAdApplication {
	url := "https://test.host/callback"
	return &v1.AzureAdApplication{
		Spec: v1.AzureAdApplicationSpec{
			ReplyUrls: []v1.AzureAdReplyUrl{
				{
					Url: v1.AzureAdReplyUrlString(url),
				},
			},
		},
	}
}
