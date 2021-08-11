package cloud

import (
	auth "cli-enonic/internal/app/commands/cloud/auth"
	util "cli-enonic/internal/app/commands/cloud/util"
	"fmt"
	qrcodeTerminal "github.com/Baozisoftware/qrcode-terminal-go"
	"github.com/pkg/browser"
	"github.com/urfave/cli"
	"io/ioutil"
	"os"
	"time"
)

var Login = cli.Command{
	Name:  "login",
	Usage: "Login to Enonic Cloud",
	Flags: []cli.Flag{
		cli.BoolFlag{
			Name:  "qr",
			Usage: "Print out QR code instead of url to log in with a mobile device",
		},
	},
	Action: func(c *cli.Context) error {
		// Check if logged in
		if _, err := auth.GetAccessToken(); err == nil {
			fmt.Fprintf(os.Stdout, "You are already logged in!\n")
			return nil
		}

		// Login
		if err := login(c.Bool("qr")); err != nil {
			return err
		}

		return nil
	},
}

func login(printQrCode bool) error {
	spin := util.CreateSpinner("Waiting for you to login")

	instructions := func(flow *auth.Flow) {
		fmt.Fprintf(os.Stdout, "Your login code is %s\n\n", flow.UserCode)
		if printQrCode {
			obj := qrcodeTerminal.New()
			obj.Get(flow.URI).Print()
			fmt.Fprintf(os.Stdout, "\n")
		} else {
			browser.Stdout = ioutil.Discard
			browser.Stderr = ioutil.Discard
			go func() {
				if err := browser.OpenURL(flow.URI); err != nil {
					fmt.Fprintf(os.Stdout, "Go to this url to login: %s\n\n", flow.URI)
				}
			}()
		}
		spin.Start()
	}

	afterTokenFetch := func(expiresAt int64) {
		spin.Stop()
		fmt.Fprintf(os.Stdout, "success!\n\n")
		fmt.Printf("Session expires at: %v\n", time.Unix(expiresAt, 0))
	}

	return auth.Login(instructions, afterTokenFetch)
}
