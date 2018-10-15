package snapshot

import (
	"fmt"
	"github.com/urfave/cli"
	"net/http"
	"log"
	"strings"
	"errors"
	"io/ioutil"
	"encoding/json"
)

var List = cli.Command{
	Name:  "ls",
	Usage: "Returns a list of existing snapshots with name and status.",
	Flags: []cli.Flag{
		cli.StringFlag{
			Name:  "auth, a",
			Usage: "Authentication token for basic authentication (user:password)",
		},
		cli.StringFlag{
			Name:  "host, t",
			Value: "localhost",
			Usage: "Host name for server",
		},
		cli.StringFlag{
			Name:  "port, p",
			Value: "8080",
			Usage: "Port number for server",
		},
		cli.StringFlag{
			Name:  "scheme, s",
			Value: "http",
			Usage: "Scheme",
		},
	},
	Action: func(c *cli.Context) error {

		req, err := createRequest(c)
		if err != nil {
			log.Fatal("Params error: ", err)
		}

		client := &http.Client{}
		resp, err := client.Do(req)
		defer resp.Body.Close()

		if err != nil {
			log.Fatal("Request error: ", err)
		}

		if bodyBytes, err := ioutil.ReadAll(resp.Body); err == nil {
			if resp.StatusCode == http.StatusOK {
				fmt.Println(parseResponse(bodyBytes))
			} else {
				fmt.Printf("Response [%d]: %s", resp.StatusCode, resp.Status)
			}
		}

		return err
	},
}

func createRequest(c *cli.Context) (*http.Request, error) {
	auth := c.String("auth")
	host := c.String("host")
	port := c.String("port")
	scheme := c.String("scheme")

	req, err := http.NewRequest("GET", fmt.Sprintf("%s://%s:%s/api/repo/snapshot/list", scheme, host, port), nil)
	if err == nil && auth != "" {
		splitAuth := strings.Split(auth, ":")
		if len(splitAuth) != 2 {
			return nil, errors.New("parameter -a must have the following format `user:password`")
		} else {
			req.SetBasicAuth(splitAuth[0], splitAuth[1])
		}
	} else if auth == "" {
		err = errors.New("required parameter -a is missing")
	}
	return req, err
}

func parseResponse(bytes []byte) string {
	list := ListResponse{}
	if err := json.Unmarshal(bytes, &list); err != nil {
		log.Fatal("JSON Error: ", err)
	}
	return fmt.Sprintf("Snapshots:\n%s", strings.Join(list.Results, "\n"))
}

type ListResponse struct {
	Results []string `json:"results"`
}
