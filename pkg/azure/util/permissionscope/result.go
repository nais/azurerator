package permissionscope

import (
	msgraph "github.com/nais/msgraph.go/v1.0"
	log "github.com/sirupsen/logrus"
)

type createResult struct {
	toCreate Map
}

func NewCreateResult(toCreate Map) CreateResult {
	return createResult{toCreate: toCreate}
}

func (a createResult) GetCreate() Map {
	return a.toCreate
}

func (a createResult) GetResult() []msgraph.PermissionScope {
	return a.toCreate.ToSlice()
}

func (a createResult) Log(logger log.Entry) {
	a.GetCreate().ToPermissionList().Log(logger, "creating desired scopes")
}

type updateResult struct {
	toCreate   Map
	toDisable  Map
	unmodified Map
	result     []msgraph.PermissionScope
}

func NewUpdateResult(toCreate Map, toDisable Map, unmodified Map, result []msgraph.PermissionScope) UpdateResult {
	return updateResult{
		toCreate:   toCreate,
		toDisable:  toDisable,
		unmodified: unmodified,
		result:     result,
	}
}

func (a updateResult) GetCreate() Map {
	return a.toCreate
}

func (a updateResult) GetDisable() Map {
	return a.toDisable
}

func (a updateResult) GetUnmodified() Map {
	return a.unmodified
}

func (a updateResult) GetResult() []msgraph.PermissionScope {
	return a.result
}

func (a updateResult) Log(logger log.Entry) {
	a.GetCreate().ToPermissionList().Log(logger, "creating desired scopes")
	a.GetDisable().ToPermissionList().Log(logger, "disabling non-desired scopes")
	a.GetUnmodified().ToPermissionList().Log(logger, "unmodified scopes")
}
