package project

import (
	"github.com/urfave/cli"
	"github.com/enonic/xp-cli/internal/app/util"
	"github.com/enonic/xp-cli/internal/app/commands/common"
	"github.com/otiai10/copy"
	"strings"
	"os"
	"path/filepath"
	"bufio"
	"fmt"
	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing/transport/http"
	"gopkg.in/src-d/go-git.v4/plumbing"
	"regexp"
	"github.com/Masterminds/semver"
	"github.com/AlecAivazis/survey"
	"gopkg.in/src-d/go-git.v4/config"
)

var GITHUB_URL = "https://github.com/"
var ENONIC_REPOSITORY_PREFIX = "enonic/"
var GIT_REPOSITORY_SUFFIX = ".git"
var DEFAULT_NAME = "com.enonic.app.mytest"
var DEFAULT_VERSION = "1.0.0-SNAPSHOT"
var DEFAULT_STARTER = "starter-vanilla"
var STARTER_LIST = []string{
	"starter-vanilla", "starter-pwa", "starter-react", "starter-babel", "starter-bootstrap3",
	"starter-typescript", "starter-admin-tool", "starter-base", "starter-academy", "starter-gulp",
	"starter-intranet", "starter-training", "starter-headless", "starter-lib", "starter-angular2",
}

var Create = cli.Command{
	Name:  "create",
	Usage: "Create new project",
	Flags: []cli.Flag{
		cli.StringFlag{
			Name:  "auth, a",
			Usage: "Authentication token for basic authentication (user:password).",
		},
		cli.StringFlag{
			Name:  "branch, b",
			Usage: "Branch to checkout.",
			Value: "xp-7-0",
		},
		cli.StringFlag{
			Name:  "checkout, c",
			Usage: "Commit hash to checkout.",
		},
		cli.StringFlag{
			Name:  "destination, dest, d",
			Usage: "Destination path.",
		},
		cli.StringFlag{
			Name:  "repository, repo, r",
			Usage: "Repository path. Format: <enonic repo> or <organisation>/<repo> or <full repo url>",
		},
		cli.StringFlag{
			Name:  "version, ver, v",
			Usage: "Version number. Format: 1.0.0-SNAPSHOT",
		},
	},
	Action: func(c *cli.Context) error {
		fmt.Fprint(os.Stderr, "\n")

		gitUrl := ensureGitRepositoryUri(c)
		name := ensureNameArg(c)
		dest := ensureDestination(c, name)
		version := ensureVersion(c)
		hash := c.String("checkout")
		branch := c.String("branch")

		fmt.Fprint(os.Stderr, "\n")
		var user, pass string
		if authString := c.String("auth"); authString != "" {
			user, pass = common.EnsureAuth(authString)
		}

		cloneAndProcessRepo(gitUrl, dest, user, pass, branch, hash)

		propsFile := filepath.Join(dest, "gradle.properties")
		processGradleProperties(propsFile, name, version)

		fmt.Fprint(os.Stderr, "\n")

		ensureProjectDataExists(nil, dest, "A sandbox is required for your project, create one?")

		fmt.Fprintf(os.Stderr, "\nProject '%s' created.\n", name)

		return nil
	},
}

func ensureVersion(c *cli.Context) string {

	return util.PromptUntilTrue(c.String("version"), func(val *string, i byte) string {
		if *val == "" {
			if i == 0 {
				return fmt.Sprintf("\nApplication version (default: '%s'):", DEFAULT_VERSION)
			} else {
				*val = DEFAULT_VERSION
				fmt.Fprintln(os.Stderr, *val)
				return ""
			}
		} else if _, err := semver.NewVersion(*val); err != nil {
			return fmt.Sprintf("Version '%s' is not valid: ", *val)
		}
		return ""
	})
}

func ensureDestination(c *cli.Context, name string) string {
	dest := c.String("destination")
	destNotSet := dest == ""
	if destNotSet && name != "" {
		lastDot := strings.LastIndex(name, ".")
		dest = name[lastDot+1:]
	}

	return util.PromptUntilTrue(dest, func(val *string, i byte) string {
		if destNotSet && i == 0 {
			return fmt.Sprintf("\nDestination folder (default: '%s'):", dest)
		} else if i > 0 && *val == "" {
			*val = dest
			fmt.Fprintln(os.Stderr, *val)
		}
		if _, err := os.Stat(*val); os.IsNotExist(err) {
			return ""
		} else if os.IsExist(err) {
			return fmt.Sprintf("Destination folder '%s' already exists: ", *val)
		} else {
			return fmt.Sprintf("Folder '%s' could not be created: ", *val)
		}
	})
}

