package permissionscope_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/nais/msgraph.go/ptr"
	msgraph "github.com/nais/msgraph.go/v1.0"
	"github.com/stretchr/testify/assert"

	"github.com/nais/azureator/pkg/azure/client/application/permissionscope"
	"github.com/nais/azureator/pkg/azure/util"
	"github.com/nais/azureator/pkg/azure/util/permissions"
)

func TestNew(t *testing.T) {
	name := "scope1"
	id := msgraph.UUID(uuid.New().String())

	expected := msgraph.PermissionScope{
		AdminConsentDescription: ptr.String(name),
		AdminConsentDisplayName: ptr.String(name),
		ID:                      &id,
		IsEnabled:               ptr.Bool(true),
		Type:                    ptr.String(permissionscope.DefaultScopeType),
		Value:                   ptr.String(name),
	}
	actual := permissionscope.New(id, name)

	assert.Equal(t, expected, actual)
}

func TestNewGenerateId(t *testing.T) {
	name := "scope1"

	actual := permissionscope.NewGenerateId(name)
	id := actual.ID

	expected := msgraph.PermissionScope{
		AdminConsentDescription: ptr.String(name),
		AdminConsentDisplayName: ptr.String(name),
		ID:                      id,
		IsEnabled:               ptr.Bool(true),
		Type:                    ptr.String(permissionscope.DefaultScopeType),
		Value:                   ptr.String(name),
	}

	assert.Equal(t, expected, actual)
}

func TestDefaultScope(t *testing.T) {
	id := msgraph.UUID(permissionscope.DefaultAccessScopeId)
	expected := msgraph.PermissionScope{
		AdminConsentDescription: ptr.String(permissionscope.DefaultAccessScopeValue),
		AdminConsentDisplayName: ptr.String(permissionscope.DefaultAccessScopeValue),
		ID:                      &id,
		IsEnabled:               ptr.Bool(true),
		Type:                    ptr.String(permissionscope.DefaultScopeType),
		Value:                   ptr.String(permissionscope.DefaultAccessScopeValue),
	}
	actual := permissionscope.DefaultScope()

	assert.Equal(t, expected, actual)
}

func TestEnsureScopesRequireAdminConsent(t *testing.T) {
	scope1 := permissionscope.NewGenerateId("scope-1")
	scope1.Type = ptr.String("User")
	scope2 := permissionscope.NewGenerateId("scope-2")
	scope2.Type = ptr.String("User")

	scopes := []msgraph.PermissionScope{scope1, scope2}
	for _, scope := range scopes {
		assert.Equal(t, "User", *scope.Type)
	}

	actual := permissionscope.EnsureScopesRequireAdminConsent(scopes)
	for _, scope := range actual {
		assert.Equal(t, permissionscope.DefaultScopeType, *scope.Type)
	}
}

func TestEnsureDefaultAppRoleIsEnabled(t *testing.T) {
	defaultScope := permissionscope.DefaultScope()
	defaultScope.IsEnabled = ptr.Bool(false)

	scopes := []msgraph.PermissionScope{defaultScope}
	for _, scope := range scopes {
		assert.False(t, *scope.IsEnabled)
	}

	actual := permissionscope.EnsureDefaultScopeIsEnabled(scopes)
	for _, scope := range actual {
		assert.True(t, *scope.IsEnabled)
	}
}

func TestEnsureDefaultScopeIsEnabled(t *testing.T) {
	defaultScope := permissionscope.DefaultScope()
	defaultScope.IsEnabled = ptr.Bool(false)

	scopes := []msgraph.PermissionScope{defaultScope}
	for _, scope := range scopes {
		assert.False(t, *scope.IsEnabled)
	}

	actual := permissionscope.EnsureDefaultScopeIsEnabled(scopes)
	for _, scope := range actual {
		assert.True(t, *scope.IsEnabled)
	}
}

func TestFromPermission(t *testing.T) {
	permission := permissions.NewGenerateIdEnabled("scope")
	scope := permissionscope.FromPermission(permission)

	assert.Equal(t, "scope", *scope.AdminConsentDescription)
	assert.Equal(t, "scope", *scope.AdminConsentDisplayName)
	assert.Equal(t, "scope", *scope.Value)
	assert.Equal(t, permission.ID, *scope.ID)
}

func TestRemoveDisabled(t *testing.T) {
	enabledScope := permissionscope.NewGenerateId("enabled-scope")
	enabledScope2 := permissionscope.NewGenerateId("enabled-scope-2")
	disabledScope := permissionscope.NewGenerateId("disabled-scope")
	disabledScope.IsEnabled = ptr.Bool(false)

	scopes := []msgraph.PermissionScope{
		enabledScope,
		enabledScope2,
		disabledScope,
	}
	application := util.EmptyApplication().
		PermissionScopes(scopes).
		Build()

	desired := permissionscope.RemoveDisabled(*application)
	assert.Len(t, desired, 2)
	assert.Contains(t, desired, enabledScope)
	assert.Contains(t, desired, enabledScope2)
	assert.NotContains(t, desired, disabledScope)
}
