package queries

import (
	"context"

	cloudApi "github.com/enonic/cli-enonic/internal/app/commands/cloud/client"
	"github.com/machinebox/graphql"
)

// GetSolutions gets all solutions for the logged in user
func GetSolutions(ctx context.Context) (*Solutions, error) {
	req := graphql.NewRequest(`
	{
		account {
			clouds {
				name
				projects {
					name
					solutions {
						id
						name
					}
				}
			}
		}
	}
	`)

	var res Solutions
	return &res, cloudApi.DoGraphQLRequest(ctx, req, &res)
}

type Solutions struct {
	Account Account `json:"account"`
}

type Account struct {
	Clouds []Cloud `json:"clouds"`
}

type Cloud struct {
	Name     string    `json:"name"`
	Projects []Project `json:"projects"`
}

type Project struct {
	Name      string     `json:"name"`
	Solutions []Solution `json:"solutions"`
}

type Solution struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}
