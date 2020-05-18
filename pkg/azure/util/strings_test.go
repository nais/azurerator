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
	ti := time.Date(2000, 1, 1, 8, 0, 0, 0, time.UTC)
	actual := DisplayName(ti)
	assert.Equal(t, "azurerator-2000-01-01T08:00:00Z", actual)
}

func TestGetReplyUrlsStringSlice(t *testing.T) {
	p := v1alpha1.AzureAdApplication{}
	actual := GetReplyUrlsStringSlice(p)
	assert.Empty(t, actual)

	url := "http://test.host/callback"
	p = v1alpha1.AzureAdApplication{Spec: v1alpha1.AzureAdApplicationSpec{ReplyUrls: []v1alpha1.AzureAdReplyUrl{{Url: url}}}}
	actual = GetReplyUrlsStringSlice(p)
	assert.NotEmpty(t, actual)
	assert.Len(t, actual, 1)
	assert.Contains(t, actual, url)
}

func TestFilterByAppId(t *testing.T) {
	p := "test"
	actual := FilterByAppId(p)
	assert.Equal(t, fmt.Sprintf("appId eq '%s'", p), actual)
}

func TestFilterByClientId(t *testing.T) {
	p := "test"
	actual := FilterByClientId(p)
	assert.Equal(t, fmt.Sprintf("clientId eq '%s'", p), actual)
}

func TestFilterByName(t *testing.T) {
	p := "test"
	actual := FilterByName(p)
	assert.Equal(t, fmt.Sprintf("displayName eq '%s'", p), actual)
}

func TestIdentifierUri(t *testing.T) {
	p := "some-uuid"
	actual := IdentifierUri(p)
	assert.Equal(t, fmt.Sprintf("api://%s", p), actual)
}

func TestMapFiltersToFilter(t *testing.T) {
	name := FilterByName("some-name")
	appid := FilterByAppId("some-appid")

	p := []azure.Filter{name, appid}
	actual := MapFiltersToFilter(p)
	assert.Equal(t, fmt.Sprintf("%s %s", name, appid), actual)

	p = []azure.Filter{}
	actual = MapFiltersToFilter(p)
	assert.Empty(t, actual)
}
