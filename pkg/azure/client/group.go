package client

import (
	"context"
	"fmt"

	msgraph "github.com/yaegashi/msgraph.go/v1.0"
)

type group struct {
	client
}

func (c client) group() group {
	return group{c}
}

func (g group) getOwnersFor(ctx context.Context, id string) ([]msgraph.DirectoryObject, error) {
	owners, err := g.graphClient.Groups().ID(id).Owners().Request().GetN(ctx, MaxNumberOfPagesToFetch)
	if err != nil {
		return owners, fmt.Errorf("failed to fetch owners for group: %w", err)
	}
	return owners, nil
}
