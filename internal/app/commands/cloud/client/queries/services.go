package queries

import (
	"context"

	cloudApi "github.com/enonic/cli-enonic/internal/app/commands/cloud/client"
)

// GetServices gets all services for the logged in user
func GetServices(ctx context.Context) (*GetServicesData, error) {
	req := cloudApi.NewGQLRequest(`
	{
		account {
			clouds {
				name
				solutions {
					id
					name
					environments {
						name
						services {
							id
							name
							kind
						}
					}
				}
			}
		}
	}
	`)

	var res GetServicesData
	return &res, cloudApi.DoGraphQLRequest(ctx, req, &res)
}

type GetServicesData struct {
	Account Account `json:"account"`
}

type Account struct {
	Clouds []Cloud `json:"clouds"`
}

type Cloud struct {
	Name      string     `json:"name"`
	Solutions []Solution `json:"solutions"`
}

type Solution struct {
	ID           string        `json:"id"`
	Name         string        `json:"name"`
	Environments []Environment `json:"environments"`
}

type Environment struct {
	Name     string    `json:"name"`
	Services []Service `json:"services"`
}

type Service struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Kind string `json:"kind"`
}
