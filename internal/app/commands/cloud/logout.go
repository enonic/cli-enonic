package cloud

import (
	"fmt"
	"os"

	auth "github.com/enonic/cli-enonic/internal/app/commands/cloud/auth"
	"github.com/urfave/cli"
)

var Logout = cli.Command{
	Name:  "logout",
	Usage: "Logout of Enonic Cloud",
	Action: func(c *cli.Context) error {
		// Logout
		err := auth.Logout()
		if err != nil {
			return fmt.Errorf("Unable to logout: %s", err)
		}

		// Done
		fmt.Fprintf(os.Stdout, "You have been logged out!\n")
		return nil
	},
}
