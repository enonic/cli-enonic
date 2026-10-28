package commands

import (
	"cli-enonic/internal/app/commands/common"
	"cli-enonic/internal/app/commands/remote"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/pkg/browser"
	"github.com/urfave/cli"
)

var Login = cli.Command{
	Name:  "login",
	Usage: "Log in to the management endpoint with device login in a browser (XP 8.2 and later)",
	Flags: []cli.Flag{
		cli.StringFlag{
			Name:  "url, u",
			Usage: "Web endpoint url where device login is served. Defaults to the management endpoint host on port 8080",
		},
		cli.StringFlag{
			Name:  "id-provider, i",
			Value: "system",
			Usage: "ID provider to log in with",
		},
	},
	Action: func(c *cli.Context) error {
		mgmtUrl := remote.GetActiveRemote().Url.URL
		webUrl := c.String("url")
		if webUrl == "" {
			webUrl = common.DefaultWebUrl(&mgmtUrl)
		}
		idProvider := c.String("id-provider")

		code, err := common.StartDeviceLogin(webUrl, idProvider)
		if err != nil {
			return cli.NewExitError(fmt.Sprintf("Could not start login at %s: %v", webUrl, err), 1)
		}

		fmt.Fprintf(os.Stderr, "Your login code is %s\n", code.UserCode)
		fmt.Fprintf(os.Stderr, "Go to %s to approve\n\n", code.VerificationUriComplete)
		browser.Stdout = io.Discard
		browser.Stderr = io.Discard
		browser.OpenURL(code.VerificationUriComplete)

		common.StartSpinner("Waiting for approval")
		token, err := common.PollDeviceLogin(webUrl, idProvider, code)
		common.StopSpinner()
		if err != nil {
			return cli.NewExitError(fmt.Sprintf("\nLogin failed: %v", err), 1)
		}

		if err := common.SaveAccessToken(&mgmtUrl, token); err != nil {
			return cli.NewExitError(fmt.Sprintf("\nCould not save access token: %v", err), 1)
		}
		fmt.Fprintf(os.Stderr, "\nLogged in to %s. Access token expires at %s\n", mgmtUrl.String(), time.Unix(token.ExpiresAt, 0).Format(time.RFC1123))
		return nil
	},
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
