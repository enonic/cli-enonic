package commands

import (
	"cli-enonic/internal/app/commands/common"
	"cli-enonic/internal/app/commands/remote"
	"cli-enonic/internal/app/util"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/pkg/browser"
	"github.com/urfave/cli"
)

var Login = cli.Command{
	Name:  "login",
	Usage: "Log in to the environment with device login in a browser (beta, XP 8.2 and later)",
	Description: "Where to log in is found through the management endpoint's OAuth discovery, unless " +
		"--auth-server gives the authorization server.",
	Flags: []cli.Flag{
		cli.StringFlag{
			Name:  "auth-server",
			Usage: "Issuer url of the authorization server to log in at, instead of the one OAuth discovery finds",
		},
		common.CLIENT_KEY_FLAG,
		common.CLIENT_CERT_FLAG,
		common.ENV_FLAG,
	},
	Action: func(c *cli.Context) error {
		target := remote.GetActiveRemote()
		mgmtUrl := target.Url.URL
		client := common.OAuthClient(c)

		var endpoints *common.OAuthEndpoints
		var err error
		if issuer := c.String("auth-server"); issuer != "" {
			endpoints, err = common.DiscoverIssuerEndpoints(client, issuer, &mgmtUrl)
		} else {
			endpoints, err = common.DiscoverEndpoints(client, &mgmtUrl)
		}
		if errors.Is(err, common.ErrNoDiscovery) {
			return cli.NewExitError(fmt.Sprintf("%s does not advertise a login server. Device login needs XP 8.2 or later.", mgmtUrl.String()), 1)
		} else if err != nil {
			return cli.NewExitError(fmt.Sprintf("Could not find where to log in to %s: %v", common.TargetLabel(target), err), 1)
		}

		code, err := common.StartDeviceLogin(client, endpoints)
		if err != nil {
			return cli.NewExitError(fmt.Sprintf("Could not start login at %s: %v", endpoints.DeviceAuthorization, err), 1)
		}

		// verification_uri_complete is optional; without it the user types the code at verification_uri
		verificationUri := code.VerificationUriComplete
		if verificationUri == "" {
			verificationUri = code.VerificationUri
		}
		if verificationUri == "" {
			return cli.NewExitError(fmt.Sprintf("No verification url in the response from %s", endpoints.DeviceAuthorization), 1)
		}
		fmt.Fprintf(os.Stderr, "To log in, open the URL below in a browser and confirm the code %s\n\n", util.FormatImportant(code.UserCode))
		fmt.Fprintf(os.Stderr, "    %s\n\n", verificationUri)
		browser.Stdout = io.Discard
		browser.Stderr = io.Discard
		browser.OpenURL(verificationUri)

		common.StartSpinner("Waiting for approval")
		token, err := common.PollDeviceLogin(client, endpoints, code)
		common.StopSpinner()
		if err != nil {
			return cli.NewExitError(fmt.Sprintf("\nLogin failed: %v", err), 1)
		}

		if err := common.SaveAccessToken(target, token); err != nil {
			return cli.NewExitError(fmt.Sprintf("\nCould not save access token: %v", err), 1)
		}
		fmt.Fprintln(os.Stderr, "Done")
		if user := token.UserLogin(); user != "" {
			fmt.Fprintf(os.Stderr, "Logged in to %s as %s\n", common.TargetLabel(target), user)
		} else {
			fmt.Fprintf(os.Stderr, "Logged in to %s\n", common.TargetLabel(target))
		}
		return nil
	},
}

var Logout = cli.Command{
	Name:  "logout",
	Usage: "Forget the login to the environment obtained with 'enonic login' (beta)",
	Flags: []cli.Flag{
		common.ENV_FLAG,
	},
	Action: func(c *cli.Context) error {
		target := remote.GetActiveRemote()
		removed, err := common.RemoveAccessToken(target)
		if err != nil {
			return cli.NewExitError(fmt.Sprintf("Could not remove access token: %v", err), 1)
		}
		if removed {
			fmt.Fprintf(os.Stderr, "Logged out from %s\n", common.TargetLabel(target))
		} else {
			fmt.Fprintf(os.Stderr, "Not logged in to %s\n", common.TargetLabel(target))
		}
		return nil
	},
}
