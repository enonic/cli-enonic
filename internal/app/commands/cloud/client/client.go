package client

import (
	"context"
	"fmt"
	"net/http"

	auth "github.com/enonic/cli-enonic/internal/app/commands/cloud/auth"
	"github.com/machinebox/graphql"
)

func createGraphQLClient() *graphql.Client {
	client := graphql.NewClient(graphQLURL)
	// client.Log = func(s string) { fmt.Println(s) }
	return client
}

func doGraphQLRequest(ctx context.Context, req *graphql.Request, res interface{}) error {
	accessToken, err := auth.GetAccessToken()
	if err != nil {
		return fmt.Errorf("could not get access token: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	return createGraphQLClient().Run(ctx, req, res)
}

func doHttpRequest(ctx context.Context, req *http.Request) error {
	accessToken, err := auth.GetAccessToken()
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

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
