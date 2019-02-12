package project

import (
	"github.com/urfave/cli"
	"github.com/enonic/enonic-cli/internal/app/util"
	"github.com/enonic/enonic-cli/internal/app/commands/common"
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
	"github.com/pkg/errors"
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

	var versionValidator = func(val interface{}) error {
		str := val.(string)
		if _, err := semver.NewVersion(str); err != nil {
			return errors.Errorf("Version '%s' is not valid: ", str)
		}
		return nil
	}

	return util.PromptOnce("Application version", c.String("version"), DEFAULT_VERSION, versionValidator)
}

func ensureDestination(c *cli.Context, name string) string {
	dest := c.String("destination")
	var defaultDest string
	if dest == "" && name != "" {
		lastDot := strings.LastIndex(name, ".")
		defaultDest = name[lastDot+1:]
	}

	var destValidator = func(val interface{}) error {
		str := val.(string)
		if val == "" || len(strings.TrimSpace(str)) < 2 {
			return errors.New("Destination folder must be at least 2 characters long: ")
		} else if stat, err := os.Stat(str); stat != nil {
			return errors.Errorf("Destination folder '%s' already exists: ", str)
		} else if os.IsNotExist(err) {
			return nil
		} else if err != nil {
			return errors.Errorf("Folder '%s' could not be created: ", str)
		}
		return nil
	}

	return util.PromptOnce("Destination folder", dest, defaultDest, destValidator)
}

func ensureNameArg(c *cli.Context) string {
	var name string
	if c.NArg() > 0 {
		name = c.Args().First()
	}
	appNameRegex, _ := regexp.Compile("^[a-z0-9.]{3,}$")

	var nameValidator = func(val interface{}) error {
		str := val.(string)
		if !appNameRegex.MatchString(str) {
			return errors.Errorf("Name '%s' is not valid. Use at least 3 lowercase letters, digits or dot (.) symbols: ", val)
		}
		return nil
	}

	return util.PromptOnce("Project name", name, DEFAULT_NAME, nameValidator)
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
			Message:  "Starter",
			Options:  append([]string{customRepoOption}, STARTER_LIST...),
			Default:  DEFAULT_STARTER,
			PageSize: 10,
		}, &repo, nil)
		util.Fatal(err, "Distribution select error: ")
	}

	if repo == customRepoOption {

		var repoValidator = func(val interface{}) error {
			str := val.(string)
			if str == "" || len(strings.TrimSpace(str)) < 3 {
				return errors.Errorf("Repository name can not be empty", val)
			}
			return nil
		}

		repo = util.PromptOnce("Custom repository", "", "", repoValidator)
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
