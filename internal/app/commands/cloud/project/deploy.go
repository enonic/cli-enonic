package cloud

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/AlecAivazis/survey"
	"github.com/enonic/cli-enonic/internal/app/commands/cloud/auth"
	multipart "github.com/enonic/cli-enonic/internal/app/commands/cloud/client"
	mutations "github.com/enonic/cli-enonic/internal/app/commands/cloud/client/mutations"
	queries "github.com/enonic/cli-enonic/internal/app/commands/cloud/client/queries"
	util "github.com/enonic/cli-enonic/internal/app/commands/cloud/util"
	commonUtil "github.com/enonic/cli-enonic/internal/app/util"
	"github.com/urfave/cli"
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
	Name:    "deploy",
	Usage:   "Deploy project to Enonic Cloud",
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
		deployCtx, err := createDeployContext(target, jarFile)
		if err != nil {
			return err
		}

		// Confirm deploy context
		if !doDeploy {
			doDeploy = commonUtil.PromptBool(fmt.Sprintf("Deploy '%s' to '%s'. Is this correct?", deployCtx.jarFile, deployCtx.serviceName), false)
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
func createDeployContext(target string, deploymentJar string) (*deployContext, error) {
	var jar string
	// Find jar to deploy
	if deploymentJar != "" {
		// This is a user specified jar
		if err := util.FileExist(deploymentJar); err != nil {
			return nil, err
		} else {
			jar = deploymentJar
		}
	} else {
		// Find jar file in project dir
		deploymentJar, err := findProjectJar()
		if err != nil {
			return nil, fmt.Errorf("could not locate project jar: %v", err)
		}
		jar = deploymentJar
	}

	// Query api and create service map
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*15)
	defer cancel()

	res, err := queries.GetServices(ctx)
	if err != nil {
		return nil, fmt.Errorf("could not retrieve services: %v", err)
	}

	xp7Services := make(map[string]deployContext)
	for _, cloud := range res.Account.Clouds {
		for _, solution := range cloud.Solutions {
			for _, environment := range solution.Environments {
				for _, service := range environment.Services {
					if service.Kind == "xp7" {
						key := fmt.Sprintf("%s/%s/%s/%s", cloud.Name, solution.Name, environment.Name, service.Name)
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
		return nil, fmt.Errorf("No eligable service found")
	}

	var targetContext deployContext
	if len(xp7Services) == 1 {
		for k := range xp7Services {
			targetContext = xp7Services[k]
		}
	} else {
		key, err := promptForService(xp7Services)
		if err != nil {
			return nil, err
		}
		targetContext = xp7Services[key]
	}

	return &targetContext, nil
}

// func loadDeployContext(deployConfigFile string, deployContextName string, deploymentJar string, xp7Services map[string]string) (*deployContext, error) {
// 	var deployConfig DeployConfig

// 	// Check and create deploy config file if needed
// 	if err := util.FileExist(deployConfigFile); err == nil {
// 		err := util.ReadFile(deployConfigFile, func(r io.Reader) error {
// 			return json.NewDecoder(r).Decode(&deployConfig)
// 		})
// 		if err != nil {
// 			return nil, fmt.Errorf("could not read deploy config file '%s': %v", deployConfigFile, err)
// 		}
// 	}

// 	// Get app name
// 	appName, err := getAppName()
// 	if err != nil {
// 		return nil, err
// 	}

// 	// Create deploy context
// 	deployCtx := &deployContext{
// 		appName: appName,
// 	}

// 	// Find relevant context in deploy config file
// 	var c *Context
// 	for _, ct := range deployConfig.Contexts {
// 		if ct.Name == deployContextName {
// 			c = &ct
// 			break
// 		}
// 	}

// 	// If no context is found, create one
// 	if c == nil {
// 		newCtx, err := createDeployContext(deployContextName, appName, xp7Services)
// 		if err != nil {
// 			return nil, err
// 		}
// 		deployConfig.Contexts = append(deployConfig.Contexts, *newCtx)
// 		c = newCtx
// 	}

// 	// Load config file for context
// 	if err := util.FileExist(c.ConfigFile); err != nil {
// 		return nil, fmt.Errorf("could not find config file '%s'", c.ConfigFile)
// 	}
// 	deployCtx.configFile = c.ConfigFile

// 	// Find service id
// 	val, ok := xp7Services[c.Service]
// 	if !ok {
// 		return nil, fmt.Errorf("could not find service '%s'", c.Service)
// 	}
// 	deployCtx.serviceID = val
// 	deployCtx.serviceName = c.Service

// 	// Find jar to deploy
// 	if deploymentJar != "" {
// 		// This is a user specified jar
// 		if err := util.FileExist(deploymentJar); err != nil {
// 			return nil, err
// 		}
// 		deployCtx.jarFile = deploymentJar
// 	} else {
// 		// Find jar file in project dir
// 		deploymentJar, err = findProjectJar()
// 		if err != nil {
// 			return nil, fmt.Errorf("could not locate project jar: %v", err)
// 		}
// 		deployCtx.jarFile = deploymentJar
// 	}

// 	// Save configuration to disk and return
// 	return deployCtx, saveDeployConfigFile(deployConfigFile, &deployConfig)
// }

// func saveDeployConfigFile(deployConfigFile string, deployConfig *DeployConfig) error {
// 	return util.WriteFile(deployConfigFile, 0666, func(w io.Writer) error {
// 		b, err := json.MarshalIndent(deployConfig, "", "\t")
// 		if err != nil {
// 			return err
// 		}
// 		_, err = w.Write(b)
// 		return err
// 	})
// }

// func createDeployContext(name string, appName string, xp7Services map[string]string) (*Context, error) {
// 	// Get service
// 	service, err := promptForService(xp7Services)
// 	if err != nil {
// 		return nil, err
// 	}

// 	// Get config file
// 	configFile, err := promptForConfigFile(appName + ".cfg")
// 	if err != nil {
// 		return nil, err
// 	}

// 	// Create context

// 	return &Context{
// 		Name:       name,
// 		Service:    service,
// 		ConfigFile: configFile,
// 	}, nil
// }

// Functions for creating deployment context

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

// // Read gradle.properties to find app name
// func getAppName() (string, error) {
// 	var name string

// 	// Find the app name
// 	err := util.ReadFile("gradle.properties", func(r io.Reader) error {
// 		rd := bufio.NewReader(r)
// 		for {
// 			line, _, err := rd.ReadLine()
// 			if err != nil {
// 				break
// 			}

// 			var key, value string

// 			strippedLine := strings.Replace(string(line), " ", "", -1)
// 			strippedLine = strings.Replace(strippedLine, "=", " ", -1)
// 			_, err = fmt.Sscanf(strippedLine, "%s %s", &key, &value)

// 			if err != nil {
// 				continue
// 			}

// 			if key == "appName" {
// 				name = value
// 				return nil
// 			}
// 		}

// 		return fmt.Errorf("could not find appName")
// 	})

// 	return name, err
// }

// Ask what service the user wants to deploy to
func promptForService(xp7Services map[string]deployContext) (string, error) {
	var keys []string
	for k := range xp7Services {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	if len(keys) == 0 {
		return "", fmt.Errorf("You do not have any services to deploy to")
	}

	var key string
	prompt := &survey.Select{
		Message: "What service do you want do deploy to?",
		Options: keys,
	}
	err := survey.AskOne(prompt, &key, nil)
	if err != nil {
		return "", err
	}
	return key, nil
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
