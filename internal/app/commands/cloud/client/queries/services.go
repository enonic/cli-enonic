package queries

import (
	"context"

	cloudApi "cli-enonic/internal/app/commands/cloud/client"
)

// GetServices gets all services for the logged in user
func GetServices(ctx context.Context) (*GetServicesData, error) {
	req := cloudApi.NewGQLRequest(`
	{
		search(params: {query: "type = 'account'"}) {
			accounts {
				name
				plan
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
	AccountsSearch AccountsSearch `json:"search"`
}

type AccountsSearch struct {
	Accounts []Account `json:"accounts"`
}

type Account struct {
	Name      string     `json:"name"`
	Plan      string     `json:"plan"`
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
