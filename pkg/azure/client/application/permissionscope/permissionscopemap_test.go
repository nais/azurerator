package permissionscope_test

import (
	"testing"

	"github.com/nais/msgraph.go/ptr"
	msgraph "github.com/nais/msgraph.go/v1.0"
	"github.com/stretchr/testify/assert"

	"github.com/nais/azureator/pkg/azure/client/application/permissionscope"
	"github.com/nais/azureator/pkg/azure/util/permissions"
)

func TestToMap(t *testing.T) {
	scopes := make([]msgraph.PermissionScope, 0)
	scope1 := permissionscope.NewGenerateId("scope1")
	scope2 := permissionscope.NewGenerateId("scope2")
	scope3 := permissionscope.NewGenerateId("scope3")
	scope3Duplicate := permissionscope.NewGenerateId("scope3")
	scopes = append(scopes, scope1)
	scopes = append(scopes, scope2)
	scopes = append(scopes, scope3)
	scopes = append(scopes, scope3Duplicate)

	scopeMap := permissionscope.ToMap(scopes)

	assert.Len(t, scopeMap, 3)
	assert.Equal(t, scope1, scopeMap["scope1"])
	assert.Equal(t, scope2, scopeMap["scope2"])
	assert.Equal(t, scope3, scopeMap["scope3"])
}

func TestMap_Add(t *testing.T) {
	scope1 := permissionscope.NewGenerateId("scope1")
	scope2 := permissionscope.NewGenerateId("scope2")
	scope3 := permissionscope.NewGenerateId("scope3")
	scope4 := permissionscope.NewGenerateId("scope4")

	scopeMap := make(permissionscope.Map)
	scopeMap.Add(scope1)
	scopeMap.Add(scope2)
	scopeMap.Add(scope3)
	scopeMap.Add(scope4)

	// duplicate scopes should not be added
	scope1Duplicate := permissionscope.NewGenerateId("scope1")
	scopeMap.Add(scope1Duplicate)

	assert.Len(t, scopeMap, 4)
	assert.Equal(t, scope1, scopeMap["scope1"])
	assert.Equal(t, scope2, scopeMap["scope2"])
	assert.Equal(t, scope3, scopeMap["scope3"])
	assert.Equal(t, scope4, scopeMap["scope4"])
}

func TestMap_ToSlice(t *testing.T) {
	scope1 := permissionscope.NewGenerateId("scope1")
	scope2 := permissionscope.NewGenerateId("scope2")
	scope3 := permissionscope.NewGenerateId("scope3")

	scopeMap := make(permissionscope.Map)
	scopeMap.Add(scope1)
	scopeMap.Add(scope2)
	scopeMap.Add(scope3)

	slice := scopeMap.ToSlice()
	assert.Contains(t, slice, scope1)
	assert.Contains(t, slice, scope2)
	assert.Contains(t, slice, scope3)
	assert.Len(t, slice, 3)
}

func TestMap_ToCreate(t *testing.T) {
	existing := make(permissionscope.Map)
	existing.Add(permissionscope.NewGenerateId("existing-scope-1"))
	existing.Add(permissionscope.NewGenerateId("existing-scope-2"))
	existing.Add(permissionscope.DefaultScope())

	desired := make(permissions.Permissions)
	desired.Add(permissions.NewGenerateIdEnabled("existing-scope-1"))
	desired.Add(permissions.NewGenerateIdEnabled("scope-2"))
	desired.Add(permissions.NewGenerateIdEnabled("scope-3"))

	toCreate := existing.ToCreate(desired)

	assert.Len(t, toCreate, 2)
	// should contain the new scopes to be created
	assert.Equal(t, permissionscope.FromPermission(desired["scope-2"]), toCreate["scope-2"])
	assert.Equal(t, permissionscope.FromPermission(desired["scope-3"]), toCreate["scope-3"])
	// should not contain default scope
	assert.Empty(t, toCreate[permissionscope.DefaultAccessScopeValue])
}

func TestMap_ToCreate_EmptyExisting_ShouldAddDefaultScope(t *testing.T) {
	existing := make(permissionscope.Map)

	desired := make(permissions.Permissions)
	desired.Add(permissions.NewGenerateIdEnabled("scope-1"))

	toCreate := existing.ToCreate(desired)
	assert.Len(t, toCreate, 2)
	// should contain the new scopes to be created
	assert.Equal(t, permissionscope.FromPermission(desired["scope-1"]), toCreate["scope-1"])
	// should contain default scope if not in existing
	assert.Equal(t, permissionscope.DefaultScope(), toCreate[permissionscope.DefaultAccessScopeValue])
}

func TestMap_ToDisable(t *testing.T) {
	existing := make(permissionscope.Map)
	existing.Add(permissionscope.NewGenerateId("existing-scope-1"))
	existing.Add(permissionscope.NewGenerateId("existing-scope-2"))

	desired := make(permissions.Permissions)
	desired.Add(permissions.NewGenerateIdEnabled("existing-scope-1"))
	desired.Add(permissions.NewGenerateIdEnabled("scope-2"))

	toDisable := existing.ToDisable(desired)

	assert.Len(t, toDisable, 1)

	// should contain non-desired scopes which should be disabled
	nonDesired := existing["existing-scope-2"]
	nonDesired.IsEnabled = ptr.Bool(false)

	assert.Equal(t, nonDesired, toDisable["existing-scope-2"])

	// should not disable default scope
	assert.NotContains(t, toDisable, permissionscope.DefaultScope())
}

func TestMap_Unmodified(t *testing.T) {
	existing := make(permissionscope.Map)
	existingScope1 := permissionscope.NewGenerateId("existing-scope-1")
	existingScope1.AdminConsentDescription = ptr.String("non standard description")
	existingScope1.UserConsentDescription = ptr.String("non standard description")
	existingScope1.AdminConsentDisplayName = ptr.String("non standard display name")
	existingScope1.UserConsentDisplayName = ptr.String("non standard display name")
	existingScope1.IsEnabled = ptr.Bool(false)
	existingScope1.Origin = ptr.String("some origin")
	existingScope1.Type = ptr.String("User")
	existing.Add(existingScope1)
	existing.Add(permissionscope.NewGenerateId("existing-scope-2"))

	desired := make(permissions.Permissions)
	desired.Add(permissions.NewGenerateIdEnabled("existing-scope-1"))
	desired.Add(permissions.NewGenerateIdEnabled("scope-2"))

	toCreate := existing.ToCreate(desired)
	toDisable := existing.ToDisable(desired)

	unmodified := existing.Unmodified(toCreate, toDisable)

	assert.Len(t, unmodified, 1)
	// should contain non-modified scopes
	assert.Equal(t, *existing["existing-scope-1"].ID, *unmodified["existing-scope-1"].ID)
	assert.Equal(t, "existing-scope-1", *unmodified["existing-scope-1"].Value)

	// unmodified scopes should conform to standard
	assert.Equal(t, "existing-scope-1", *unmodified["existing-scope-1"].AdminConsentDescription)
	assert.Equal(t, "existing-scope-1", *unmodified["existing-scope-1"].AdminConsentDisplayName)
	assert.True(t, *unmodified["existing-scope-1"].IsEnabled)
	assert.Nil(t, unmodified["existing-scope-1"].Origin)
	assert.Equal(t, "Admin", *unmodified["existing-scope-1"].Type)
	assert.Nil(t, unmodified["existing-scope-1"].UserConsentDescription)
	assert.Nil(t, unmodified["existing-scope-1"].UserConsentDisplayName)
}
