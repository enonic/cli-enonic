package snapshot

import (
	"github.com/urfave/cli"
	"net/http"
	"fmt"
	"bytes"
	"encoding/json"
)

var New = cli.Command{
	Name:  "new",
	Usage: "Stores a snapshot of the current state of the repository.",
	Flags: append([]cli.Flag{
		cli.StringFlag{
			Name:  "repo, r",
			Usage: "The name of the repository to snapshot",
		},
	}, SNAPSHOT_FLAGS...),
	Action: func(c *cli.Context) error {

		req := createNewRequest(c)

		resp := sendRequest(req)

		fmt.Println(parseResponse(resp))

		return nil
	},
}

func createNewRequest(c *cli.Context) *http.Request {
	body := new(bytes.Buffer)
	params := map[string]interface{}{}
	if repo := c.String("repo"); repo != "" {
		params["repositoryId"] = repo
	}
	json.NewEncoder(body).Encode(params)

	req := createRequest(c, "POST", "api/repo/snapshot", body)
	req.Header.Set("Content-Type", "application/json")

	return req
}
