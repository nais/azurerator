package approle

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

func (a createResult) GetResult() []msgraph.AppRole {
	return a.GetCreate().ToSlice()
}

func (a createResult) Log(logger log.Entry) {
	a.GetCreate().ToPermissionList().Log(logger, "creating desired roles")
}

type updateResult struct {
	toCreate   Map
	toDisable  Map
	unmodified Map
	result     []msgraph.AppRole
}

func NewUpdateResult(toCreate Map, toDisable Map, unmodified Map, result []msgraph.AppRole) UpdateResult {
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

func (a updateResult) GetResult() []msgraph.AppRole {
	return a.result
}

func (a updateResult) Log(logger log.Entry) {
	a.GetCreate().ToPermissionList().Log(logger, "creating desired roles")
	a.GetDisable().ToPermissionList().Log(logger, "disabling non-desired roles")
	a.GetUnmodified().ToPermissionList().Log(logger, "unmodified roles")
}
