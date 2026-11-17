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
	"sort"
	"strings"
	"text/tabwriter"
	"time"
)

const LIST_EVENT = "list"

var List = cli.Command{
	Name:    "list",
	Aliases: []string{"ls"},
	Usage:   "List installed applications",
	Flags: append([]cli.Flag{
		cli.BoolFlag{
			Name:  "json",
			Usage: "Print the full application details as JSON",
		},
		cli.BoolFlag{
			Name:  "all",
			Usage: "Include XP's system applications (XP 8.2 and later; older XP lists them anyway)",
		},
		common.FORCE_FLAG,
	}, common.AUTH_AND_TLS_FLAGS...),
	Action: func(c *cli.Context) error {

		apps := listApps(c)
		// only the management APIs of XP 8.2 and later tell system applications apart
		knowsSystem := common.ServerApis(c)
		all := c.Bool("all")
		if knowsSystem && !all {
			apps.Applications = withoutSystem(apps.Applications)
		}

		sort.Slice(apps.Applications, func(i, j int) bool {
			return apps.Applications[i].Key < apps.Applications[j].Key
		})

		if c.Bool("json") {
			fmt.Fprintln(os.Stdout, util.PrettyPrintJSON(apps))
		} else {
			printApplications(os.Stdout, apps.Applications, knowsSystem, all)
		}

		return nil
	},
}

func withoutSystem(apps []Application) []Application {
	kept := make([]Application, 0, len(apps))
	for _, app := range apps {
		if !app.System {
			kept = append(kept, app)
		}
	}
	return kept
}

// printApplications lists the applications; knowsSystem tells whether system applications can be told apart,
// all whether they are included and marked
func printApplications(out io.Writer, apps []Application, knowsSystem, all bool) {
	showSystem := knowsSystem && all
	if len(apps) == 0 {
		if knowsSystem && !all {
			fmt.Fprintln(os.Stderr, "No applications installed. Use --all to include system applications")
		} else {
			fmt.Fprintln(os.Stderr, "No applications installed")
		}
		return
	}

	columns := []string{"KEY", "NAME", "VERSION", "STATE", "LOCAL"}
	if showSystem {
		columns = append(columns, "SYSTEM")
	}
	writer := tabwriter.NewWriter(out, 0, 0, 3, ' ', 0)
	fmt.Fprintln(writer, strings.Join(columns, "\t"))
	for _, app := range apps {
		row := []string{app.Key, app.DisplayName, app.Version, app.State, yes(app.Local)}
		if showSystem {
			row = append(row, yes(app.System))
		}
		fmt.Fprintln(writer, strings.Join(row, "\t"))
	}
	writer.Flush()
}

func yes(value bool) string {
	if value {
		return "yes"
	}
	return ""
}

func listApps(c *cli.Context) *ApplicationsResult {
	if common.ServerApis(c) {
		res := common.SendRequest(c, common.CreateRequest(c, "GET", common.ServerApiPath("app"), nil), "Loading applications")
		var infos ApplicationInfos
		common.ParseResponse(res, &infos)
		fmt.Fprintln(os.Stderr, "Done")
		return infos.result()
	}

	// the legacy REST has no endpoint that returns the applications, they only arrive as the first event on this stream
	req := common.CreateRequest(c, "GET", "app/events", nil)
	req.Header.Set("Accept", common.SSE_CONTENT_TYPE)

	res := common.SendRequest(c, req, "Loading applications")
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		var body interface{}
		common.ParseResponse(res, &body)
		os.Exit(1) // ParseResponse already exits on a failure response, a 2xx without a stream is unusable too
	}

	result, err := readApplicationList(res.Body)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error reading applications: ", err)
		os.Exit(1)
	}

	fmt.Fprintln(os.Stderr, "Done")

	return result
}

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
	System           bool      `json:"system"`
	ModifiedTime     time.Time `json:"modifiedTime"`
	MinSystemVersion string    `json:"minSystemVersion,omitempty"`
	MaxSystemVersion string    `json:"maxSystemVersion,omitempty"`
	Url              string    `json:"url,omitempty"`
	VendorName       string    `json:"vendorName,omitempty"`
	VendorUrl        string    `json:"vendorUrl,omitempty"`
}

// ApplicationInfos is how the management API lists applications: an application's name is its title
type ApplicationInfos struct {
	Applications []ApplicationInfo `json:"applications"`
}

type ApplicationInfo struct {
	Application
	Title string `json:"title,omitempty"`
}

func (info ApplicationInfo) application() Application {
	app := info.Application
	if app.DisplayName == "" {
		app.DisplayName = info.Title
	}
	return app
}

func (infos ApplicationInfos) result() *ApplicationsResult {
	apps := make([]Application, len(infos.Applications))
	for i, info := range infos.Applications {
		apps[i] = info.application()
	}
	return &ApplicationsResult{Applications: apps}
}
