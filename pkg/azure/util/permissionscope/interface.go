package permissionscope

import (
	msgraph "github.com/nais/msgraph.go/v1.0"
	log "github.com/sirupsen/logrus"
)

type Result interface {
	GetResult() []msgraph.PermissionScope
	Log(logger log.Entry)
}

type CreateResult interface {
	Result

	GetCreate() Map
}

type UpdateResult interface {
	CreateResult

	GetDisable() Map
	GetUnmodified() Map
}
