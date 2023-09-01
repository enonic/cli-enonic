package cloud

import (
	"cli-enonic/internal/app/commands/cloud/auth"
	multipart "cli-enonic/internal/app/commands/cloud/client"
	mutations "cli-enonic/internal/app/commands/cloud/client/mutations"
	queries "cli-enonic/internal/app/commands/cloud/client/queries"
	"cli-enonic/internal/app/commands/common"
	commonUtil "cli-enonic/internal/app/util"
	"context"
	"errors"
	"fmt"
	"github.com/urfave/cli"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	defaultDeployConfigFile = ".enonic-cloud"
	defaultDeployContext    = "default"
)

// DeployConfig is the schema for the deploy configuration file
type DeployConfig struct {
	Contexts []Context `json:"contexts"`
}

// Context is the deploy configuration context
type Context struct {
	Name       string `json:"name"`
	Service    string `json:"service"`
	ConfigFile string `json:"configFile"`
}

// Internal struct passed around when actually deploying a jar
type deployContext struct {
	solutionID  string
	jarID       string
	jarFile     string
	serviceID   string
	serviceName string
}

// Cli command

var ProjectDeploy = cli.Command{
	Name:    "install",
	Usage:   "Install project jar to Enonic Cloud",
	Aliases: []string{},
	Flags: []cli.Flag{
		cli.StringFlag{
			Name:  "j",
			Usage: "Jar to deploy (default: \"./build/libs/*.jar\")",
		},
		cli.IntFlag{
			Name:  "t",
			Usage: "Upload timeout in seconds",
			Value: 300,
		},
		cli.BoolFlag{
			Name:  "y",
			Usage: "Skip confirmation prompt",
		},
	},
	Action: func(c *cli.Context) error {
		// Check if logged in
		if !auth.IsLoggedIn() {
			return errors.New("Login with 'enonic cloud login'")
		}

		// Load parameters
		var target string
		if len(c.Args()) == 1 {
			target = c.Args().Get(0)
		}
		jarFile := c.String("j")
		operationTimeout := time.Second * time.Duration(c.Int("t"))
		doDeploy := c.Bool("y")

		// Create deploy context
		deployCtx, err := createDeployContext(target, jarFile, common.IsForceMode(c))
		if err != nil {
			return err
		}

		// Confirm deploy context
		if !doDeploy {
			doDeploy = commonUtil.PromptBool(fmt.Sprintf("Deploy '%s' to '%s'. Is this correct", deployCtx.jarFile, deployCtx.serviceName), true)
		}
		if !doDeploy {
			return fmt.Errorf("deployment not confirmed by user")
		}

		// Do the actual deployment
		ctx, cancel := context.WithTimeout(context.Background(), operationTimeout)
		defer cancel()

		// Upload the app
		err = uploadApp(ctx, deployCtx)
		if err != nil {
			return err
		}

		fmt.Fprintf(os.Stdout, "Deploying jar to service '%s'\n", deployCtx.serviceName)
		err = mutations.CreateXp7AppFromUpload(ctx, deployCtx.serviceID, deployCtx.jarID)
		if err != nil {
			return err
		}

		fmt.Fprintf(os.Stdout, "Success!\n")
		return nil
	},
}

// Functions to setup deployment context

// Create deployment context
func createDeployContext(target string, deploymentJar string, force bool) (*deployContext, error) {
	var jar string
	// Find jar to deploy
	jar = commonUtil.PromptProjectJar(deploymentJar, force)

	// Query api and create service map
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*15)
	defer cancel()

	res, err := queries.GetServices(ctx)
	if err != nil {
		return nil, fmt.Errorf("could not retrieve services: %v", err)
	}

	xp7Services := make(map[string]deployContext)
	accounts := res.Search.Accounts

	sort.Slice(accounts[:], func(i, j int) bool {
		return accounts[i].Name == "Hobby"
	})

	for _, account := range accounts {
		for _, solution := range account.Solutions {
			for _, environment := range solution.Environments {
				for _, service := range environment.Services {
					if service.Kind == "xp7" {
						key := fmt.Sprintf("%s/%s/%s/%s", account.Name, solution.Name, environment.Name, service.Name)
						if strings.HasPrefix(key, target) {
							xp7Services[key] = deployContext{
								serviceName: key,
								serviceID:   service.ID,
								solutionID:  solution.ID,
								jarFile:     jar,
							}
						}
					}
				}
			}
		}
	}

	if len(xp7Services) == 0 {
		return nil, fmt.Errorf("No eligible service found")
	}

	var targetContext deployContext
	if len(xp7Services) == 1 {
		for k := range xp7Services {
			targetContext = xp7Services[k]
		}
	} else {
		key, _, err := promptForService(xp7Services)
		if err != nil {
			return nil, err
		}
		targetContext = xp7Services[key]
	}

	return &targetContext, nil
}

// Try to find the project jar
func findProjectJar() (string, error) {
	// Look under ./build/libs
	root := filepath.Join("build", "libs")
	var match string

	// Return the first match we can find
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		matched, err := filepath.Match(filepath.Join(root, "*.jar"), path)
		if err != nil {
			return err
		} else if matched {
			match = path
			return nil
		}
		return nil
	})

	if match == "" {
		return "", fmt.Errorf("could not find jar")
	}

	return match, err
}

// Ask what service the user wants to deploy to
func promptForService(xp7Services map[string]deployContext) (string, int, error) {
	var keys []string
	for k := range xp7Services {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	if len(keys) == 0 {
		return "", -1, fmt.Errorf("You do not have any services to deploy to")
	}

	return commonUtil.PromptSelect(&commonUtil.SelectOptions{
		Message: "Select service you want to deploy to",
		Options: keys,
	})
}

// Upload app given a valid deploy context
func uploadApp(ctx context.Context, deployCtx *deployContext) error {
	// Upload app
	jarID, err := multipart.UploadApp(ctx, deployCtx.solutionID, deployCtx.jarFile, "Uploading jar ")
	if err != nil {
		return fmt.Errorf("unable to upload app: %v", err)
	}

	deployCtx.jarID = jarID

	return nil
}
