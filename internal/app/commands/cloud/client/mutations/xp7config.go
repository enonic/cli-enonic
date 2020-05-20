package mutations

import (
	"context"
	"fmt"

	cloudApi "github.com/enonic/cli-enonic/internal/app/commands/cloud/client"
	"github.com/machinebox/graphql"
)

type CreateXp7ConfigData struct {
	CreateXp7Config CreateXp7Config `json:"createXp7Config"`
}

type CreateXp7Config struct {
	Token string `json:"token"`
}

// CreateXp7Config starts the flow of uploading an app to the Cloud API
func CreateXp7ConfigRequest(ctx context.Context, serviceId string, appName string, nodeId string, configFile string) (*CreateXp7ConfigData, error) {
	// TODO: Actually send config

	req := graphql.NewRequest(fmt.Sprintf(`
	mutation {
		createXp7Config(serviceId: "%s", appName: "%s", nodeId: "%s") {
			token
		}
	}
	`, serviceId, appName, nodeId))

	var res CreateXp7ConfigData
	return &res, cloudApi.DoGraphQLRequest(ctx, req, &res)
}
