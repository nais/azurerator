package team

import (
	"github.com/nais/azureator/pkg/azure"
)

type team struct {
	azure.RuntimeClient
}

func NewTeam(runtimeClient azure.RuntimeClient) azure.Team {
	return team{RuntimeClient: runtimeClient}
}

func (t team) Owners() azure.TeamOwners {
	return newOwners(t.RuntimeClient)
}

func (t team) Groups() azure.TeamGroups {
	return newGroups(t.RuntimeClient)
}
