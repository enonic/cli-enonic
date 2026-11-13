package commands

import (
	"cli-enonic/internal/app/commands/common"
	"cli-enonic/internal/app/commands/remote"
	"cli-enonic/internal/app/util"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"

	"github.com/pkg/browser"
	"github.com/urfave/cli"
)

var Login = cli.Command{
	Name:  "login",
	Usage: "Log in to the management endpoint with device login in a browser (XP 8.2 and later)",
	Description: "Where to log in is found through the management endpoint's OAuth discovery. Without it, " +
		"device login runs at the ID provider given by --id-provider on the web endpoint given by --url.",
	Flags: []cli.Flag{
		cli.StringFlag{
			Name:  "url, u",
			Usage: "Web endpoint url where device login is served, instead of the one OAuth discovery finds. Defaults to the management endpoint host on port 8080",
		},
		cli.StringFlag{
			Name:  "id-provider, i",
			Value: "system",
			Usage: "ID provider to log in with, instead of the one OAuth discovery finds",
		},
		common.CLIENT_KEY_FLAG,
		common.CLIENT_CERT_FLAG,
	},
	Action: func(c *cli.Context) error {
		mgmtUrl := remote.GetActiveRemote().Url.URL
		client := common.OAuthClient(c)

		endpoints, err := loginEndpoints(c, client, &mgmtUrl)
		if err != nil {
			return cli.NewExitError(fmt.Sprintf("Could not find where to log in to %s: %v", mgmtUrl.String(), err), 1)
		}

		code, err := common.StartDeviceLogin(client, endpoints)
		if err != nil {
			return cli.NewExitError(fmt.Sprintf("Could not start login at %s: %v", endpoints.DeviceAuthorization, err), 1)
		}

		fmt.Fprintf(os.Stderr, "To log in, open the URL below in a browser and confirm the code %s\n\n", util.FormatImportant(code.UserCode))
		fmt.Fprintf(os.Stderr, "    %s\n\n", code.VerificationUriComplete)
		browser.Stdout = io.Discard
		browser.Stderr = io.Discard
		browser.OpenURL(code.VerificationUriComplete)

		common.StartSpinner("Waiting for approval")
		token, err := common.PollDeviceLogin(client, endpoints, code)
		common.StopSpinner()
		if err != nil {
			return cli.NewExitError(fmt.Sprintf("\nLogin failed: %v", err), 1)
		}

		if err := common.SaveAccessToken(&mgmtUrl, token); err != nil {
			return cli.NewExitError(fmt.Sprintf("\nCould not save access token: %v", err), 1)
		}
		fmt.Fprintln(os.Stderr, "Done")
		if user := token.UserLogin(); user != "" {
			fmt.Fprintf(os.Stderr, "Logged in to %s as %s\n", mgmtUrl.String(), user)
		} else {
			fmt.Fprintf(os.Stderr, "Logged in to %s\n", mgmtUrl.String())
		}
		return nil
	},
}

// loginEndpoints are the ones OAuth discovery finds, unless --url or --id-provider says where to log in
// or the management endpoint serves no discovery
func loginEndpoints(c *cli.Context, client *http.Client, mgmtUrl *url.URL) (*common.OAuthEndpoints, error) {
	if !c.IsSet("url") && !c.IsSet("id-provider") {
		endpoints, err := common.DiscoverEndpoints(client, mgmtUrl)
		if !errors.Is(err, common.ErrNoDiscovery) {
			return endpoints, err
		}
	}
	webUrl := c.String("url")
	if webUrl == "" {
		webUrl = common.DefaultWebUrl(mgmtUrl)
	}
	return common.IdProviderEndpoints(webUrl, c.String("id-provider")), nil
}

var Logout = cli.Command{
	Name:  "logout",
	Usage: "Forget the access token obtained with 'enonic login'",
	Action: func(c *cli.Context) error {
		mgmtUrl := remote.GetActiveRemote().Url.URL
		removed, err := common.RemoveAccessToken(&mgmtUrl)
		if err != nil {
			return cli.NewExitError(fmt.Sprintf("Could not remove access token: %v", err), 1)
		}
		if removed {
			fmt.Fprintf(os.Stderr, "Logged out from %s\n", mgmtUrl.String())
		} else {
			fmt.Fprintf(os.Stderr, "Not logged in to %s\n", mgmtUrl.String())
		}
		return nil
	},
}
