package commands

import (
	"bufio"
	"cli-enonic/internal/app/commands/common"
	"cli-enonic/internal/app/commands/remote"
	"cli-enonic/internal/app/util"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/pkg/browser"
	"github.com/urfave/cli"
)

var Login = cli.Command{
	Name:  "login",
	Usage: "Log in to the environment with device login in a browser (beta, XP 8.2 and later)",
	Description: "Where to log in is found through the management endpoint's OAuth discovery, unless " +
		"--auth-server gives the authorization server.\n\n" +
		"   Device login shows a code to confirm in a browser, which can be on another device. With --browser, " +
		"the browser is sent back to the CLI instead (authorization code with PKCE), so there is no code to " +
		"confirm. A browser on another device, or on a machine reached over SSH, cannot reach the CLI: paste the " +
		"address of the page it lands on into the CLI.",
	Flags: []cli.Flag{
		cli.BoolFlag{
			Name:  "browser",
			Usage: "Log in with the browser on this machine, which returns to the CLI, instead of device login",
		},
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

		var token *common.AccessToken
		if c.Bool("browser") {
			token, err = browserLogin(client, endpoints)
		} else {
			token, err = deviceLogin(client, endpoints)
		}
		if err != nil {
			return err
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

// deviceLogin shows a code to confirm in a browser, possibly on another device, and polls until it is
// approved (RFC 8628)
func deviceLogin(client *http.Client, endpoints *common.OAuthEndpoints) (*common.AccessToken, error) {
	if endpoints.DeviceAuthorization == "" {
		return nil, cli.NewExitError("The authorization server does not offer device login. Try 'enonic login --browser'.", 1)
	}
	code, err := common.StartDeviceLogin(client, endpoints)
	if err != nil {
		return nil, cli.NewExitError(fmt.Sprintf("Could not start login at %s: %v", endpoints.DeviceAuthorization, err), 1)
	}

	// verification_uri_complete is optional; without it the user types the code at verification_uri
	verificationUri := code.VerificationUriComplete
	if verificationUri == "" {
		verificationUri = code.VerificationUri
	}
	if verificationUri == "" {
		return nil, cli.NewExitError(fmt.Sprintf("No verification url in the response from %s", endpoints.DeviceAuthorization), 1)
	}
	fmt.Fprintf(os.Stderr, "To log in, open the URL below in a browser and confirm the code %s\n\n", util.FormatImportant(code.UserCode))
	fmt.Fprintf(os.Stderr, "    %s\n\n", verificationUri)
	if browserAvailable() {
		startBrowser(verificationUri)
	}

	common.StartSpinner("Waiting for approval")
	token, err := common.PollDeviceLogin(client, endpoints, code)
	common.StopSpinner()
	if err != nil {
		return nil, cli.NewExitError(fmt.Sprintf("\nLogin failed: %v", err), 1)
	}
	return token, nil
}

// browserLogin opens the authorization page in the browser on this machine, which is sent back to a
// one-time server on 127.0.0.1 with a code that is exchanged for tokens (RFC 6749, RFC 7636, RFC 8252)
func browserLogin(client *http.Client, endpoints *common.OAuthEndpoints) (*common.AccessToken, error) {
	if endpoints.Authorization == "" {
		return nil, cli.NewExitError("The authorization server does not offer browser login. Run 'enonic login' without --browser.", 1)
	}
	login, err := common.StartBrowserLogin(endpoints)
	if err != nil {
		return nil, cli.NewExitError(fmt.Sprintf("Could not start login: %v", err), 1)
	}

	opened := browserAvailable() && startBrowser(login.AuthorizationUrl)
	if opened {
		fmt.Fprintf(os.Stderr, "Log in in the browser that opened. If it did not, open this URL:\n\n")
	} else {
		fmt.Fprintf(os.Stderr, "There is no browser here. Open this URL in a browser on any device:\n\n")
	}
	fmt.Fprintf(os.Stderr, "    %s\n\n", login.AuthorizationUrl)
	fmt.Fprintf(os.Stderr, "After you approve, the browser comes back to this CLI. If it shows an error page instead, as it does\n"+
		"on another device, copy the address of that page and paste it here.\n\n")

	// the redirect and a pasted address race; whichever comes first is the answer
	go readPastes(pasteInput, login)

	token, err := login.Wait(client)
	if err != nil {
		hint := ""
		if strings.HasPrefix(err.Error(), "unauthorized_client") {
			hint = "\nThe ID provider does not let the Enonic CLI use browser login. Run 'enonic login' without --browser."
		}
		return nil, cli.NewExitError(fmt.Sprintf("Login failed: %v%s", err, hint), 1)
	}
	return token, nil
}

// pasteInput is where pasted addresses are read from; swapped in tests
var pasteInput io.Reader = os.Stdin

// readPastes hands each line to the login until one is taken, explaining a line that is not the right address
func readPastes(input io.Reader, login *common.BrowserLogin) {
	scanner := bufio.NewScanner(input)
	scanner.Buffer(make([]byte, 0, 4096), 64*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if err := login.Paste(line); err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			continue
		}
		return
	}
}

// browserAvailable tells whether a browser can be opened on this machine: not in an SSH session, and on
// Linux and the BSDs not without a display
var browserAvailable = detectBrowser

func detectBrowser() bool {
	for _, name := range []string{"SSH_CONNECTION", "SSH_CLIENT", "SSH_TTY"} {
		if os.Getenv(name) != "" {
			return false
		}
	}
	switch runtime.GOOS {
	case "windows", "darwin":
		return true
	default:
		return os.Getenv("DISPLAY") != "" || os.Getenv("WAYLAND_DISPLAY") != ""
	}
}

// startBrowser opens the URL in the default browser without waiting for it: the opener can run the browser in
// the foreground (xdg-open with BROWSER, for one), and the login must not wait for it to exit. It waits a moment
// for an opener that fails at once, e.g. none installed, and reports false then
func startBrowser(u string) bool {
	failed := make(chan error, 1)
	go func() { failed <- openBrowser(u) }()
	select {
	case err := <-failed:
		return err == nil
	case <-time.After(browserStartWait):
		return true
	}
}

// how long startBrowser waits for an opener to fail
var browserStartWait = time.Second

// openBrowser opens the URL in the default browser, quietly; swapped in tests
var openBrowser = openDefaultBrowser

func openDefaultBrowser(u string) error {
	browser.Stdout = io.Discard
	browser.Stderr = io.Discard
	return browser.OpenURL(u)
}
