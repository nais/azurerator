package team

import (
	"github.com/nais/azureator/pkg/azure"
	"github.com/nais/azureator/pkg/azure/client/application"
	"github.com/nais/azureator/pkg/azure/client/group"
	"github.com/nais/azureator/pkg/azure/client/serviceprincipal"
	"github.com/nais/azureator/pkg/azure/transaction"
)

type Client interface {
	azure.RuntimeClient
	Application() application.Application
	AppRoleAssignments(tx transaction.Transaction, targetId azure.ObjectId) serviceprincipal.AppRoleAssignments
	Groups() group.Groups
	Team() Team
	ServicePrincipal() serviceprincipal.ServicePrincipal
}

type Team interface {
	Owners() Owners
	Groups() Groups
}

type team struct {
	Client
}

func NewTeam(client Client) Team {
	return team{Client: client}
}

func (t team) Owners() Owners {
	return newOwners(t.Client)
}

func (t team) Groups() Groups {
	return newGroups(t.Client)
}
