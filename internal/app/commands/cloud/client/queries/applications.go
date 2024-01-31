package queries

import (
	"context"
	"fmt"

	cloudApi "cli-enonic/internal/app/commands/cloud/client"
)

// GetApplications gets all xp7Apps in service by ID for the logged in user
func GetApplications(ctx context.Context, serviceId string) (*GetAppsData, error) {
	req := cloudApi.NewGQLRequest(fmt.Sprintf(`
	{
		search(params: {query: "type = 'CRD' AND kind = 'Xp7App' AND serviceId = '%v'"}) {
			xp7Applications {
				id
				image {
					appName
				}
			}
		}
	}
	`, serviceId))

	var res GetAppsData
	return &res, cloudApi.DoGraphQLRequest(ctx, req, &res)
}

type GetAppsData struct {
	AppsSearch AppsSearch `json:"search"`
}

type AppsSearch struct {
	Applications []Application `json:"xp7Applications"`
}

type Application struct {
	ID    string `json:"id"`
	Image Image  `json:"image"`
}

type Image struct {
	ID      string `json:"id"`
	AppName string `json:"appName"`
}
