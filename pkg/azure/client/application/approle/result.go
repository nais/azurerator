package approle

import (
	msgraph "github.com/nais/msgraph.go/v1.0"
	log "github.com/sirupsen/logrus"
)

type Result interface {
	GetResult() []msgraph.AppRole
	Log(logger log.Entry)
}

type createResult struct {
	toCreate Map
}

func NewCreateResult(toCreate Map) Result {
	return createResult{toCreate: toCreate}
}

func (a createResult) GetResult() []msgraph.AppRole {
	return a.toCreate.ToSlice()
}

func (a createResult) Log(logger log.Entry) {
	a.toCreate.ToPermissionList().Log(logger, "creating desired roles")
}

type updateResult struct {
	toCreate   Map
	toDisable  Map
	unmodified Map
	result     []msgraph.AppRole
}

func NewUpdateResult(toCreate Map, toDisable Map, unmodified Map, result []msgraph.AppRole) Result {
	return updateResult{
		toCreate:   toCreate,
		toDisable:  toDisable,
		unmodified: unmodified,
		result:     result,
	}
}

func (a updateResult) GetResult() []msgraph.AppRole {
	return a.result
}

func (a updateResult) Log(logger log.Entry) {
	a.toCreate.ToPermissionList().Log(logger, "creating desired roles")
	a.toDisable.ToPermissionList().Log(logger, "disabling non-desired roles")
	a.unmodified.ToPermissionList().Log(logger, "unmodified roles")
}
