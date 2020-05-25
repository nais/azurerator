package util

import (
	"fmt"
	"testing"
	"time"

	"github.com/nais/azureator/api/v1alpha1"
	"github.com/nais/azureator/pkg/azure"
	"github.com/stretchr/testify/assert"
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
		p := v1alpha1.AzureAdApplication{}
		actual := GetReplyUrlsStringSlice(p)
		assert.Empty(t, actual)
	})

	t.Run("Application with reply URL should return equivalent string slice of reply URLs", func(t *testing.T) {
		url := "http://test.host/callback"
		p := v1alpha1.AzureAdApplication{Spec: v1alpha1.AzureAdApplicationSpec{ReplyUrls: []v1alpha1.AzureAdReplyUrl{{Url: url}}}}
		actual := GetReplyUrlsStringSlice(p)
		assert.NotEmpty(t, actual)
		assert.Len(t, actual, 1)
		assert.Contains(t, actual, url)
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

func TestIdentifierUri(t *testing.T) {
	t.Run("Given a UUID, the Identifier URI should be a formatted string following a template", func(t *testing.T) {
		p := "some-uuid"
		actual := IdentifierUri(p)
		expected := fmt.Sprintf("api://%s", p)
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
