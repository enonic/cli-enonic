package sandbox

import (
	"cli-enonic/internal/app/commands/common"
	"fmt"
	"github.com/urfave/cli"
	"os"
)

var Upgrade = cli.Command{
	Name:      "upgrade",
	Aliases:   []string{"up"},
	Usage:     "Upgrades the distribution version.",
	ArgsUsage: "<name>",
	Flags: []cli.Flag{
		cli.StringFlag{
			Name:  "version, v",
			Usage: "Distro version to upgrade to.",
		},
		cli.BoolFlag{
			Name:  "all, a",
			Usage: "List all distro versions.",
		},
		cli.StringFlag{
			Name:  "image, i",
			Usage: "New Docker image to use (e.g. 'enonic/xp:7.13.4-sdk').",
		},
		common.FORCE_FLAG,
	},
	Action: func(c *cli.Context) error {

		var sandboxName string
		if c.NArg() > 0 {
			sandboxName = c.Args().First()
		} else if common.HasProjectData(".") {
			prjData := common.ReadProjectData(".")
			sandboxName = prjData.Sandbox
		}
		sandbox, _ := EnsureSandboxExists(c, EnsureSandboxOptions{
			Name:               sandboxName,
			SelectBoxMessage:   "Select sandbox to upgrade",
			ShowSuccessMessage: true,
		})
		if sandbox == nil {
			os.Exit(1)
		}

		if IsDockerDistro(sandbox.Distro) {
			imageStr := c.String("image")
			if imageStr == "" {
				if common.IsForceMode(c) {
					fmt.Fprintf(os.Stderr, "Docker-based sandbox '%s' requires --image flag to change the docker image.\n", sandbox.Name)
					os.Exit(1)
				}
				imageStr = promptDockerImage("", false)
			}
			EnsureDockerImageExists(imageStr)
			sandbox.Distro = FormatDockerDistro(imageStr)
			writeSandboxData(sandbox)
			fmt.Fprintf(os.Stdout, "Sandbox '%s' docker image changed to '%s'.\n", sandbox.Name, imageStr)
			return nil
		}

		minDistroVer := parseDistroVersion(sandbox.Distro, false)
		version, total := ensureVersionCorrect(c, c.String("version"), minDistroVer, false, c.Bool("all"), common.IsForceMode(c))
		if total == 0 {
			fmt.Fprintf(os.Stdout, "Sandbox '%s' is using the latest release of Enonic XP\n", sandbox.Name)
			os.Exit(0)
		}

		sandbox.Distro = formatDistroVersion(version)
		writeSandboxData(sandbox)
		fmt.Fprintf(os.Stdout, "Sandbox '%s' distro upgraded to '%s'.\n", sandbox.Name, sandbox.Distro)

		return nil
	},
}