func ensureNameArg(c *cli.Context) string {
	var name string
	if c.NArg() > 0 {
		name = c.Args().First()
	}
	appNameRegex, _ := regexp.Compile("[a-z.]")

	return util.PromptUntilTrue(name, func(val *string, i byte) string {
		if *val == "" {
			if i == 0 {
				return fmt.Sprintf("\nProject name (default: '%s'):", DEFAULT_NAME)
			} else {
				*val = DEFAULT_NAME
				fmt.Fprintln(os.Stderr, *val)
				return ""
			}
		} else {
			if !appNameRegex.MatchString(*val) {
				return fmt.Sprintf("Name '%s' is not valid. Use lowercase letters and dot (.) symbol only: ", *val)
			}
			return ""
		}
	})
}

func processGradleProperties(propsFile, name, version string) {
	file, err := os.OpenFile(propsFile, os.O_RDWR|os.O_SYNC, 0755)
	if err == nil {
		defer file.Close()

		gp := NewGradleProcessor(name, version)
		scn := bufio.NewScanner(file)
		totalSize := 0
		prLines := make([]string, 0)

		for scn.Scan() {
			prLine := gp.processLine(scn.Text())
			prLines = append(prLines, prLine)
		}

		file.Seek(0, 0)
		wrt := bufio.NewWriter(file)
		for _, ln := range prLines {
			written, err := wrt.WriteString(ln + "\n")
			util.Fatal(err, "Error writing to gradle.properties file")
			totalSize += written
		}

		err3 := wrt.Flush()
		util.Fatal(err3, "Error writing to gradle.properties file")

		file.Truncate(int64(totalSize))
	}
}

func ensureGitRepositoryUri(c *cli.Context) string {
	var customRepoOption = "Custom repo"
	repo := c.String("repository")

	if repo == "" {
		err := survey.AskOne(&survey.Select{
			Message:  fmt.Sprintf("Starter (default: %s):", DEFAULT_STARTER),
			Options:  append([]string{customRepoOption}, STARTER_LIST...),
			Default:  DEFAULT_STARTER,
			PageSize: 10,
		}, &repo, nil)
		util.Fatal(err, "Distribution select error: ")
	}

	if repo == customRepoOption {
		repo = util.PromptUntilTrue("", func(val *string, i byte) string {
			if *val == "" {
				if i == 0 {
					return "Custom repo: "
				} else {
					return "Custom repo can not be empty: "
				}
			} else {
				return ""
			}
		})
	}

	if strings.Contains(repo, "://") {
		return repo
	} else if strings.Contains(repo, "/") {
		return GITHUB_URL + repo + GIT_REPOSITORY_SUFFIX
	} else {
		return GITHUB_URL + ENONIC_REPOSITORY_PREFIX + repo + GIT_REPOSITORY_SUFFIX
	}
}

func cloneAndProcessRepo(gitUrl, dest, user, pass, branch, hash string) {
	tempDest := filepath.Join(dest, ".InitAppTemporaryDirectory")
	gitClone(gitUrl, tempDest, user, pass, branch, hash)
	clearGitData(tempDest)
	copy.Copy(tempDest, dest)
	os.RemoveAll(tempDest)
}

func gitClone(url, dest, user, pass, branch, hash string) {
	var auth *http.BasicAuth
	if user != "" || pass != "" {
		auth = &http.BasicAuth{
			Username: user,
			Password: pass,
		}
	}

	repo, err := git.PlainClone(dest, false, &git.CloneOptions{
		Auth:     auth,
		URL:      url,
		Progress: os.Stderr,
	})
	util.Fatal(err, fmt.Sprintf("Could not connect to a remote repository '%s", url))

	if branch != "" || hash != "" {
		err2 := repo.Fetch(&git.FetchOptions{
			RemoteName: "origin",
			RefSpecs:   []config.RefSpec{"refs/*:refs/*", "HEAD:refs/heads/HEAD"},
		})
		util.Fatal(err2, "Could not fetch remote repo")

		tree, err := repo.Worktree()
		util.Fatal(err, "Could not get repo tree")

		opts := &git.CheckoutOptions{}
		if branch != "" {
			opts.Branch = plumbing.NewBranchReferenceName(branch)
		} else if hash != "" {
			opts.Hash = plumbing.NewHash(hash)
		}

		err3 := tree.Checkout(opts)
		util.Fatal(err3, fmt.Sprintf("Could not checkout hash '%s' or branch '%s'", hash, branch))
	}
}

func clearGitData(dest string) {
	os.RemoveAll(filepath.Join(dest, ".git"))
	os.Remove(filepath.Join(dest, "README.md"))
	os.Remove(filepath.Join(dest, ".gitignore"))
}
