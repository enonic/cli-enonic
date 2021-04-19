package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	auth "cli-enonic/internal/app/commands/cloud/auth"
)

// DoGraphQLRequest execute a GraphQL request
func DoGraphQLRequest(ctx context.Context, req *GQLRequest, res interface{}) error {
	// Set access token in request
	accessToken, err := auth.GetAccessToken()
	if err != nil {
		return fmt.Errorf("could not get access token: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	// Do request
	return createGraphQLClient().Run(ctx, req, res)
}

func createGraphQLClient() *GQLClient {
	client := NewGQLClient(graphQLURL)
	// client.Log = func(s string) { fmt.Println(s) }
	return client
}

func doHTTPRequest(ctx context.Context, req *http.Request, res interface{}) error {
	// Set access token in request
	accessToken, err := auth.GetAccessToken()
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	// Do request
	client := &http.Client{}
	actualRes, err := client.Do(req.WithContext(ctx))
	if err != nil {
		return err
	}

	// Check the response
	if actualRes.StatusCode != http.StatusOK {
		return fmt.Errorf("bad response: %s", actualRes.Status)
	}

	defer actualRes.Body.Close()
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, actualRes.Body); err != nil {
		return err
	}

	if err := json.NewDecoder(&buf).Decode(&res); err != nil {
		return err
	}

	return nil
}
