package sandbox

import (
	"github.com/urfave/cli"
	"fmt"
	"github.com/enonic/xp-cli/internal/app/util"
	"os"
)

var Delete = cli.Command{
	Name:    "delete",
	Usage:   "Delete a sandbox",
	Aliases: []string{"del"},
	Action: func(c *cli.Context) error {

		name := ensureSandboxNameArg(c, "Select sandbox to delete:")

		if boxesData := readSandboxesData(); boxesData.Running == name {
			fmt.Fprintf(os.Stderr, "Sandbox '%s' is currently running, stop it first!", name)
			os.Exit(1)
		}

		data := readSandboxData(name)
		boxes := getSandboxesUsingDistro(data.Distro)
		if len(boxes) == 1 && boxes[0] == name && acceptToDeleteDistro(data.Distro) {
			deleteDistro(data.Distro)
		}

		deleteSandbox(name)
		fmt.Fprintf(os.Stderr, "Sandbox '%s' deleted", name)

		return nil
	},
}

func acceptToDeleteDistro(distro string) bool {
	answer := util.PromptUntilTrue("", func(val string, ind byte) string {
		if ind == 0 {
			return fmt.Sprintf("Distro '%s' is not used any more, would you like to delete it ? [Y/n] ", distro)
		} else {
			switch val {
			case "Y", "n":
				return ""
			default:
				return "Please type 'Y' for yes, or 'n' for no: "
			}
		}
	})
	return answer == "Y"
}
