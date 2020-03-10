package cloud

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	auth "github.com/enonic/cli-enonic/internal/app/commands/cloud/auth"
	cloudApi "github.com/enonic/cli-enonic/internal/app/commands/cloud/client"
	util "github.com/enonic/cli-enonic/internal/app/commands/cloud/util"
	"github.com/urfave/cli"
)

var App = cli.Command{
	Name:        "app",
	Usage:       "Manage apps in Enonic Cloud",
	Aliases:     []string{},
	Subcommands: []cli.Command{AppDeploy},
}

var AppDeploy = cli.Command{
	Name:    "deploy",
	Usage:   "Deploy app to Enonic Cloud",
	Aliases: []string{},
	Flags: []cli.Flag{
		cli.StringFlag{
			Name:  "c",
			Usage: "Configuration for app",
		},
		cli.StringFlag{
			Name:  "n",
			Usage: "App name",
		},
		cli.StringFlag{
			Name:  "s",
			Usage: "Solution id",
		},
	},
	Action: func(c *cli.Context) error {
		// Check if logged in
		if !auth.IsLoggedIn() {
			return errors.New("Login with 'enonic cloud login'")
		}

		// Check solution param
		solution := c.String("s")
		if solution == "" {
			return errors.New("Provide a solution id with flag '-s'")
		}

		// Check app name param
		appName := c.String("n")
		if appName == "" {
			return errors.New("Provide app name with flag '-n'")
		}

		// Check config param
		configFile := c.String("c")
		if configFile == "" {
			return errors.New("Provide an app config file with flag '-c'")
		}
		if err := util.FileExist(configFile); err != nil {
			return err
		}
		config, err := util.ReadFileToString(configFile)
		if err != nil {
			return err
		}

		// Check jar
		if len(c.Args()) != 1 {
			return errors.New("Provide app jar as an argument")
		}
		jar := c.Args()[0]
		if err := util.FileExist(jar); err != nil {
			return err
		}

		// Get deploy key
		fmt.Fprintf(os.Stdout, "Uploading config ... ")
		ctx, cancel := context.WithTimeout(context.Background(), time.Minute*15)
		defer cancel()
		createXp7ConfigRes, err := cloudApi.CreateXp7Config(ctx, solution, appName, config)
		if err != nil {
			fmt.Fprintf(os.Stdout, "Failed!\n")
			return fmt.Errorf("unable to upload config: %v", err)
		}
		fmt.Fprintf(os.Stdout, "Done!\n")

		// Upload app
		fmt.Fprintf(os.Stdout, "Uploading jar ... ")
		ctx, cancel = context.WithTimeout(context.Background(), time.Minute*15)
		defer cancel()

		if err := cloudApi.UploadApp(ctx, createXp7ConfigRes.CreateXp7Config, jar); err != nil {
			fmt.Fprintf(os.Stdout, "Failed!\n")
			return fmt.Errorf("unable to upload app: %v", err)
		}
		fmt.Fprintf(os.Stdout, "Done!\n")

		return nil
	},
}
