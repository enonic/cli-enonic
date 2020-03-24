package mutations

import (
	"context"
	"fmt"

	cloudApi "github.com/enonic/cli-enonic/internal/app/commands/cloud/client"
	"github.com/machinebox/graphql"
)

type CreateXp7ConfigResponse struct {
	CreateXp7Config string
}

// CreateXp7Config starts the flow of uploading an app to the Cloud API
func CreateXp7Config(ctx context.Context, solutionId string, appName string, configFile string) (*CreateXp7ConfigResponse, error) {
	// TODO: Actually send config

	req := graphql.NewRequest(fmt.Sprintf(`
	mutation {
		createXp7Config(solutionId: "%s", appName: "%s")
	}
	`, solutionId, appName))

	var res CreateXp7ConfigResponse
	return &res, cloudApi.DoGraphQLRequest(ctx, req, &res)
}
