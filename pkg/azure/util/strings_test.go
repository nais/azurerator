package util

import (
	"fmt"
	"testing"
	"time"

	v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	"github.com/stretchr/testify/assert"

	"github.com/nais/azureator/pkg/azure"
)

func TestDisplayName(t *testing.T) {
	t.Run("DisplayName should return string with formatted timestamp", func(t *testing.T) {
		ti := time.Date(2000, 1, 1, 8, 0, 0, 0, time.UTC)
		actual := DisplayName(ti)
		assert.Equal(t, "azurerator-2000-01-01T08:00:00Z", actual)
	})
}

func TestGetReplyUrlsStringSlice(t *testing.T) {
	t.Run("Empty Application should return empty slice of reply URLs", func(t *testing.T) {
		p := v1.AzureAdApplication{}
		actual := GetReplyUrlsStringSlice(p)
		assert.Empty(t, actual)
	})

	t.Run("Application with reply URL should return equivalent string slice of reply URLs", func(t *testing.T) {
		url := "http://test.host/callback"
		p := v1.AzureAdApplication{Spec: v1.AzureAdApplicationSpec{ReplyUrls: []v1.AzureAdReplyUrl{{Url: url}}}}
		actual := GetReplyUrlsStringSlice(p)
		assert.NotEmpty(t, actual)
		assert.Len(t, actual, 1)
		assert.Contains(t, actual, url)
	})

	t.Run("Application with duplicate reply URLs should return set of reply URLs", func(t *testing.T) {
		p := v1.AzureAdApplication{Spec: v1.AzureAdApplicationSpec{
			ReplyUrls: []v1.AzureAdReplyUrl{
				{Url: "https://test.host/callback"},
				{Url: "https://test.host/callback"},
				{Url: "https://test.host/other-callback"},
				{Url: "https://test.host/other-callback"},
			},
		}}
		actual := GetReplyUrlsStringSlice(p)
		assert.NotEmpty(t, actual)
		assert.Len(t, actual, 2)
		assert.ElementsMatch(t, actual, []string{"https://test.host/callback", "https://test.host/other-callback"})
	})
}

func TestFilters(t *testing.T) {
	p := "test"
	cases := []struct {
		name     string
		fn       func(string) string
		expected string
	}{
		{
			name:     "Filter by AppId",
			fn:       FilterByAppId,
			expected: fmt.Sprintf("appId eq '%s'", p),
		},
		{
			name:     "Filter by Client ID",
			fn:       FilterByClientId,
			expected: fmt.Sprintf("clientId eq '%s'", p),
		},
		{
			name:     "Filter by DisplayName",
			fn:       FilterByName,
			expected: fmt.Sprintf("displayName eq '%s'", p),
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			actual := c.fn(p)
			assert.Equal(t, c.expected, actual)
		})
	}
}

func TestIdentifierUriClientId(t *testing.T) {
	t.Run("Given a UUID, the Identifier URI should be a formatted string following a template", func(t *testing.T) {
		p := "some-uuid"
		actual := IdentifierUriClientId(p)
		expected := "api://some-uuid"
		assert.Equal(t, expected, actual)
	})
}

func TestIdentifierUriHumanReadable(t *testing.T) {
	t.Run("Given an Application spec, the Identifier URI should be a formatted string following a template", func(t *testing.T) {
		spec := v1.AzureAdApplication{}
		spec.SetName("test")
		spec.SetNamespace("test-namespace")
		spec.SetClusterName("test-cluster")
		actual := IdentifierUriHumanReadable(spec)
		expected := "api://test-cluster.test-namespace.test"
		assert.Equal(t, expected, actual)
	})
}

func TestMapFiltersToFilter(t *testing.T) {
	t.Run("Empty slice of filters should return empty string", func(t *testing.T) {
		p := make([]azure.Filter, 0)
		actual := MapFiltersToFilter(p)
		assert.Empty(t, actual)
	})

	t.Run("Multiple filters should return concatenated string of filters", func(t *testing.T) {
		name := FilterByName("some-name")
		appid := FilterByAppId("some-appid")

		p := []azure.Filter{name, appid}
		actual := MapFiltersToFilter(p)
		assert.Equal(t, fmt.Sprintf("%s %s", name, appid), actual)
	})
}
