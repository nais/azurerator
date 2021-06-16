package application_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/nais/msgraph.go/ptr"
	msgraph "github.com/nais/msgraph.go/v1.0"
	"github.com/stretchr/testify/assert"

	"github.com/nais/azureator/pkg/azure/client/application"
	"github.com/nais/azureator/pkg/azure/util/permissions"
	"github.com/nais/azureator/pkg/azure/util/permissionscope"
)

func TestPermissionScopes_DescribeCreate_DesiredIsEmpty_ShouldAddDefaultScope(t *testing.T) {
	desired := make(permissions.Permissions)
	scopes := application.Application{}.OAuth2PermissionScopes().DescribeCreate(desired).GetResult()

	assert.Len(t, scopes, 1)
	assertContainsDefaultScope(t, scopes)
}

func TestPermissionScopes_DescribeCreate_DisabledDefaultScopeInDesired_ShouldNotDisable(t *testing.T) {
	desired := make(permissions.Permissions)
	defaultScope := permissionscope.DefaultScope()
	defaultScope.IsEnabled = ptr.Bool(false)
	desired.Add(permissions.FromPermissionScope(defaultScope))

	scopes := application.Application{}.OAuth2PermissionScopes().DescribeCreate(desired).GetResult()

	assert.Len(t, scopes, 1)
	assertContainsDefaultScope(t, scopes)
}

func TestPermissionScopes_DescribeCreate_CustomScopes_ShouldAddCustomScopeAndDefaultScope(t *testing.T) {
	scope1 := permissionscope.NewGenerateId("scope-1")
	scope2 := permissionscope.NewGenerateId("scope-2")

	desired := make(permissions.Permissions)
	desired.Add(permissions.FromPermissionScope(scope1))
	desired.Add(permissions.FromPermissionScope(scope2))

	scopes := application.Application{}.OAuth2PermissionScopes().DescribeCreate(desired).GetResult()

	assertContainsScope(t, scope1, scopes)
	assertContainsScope(t, scope2, scopes)
	assertContainsDefaultScope(t, scopes)

	// assert that length of PermissionScopes equals length of the resulting union set of (desired + Default PermissionScope)
	assert.Len(t, scopes, len(desired)+1)
}

func TestPermissionScopes_DescribeUpdate_DefaultRoleNotExistsInDesiredNorExisting_ShouldAdd(t *testing.T) {
	desired := make(permissions.Permissions)
	existing := make([]msgraph.PermissionScope, 0)
	scopes := application.Application{}.OAuth2PermissionScopes().DescribeUpdate(desired, existing).GetResult()

	assert.Len(t, scopes, 1)
	assertContainsDefaultScope(t, scopes)
}

func TestPermissionScopes_DescribeUpdate_DefaultScopeAlreadyExists_ShouldOnlyUpdateTypeAndEnable(t *testing.T) {
	desired := make(permissions.Permissions)
	existing := []msgraph.PermissionScope{
		permissionscope.DefaultScope(),
	}
	scopes := application.Application{}.OAuth2PermissionScopes().DescribeUpdate(desired, existing).GetResult()

	assert.Len(t, scopes, 1)
	assertContainsDefaultScope(t, scopes)

	// test case where the default scope was previously created by another entity with differing values in the fields
	id := msgraph.UUID(uuid.New().String())
	existing = []msgraph.PermissionScope{
		{
			Type:                    ptr.String("User"),
			AdminConsentDescription: ptr.String("Description"),
			AdminConsentDisplayName: ptr.String("DisplayName"),
			ID:                      &id,
			IsEnabled:               ptr.Bool(false),
			Value:                   ptr.String(permissionscope.DefaultAccessScopeValue),
		},
	}
	scopes = application.Application{}.OAuth2PermissionScopes().DescribeUpdate(desired, existing).GetResult()

	assert.Len(t, scopes, 1)
	assertContainsDefaultScopeWithLambda(t, scopes, func(t assert.TestingT, expected, actual msgraph.PermissionScope) {
		assert.Equal(t, *actual.AdminConsentDescription, permissionscope.DefaultAccessScopeValue)
		assert.Equal(t, *actual.AdminConsentDisplayName, permissionscope.DefaultAccessScopeValue)
		assert.Equal(t, *actual.ID, id)
		assert.True(t, *actual.IsEnabled)
		assert.Equal(t, *actual.Type, permissionscope.DefaultScopeType)
		assert.Equal(t, *actual.Value, permissionscope.DefaultAccessScopeValue)
	})
}

