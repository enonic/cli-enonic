package mutations

import (
	"context"
	"fmt"

	cloudApi "cli-enonic/internal/app/commands/cloud/client"
)

type ErrorData struct {
	Errors []string `json:"errors"`
}

type CreateAppData struct {
	Data ErrorData `json:"createXp7Application"`
}

type UpdateAppData struct {
	Data ErrorData `json:"updateXp7Application"`
}

// CreateXp7App creates new application on service that uses the uploaded jar
func CreateXp7App(ctx context.Context, serviceID string, jarID string) error {
	req := cloudApi.NewGQLRequest(fmt.Sprintf(`
	mutation {
		createXp7Application(params: {serviceId: "%s", imageId: "%s"}) {
			application {
				id
			}
			errors
		}
	}
	`, serviceID, jarID))

	var res CreateAppData
	err := cloudApi.DoGraphQLRequest(ctx, req, &res)
	if err != nil {
		return err
	}

	if len(res.Data.Errors) > 0 {
		return fmt.Errorf("error creating application: %v", res.Data.Errors[0])
	}

	return nil
}

// UpdateXp7App updates existing application on service that uses the uploaded jar
func UpdateXp7App(ctx context.Context, appID string, imageID string) error {
	req := cloudApi.NewGQLRequest(fmt.Sprintf(`
	mutation {
		updateXp7Application(params: {id: "%s", imageId: "%s"}) {
			application {
				id
			}
			errors
		}
	}
	`, appID, imageID))

	var res UpdateAppData
	err := cloudApi.DoGraphQLRequest(ctx, req, &res)
	if err != nil {
		return err
	}

	if len(res.Data.Errors) > 0 {
		return fmt.Errorf("error updating application: %v", res.Data.Errors[0])
	}

	return nil
}
