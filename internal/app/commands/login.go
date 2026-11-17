package commands

import (
	"bufio"
	"cli-enonic/internal/app/commands/common"
	"cli-enonic/internal/app/commands/env"
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
	"golang.org/x/term"
)

var Login = cli.Command{
	Name:      "login",
	Usage:     "Log in to the environment with device login in a browser (beta, XP 8.2 and later)",
	ArgsUsage: "[<environment name> | <management url>]",
	Description: "With an environment name or a management url, that environment becomes the active one, and one " +
		"that does not exist yet is added: a name as an Enonic Cloud environment, like 'enonic env add " +
		"<account>/<solution>/<environment>', a url of an Enonic Cloud management host as " +
		"<account>/<solution>/<environment>, any other url under the first label of its host without -mgmt, " +
		"e.g. 'acme' for https://acme-mgmt.example.com. Without " +
		"one, it logs in to the active environment.\n\n" +
		"   Where to log in is found through the management endpoint's OAuth discovery, unless " +
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
		target, err := loginTarget(c)
		if err != nil {
			return err
		}
		mgmtUrl := target.Url.URL
		client := common.OAuthClient(c)

		var endpoints *common.OAuthEndpoints
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
		fmt.Fprintf(os.Stderr, "Logged in to %s\n", common.TargetLabel(target))
		return nil
	},
}

// loginTarget is the environment named by the argument, made the active one, or else the active environment
func loginTarget(c *cli.Context) (*remote.RemoteData, error) {
	switch {
	case c.NArg() > 1:
		cli.ShowCommandHelp(c, c.Command.Name)
		return nil, cli.NewExitError("", 1)
	case c.NArg() == 0:
		return remote.GetActiveRemote(), nil
	case c.IsSet("env"):
		return nil, cli.NewExitError("Give the environment either as an argument or with --env, not both.", 1)
	default:
		return env.Activate(c.Args().First())
	}
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
	if canPrompt() {
		fmt.Fprintf(os.Stderr, "One-time code %s\n", util.FormatImportant(code.UserCode))
		fmt.Fprintf(os.Stderr, "Press Enter to open %s in your browser...\n\n", verificationUri)
		start := browserStarter()
		go readLines(loginInput, func(string) bool {
			start(verificationUri)
			return true
		})
	} else {
		fmt.Fprintf(os.Stderr, "To log in, open the URL below in a browser and confirm the code %s\n\n", util.FormatImportant(code.UserCode))
		fmt.Fprintf(os.Stderr, "    %s\n\n", verificationUri)
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

	prompt := canPrompt()
	if prompt {
		fmt.Fprintf(os.Stderr, "To log in, open this URL:\n\n")
	} else {
		fmt.Fprintf(os.Stderr, "Open this URL in a browser on any device:\n\n")
	}
	fmt.Fprintf(os.Stderr, "    %s\n\n", login.AuthorizationUrl)
	fmt.Fprintf(os.Stderr, "After you approve, the browser comes back to this CLI. If it shows an error page instead, as it does\n"+
		"on another device, copy the address of that page and paste it here.\n\n")
	if prompt {
		fmt.Fprintf(os.Stderr, "Press Enter to open it in your browser...\n\n")
	}

	// the redirect and a pasted address race; whichever comes first is the answer. An empty line opens the
	// browser, once
	start := browserStarter()
	opened := false
	go readLines(loginInput, func(line string) bool {
		if line == "" {
			if prompt && !opened {
				opened = true
				start(login.AuthorizationUrl)
			}
			return false
		}
		if err := login.Paste(line); err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			return false
		}
		return true
	})

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

// loginInput is where Enter and pasted addresses are read from; swapped in tests
var loginInput io.Reader = os.Stdin

// readLines hands each trimmed line to handle until it reports that it is done
func readLines(input io.Reader, handle func(line string) bool) {
	scanner := bufio.NewScanner(input)
	scanner.Buffer(make([]byte, 0, 4096), 64*1024)
	for scanner.Scan() {
		if handle(strings.TrimSpace(scanner.Text())) {
			return
		}
	}
}

// canPrompt tells whether to ask for Enter before opening the browser: a browser can be opened here and
// someone types at the terminal
func canPrompt() bool {
	return browserAvailable() && interactive()
}

// interactive tells whether standard input is a terminal; swapped in tests
var interactive = isTerminal

func isTerminal() bool {
	return term.IsTerminal(int(os.Stdin.Fd()))
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
	return browserStarter()(u)
}

// browserStarter is startBrowser with the opener and the wait taken now, for a caller that opens the browser
// later from another goroutine
func browserStarter() func(u string) bool {
	opener, wait := openBrowser, browserStartWait
	return func(u string) bool {
		failed := make(chan error, 1)
		go func() { failed <- opener(u) }()
		select {
		case err := <-failed:
			return err == nil
		case <-time.After(wait):
			return true
		}
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
