package mutations

import (
	"context"
	"fmt"

	cloudApi "github.com/enonic/cli-enonic/internal/app/commands/cloud/client"
)

// CreateXp7AppFromUpload deploys jar to a service
func CreateXp7AppFromUpload(ctx context.Context, serviceID string, jarID string) error {
	req := cloudApi.NewGQLRequest(fmt.Sprintf(`
	mutation {
		createXp7AppFromUpload(serviceId: "%s", jarId: "%s") {
			id
		}
	}
	`, serviceID, jarID))

	var res map[string]interface{}
	return cloudApi.DoGraphQLRequest(ctx, req, &res)
}
