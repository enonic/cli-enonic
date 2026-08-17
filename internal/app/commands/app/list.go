package app

import (
	"cli-enonic/internal/app/commands/common"
	"cli-enonic/internal/app/util"
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"github.com/urfave/cli"
	"io"
	"net/http"
	"os"
	"time"
)

// XP publishes the installed applications as the first event on the app events stream
const LIST_EVENT = "list"

var List = cli.Command{
	Name:    "list",
	Aliases: []string{"ls"},
	Usage:   "List installed applications",
	Flags:   append([]cli.Flag{common.FORCE_FLAG}, common.AUTH_AND_TLS_FLAGS...),
	Action: func(c *cli.Context) error {

		apps := listApps(c)

		fmt.Fprintln(os.Stdout, util.PrettyPrintJSON(apps))

		return nil
	},
}

func listApps(c *cli.Context) *ApplicationsResult {
	req := common.CreateRequest(c, "GET", "app/events", nil)
	req.Header.Set("Accept", common.SSE_CONTENT_TYPE)

	res := common.SendRequest(c, req, "Loading applications")
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		// not an event stream, so report it the way the other commands do.
		// ParseResponse exits on a failure response, anything else is unusable here
		var body interface{}
		common.ParseResponse(res, &body)
		os.Exit(1)
	}

	result, err := readApplicationList(res.Body)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error reading applications: ", err)
		os.Exit(1)
	}

	fmt.Fprintln(os.Stderr, "Done")

	return result
}

// readApplicationList returns the applications carried by the first "list" event
// on an app events stream, ignoring any other event that may arrive before it.
func readApplicationList(stream io.Reader) (*ApplicationsResult, error) {
	reader := common.NewSseReader(stream)

	for {
		event, err := reader.Next()
		if err != nil {
			if err == io.EOF {
				return nil, errors.Errorf("stream ended without a '%s' event", LIST_EVENT)
			}
			return nil, err
		}

		if event.Event != LIST_EVENT {
			continue
		}

		var result ApplicationsResult
		if err = json.Unmarshal([]byte(event.Data), &result); err != nil {
			return nil, err
		}
		return &result, nil
	}
}

type ApplicationsResult struct {
	Applications []Application `json:"applications"`
}

type Application struct {
	Key              string    `json:"key"`
	DisplayName      string    `json:"displayName,omitempty"`
	Version          string    `json:"version"`
	State            string    `json:"state"`
	Local            bool      `json:"local"`
	ModifiedTime     time.Time `json:"modifiedTime"`
	MinSystemVersion string    `json:"minSystemVersion,omitempty"`
	MaxSystemVersion string    `json:"maxSystemVersion,omitempty"`
	Url              string    `json:"url,omitempty"`
	VendorName       string    `json:"vendorName,omitempty"`
	VendorUrl        string    `json:"vendorUrl,omitempty"`
}
