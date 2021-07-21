package permissionscope

import (
	msgraph "github.com/nais/msgraph.go/v1.0"
	log "github.com/sirupsen/logrus"
)

type Result interface {
	GetResult() []msgraph.PermissionScope
	Log(logger log.Entry)
}

type createResult struct {
	toCreate Map
}

func NewCreateResult(toCreate Map) Result {
	return createResult{toCreate: toCreate}
}

func (a createResult) GetResult() []msgraph.PermissionScope {
	return a.toCreate.ToSlice()
}

func (a createResult) Log(logger log.Entry) {
	a.toCreate.ToPermissionList().Log(logger, "creating desired scopes")
}

type updateResult struct {
	toCreate   Map
	toDisable  Map
	unmodified Map
	result     []msgraph.PermissionScope
}

func NewUpdateResult(toCreate Map, toDisable Map, unmodified Map, result []msgraph.PermissionScope) Result {
	return updateResult{
		toCreate:   toCreate,
		toDisable:  toDisable,
		unmodified: unmodified,
		result:     result,
	}
}

func (a updateResult) GetResult() []msgraph.PermissionScope {
	return a.result
}

func (a updateResult) Log(logger log.Entry) {
	a.toCreate.ToPermissionList().Log(logger, "creating desired scopes")
	a.toDisable.ToPermissionList().Log(logger, "disabling non-desired scopes")
	a.unmodified.ToPermissionList().Log(logger, "unmodified scopes")
}
