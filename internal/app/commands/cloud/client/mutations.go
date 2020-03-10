package client

import (
	"context"
	"fmt"

	"github.com/machinebox/graphql"
)

type CreateXp7ConfigResponse struct {
	CreateXp7Config string
}

func CreateXp7Config(ctx context.Context, solutionId string, appName string, config string) (*CreateXp7ConfigResponse, error) {
	// TODO: Actually send config
	req := graphql.NewRequest(fmt.Sprintf(`
	mutation {
		createXp7Config(solutionId: "%s", appName: "%s")
	}
	`, solutionId, appName))

	var res CreateXp7ConfigResponse
	err := doGraphQLRequest(ctx, req, &res)
	if err != nil {
		return nil, err
	}

	return &res, nil
}
