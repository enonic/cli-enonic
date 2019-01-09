package sandbox

import (
	"github.com/urfave/cli"
	"fmt"
	"os"
	"github.com/enonic/xp-cli/internal/app/util"
	"github.com/Masterminds/semver"
)

var Upgrade = cli.Command{
	Name:    "upgrade",
	Aliases: []string{"up"},
	Usage:   "Upgrades the distribution version.",
	Action: func(c *cli.Context) error {

		sandbox := EnsureSandboxNameExists(c, "Select sandbox:")
		if VERSION_LATEST == sandbox.Distro {
			fmt.Fprintf(os.Stderr, "Sandbox '%s' already has the latest distro version.\n", sandbox.Name)
			os.Exit(0)
		}
		version := ensureVersionArg(c)
		preventVersionDowngrade(sandbox, version)

		_, distroVer := ensureDistroPresent(version)
		writeSandboxData(sandbox.Name, SandboxData{distroVer})
		fmt.Fprintf(os.Stderr, "Sandbox '%s' distro set to: %s", sandbox.Name, distroVer)

		return nil
	},
}

func preventVersionDowngrade(sandbox Sandbox, newString string) {
	if VERSION_LATEST != newString {
		oldVer, err := semver.NewVersion(sandbox.Distro)
		util.Fatal(err, fmt.Sprintf("Could not parse sandbox distro version from: %s", sandbox.Distro))
		newVer, err2 := semver.NewVersion(newString)
		util.Fatal(err2, fmt.Sprintf("Could not parse new distro version from: %s", newString))

		if oldVer.Compare(newVer) > 0 {
			fmt.Fprintf(os.Stderr, "Sandbox '%s' already has newer distro version: %s\n", sandbox.Name, sandbox.Distro)
			os.Exit(0)
		}
	}
}

func ensureVersionArg(c *cli.Context) string {
	var version string
	if c.NArg() > 1 {
		version = c.Args().Get(1) // Get the second, as the first is the name
	}
	return ensureVersionCorrect(version)
}