func TestPermissionScopes_DescribeUpdate_DisabledDefaultScopeInDesired_ShouldNotDisable(t *testing.T) {
	desired := make(permissions.Permissions)
	desired.Add(permissions.NewGenerateIdDisabled(permissionscope.DefaultAccessScopeValue))

	existing := make([]msgraph.PermissionScope, 0)
	scopes := application.Application{}.OAuth2PermissionScopes().DescribeUpdate(desired, existing).GetResult()

	assert.Len(t, scopes, 1)
	assertContainsDefaultScope(t, scopes)
}

func TestPermissionScopes_DescribeUpdate_CustomScopes_ShouldAddDesiredScopesAndRemoveNonDesiredScopes(t *testing.T) {
	scope1 := permissionscope.NewGenerateId("scope-1")
	scope2 := permissionscope.NewGenerateId("scope-2")
	scope3 := permissionscope.NewGenerateId("scope-3")

	desired := make(permissions.Permissions)
	desired.Add(permissions.FromPermissionScope(scope1))
	desired.Add(permissions.FromPermissionScope(scope2))
	desired.Add(permissions.NewGenerateIdDisabled(permissionscope.DefaultAccessScopeValue))

	existing := []msgraph.PermissionScope{
		permissionscope.DefaultScope(),
		scope1,
		scope3,
	}

	scopes := application.Application{}.OAuth2PermissionScopes().DescribeUpdate(desired, existing).GetResult()

	// assert that scope "scope-1" still exists and is unmodified
	assertContainsScope(t, scope1, scopes)
	// assert that scope "scope-2" is added
	assertContainsScope(t, scope2, scopes)
	// assert that the default scope still exists and is unmodified
	assertContainsDefaultScope(t, scopes)

	// assert that the non-desired scope "scope-3" still exists and is set to disabled
	assertContainsScopeWithLambda(t, scope3, scopes, func(t assert.TestingT, expected, actual msgraph.PermissionScope) {
		assert.Equal(t, *expected.Value, *actual.Value)
		assert.Equal(t, *expected.ID, *actual.ID)
		assert.Equal(t, *expected.Value, *actual.AdminConsentDisplayName)
		assert.Equal(t, *expected.Value, *actual.AdminConsentDescription)
		assert.False(t, *actual.IsEnabled)
		assert.Equal(t, *actual.Type, permissionscope.DefaultScopeType)
	})

	// assert that length of PermissionScopes equals length of the resulting union set of (existing + desired)
	assert.Len(t, scopes, len(desired)+1)
}

func defaultScopeAsserter() func(t assert.TestingT, expected, actual msgraph.PermissionScope) {
	return func(t assert.TestingT, expected, actual msgraph.PermissionScope) {
		assert.Equal(t, *expected.Value, *actual.Value)
		assert.Equal(t, *expected.ID, *actual.ID)
		assert.Equal(t, *expected.Value, *actual.AdminConsentDisplayName)
		assert.Equal(t, *expected.Value, *actual.AdminConsentDescription)
		assert.True(t, *actual.IsEnabled)
		assert.Equal(t, *actual.Type, permissionscope.DefaultScopeType)
	}
}

func assertContainsScope(t assert.TestingT, expected msgraph.PermissionScope, scopes []msgraph.PermissionScope) {
	assertContainsScopeWithLambda(t, expected, scopes, defaultScopeAsserter())
}

func assertContainsScopeWithLambda(t assert.TestingT, expected msgraph.PermissionScope, scopes []msgraph.PermissionScope, validatingFunc func(t assert.TestingT, expected, actual msgraph.PermissionScope)) {
	found := false
	for _, scope := range scopes {
		if *scope.Value == *expected.Value {
			found = true
			validatingFunc(t, expected, scope)
			break
		}
	}
	assert.True(t, found)
}

func assertContainsDefaultScope(t assert.TestingT, scopes []msgraph.PermissionScope) {
	assertContainsDefaultScopeWithLambda(t, scopes, defaultScopeAsserter())
}

func assertContainsDefaultScopeWithLambda(t assert.TestingT, scopes []msgraph.PermissionScope, validatingFunc func(t assert.TestingT, expected, actual msgraph.PermissionScope)) {
	assertContainsScopeWithLambda(t, permissionscope.DefaultScope(), scopes, validatingFunc)
}
