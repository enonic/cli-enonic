package client

import (
	"context"
	"fmt"
	"net/http"

	auth "github.com/enonic/cli-enonic/internal/app/commands/cloud/auth"
	"github.com/machinebox/graphql"
)

// DoGraphQLRequest execute a GraphQL request
func DoGraphQLRequest(ctx context.Context, req *graphql.Request, res interface{}) error {
	// Set access token in request
	accessToken, err := auth.GetAccessToken()
	if err != nil {
		return fmt.Errorf("could not get access token: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	// Do request
	return createGraphQLClient().Run(ctx, req, res)
}

func createGraphQLClient() *graphql.Client {
	client := graphql.NewClient(graphQLURL)
	// client.Log = func(s string) { fmt.Println(s) }
	return client
}

func doHTTPRequest(ctx context.Context, req *http.Request) error {
	// Set access token in request
	accessToken, err := auth.GetAccessToken()
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	// Do request
	client := &http.Client{}
	res, err := client.Do(req.WithContext(ctx))
	if err != nil {
		return err
	}

	// Check the response
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("bad response: %s", res.Status)
	}

	return nil
}
