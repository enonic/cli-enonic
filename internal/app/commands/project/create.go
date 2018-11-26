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
)

var GITHUB_URL = "https://github.com/"
var ENONIC_REPOSITORY_PREFIX = "enonic/"
var GIT_REPOSITORY_SUFFIX = ".git"

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
		},
		cli.StringFlag{
			Name:  "hash",
			Usage: "Commit hash to checkout.",
		},
		cli.StringFlag{
			Name:  "destination, dest, d",
			Usage: "Destination path.",
		},
		cli.StringFlag{
			Name:  "repository, repo, r",
			Usage: "Repository path. Format: <enonic repo> or <organisation>/<repo> or <full repo url>",
			Value: "starter-vanilla",
		},
		cli.StringFlag{
			Name:  "version, ver, v",
			Usage: "Version number.",
			Value: "1.0.0-SNAPSHOT",
		},
	},
	Action: func(c *cli.Context) error {

		name := ensureNameArg(c)
		gitUrl := ensureGitRepositoryUri(c)
		dest := verifyDestinationNotExists(c, name)
		hash := c.String("hash")
		branch := c.String("branch")

		var user, pass string
		if c.String("auth") != "" {
			user, pass = common.EnsureAuth(c)
		}

		cloneAndProcessRepo(gitUrl, dest, user, pass, branch, hash)

		propsFile := filepath.Join(dest, "gradle.properties")
		processGradleProperties(propsFile, name, c.String("version"))

		return nil
	},
}

func verifyDestinationNotExists(c *cli.Context, name string) string {
	if dest := c.String("destination"); dest != "" {
		return util.PromptUntilTrue(dest, func(val string, i byte) string {
			if _, err := os.Stat(val); !os.IsNotExist(err) {
				return fmt.Sprintf("Destination folder '%s' already exists: ", val)
			}
			return ""
		})
	}
	return name
}

func ensureNameArg(c *cli.Context) string {
	var name string
	if c.NArg() > 0 {
		name = c.Args().First()
	}

	return util.PromptUntilTrue(name, func(val string, i byte) string {
		if len(strings.TrimSpace(val)) == 0 {
			if i == 0 {
				return "Enter the name of the project: "
			} else {
				return "Project name can not be empty: "
			}
		} else {
			if _, err := os.Stat(val); !os.IsNotExist(err) {
				return fmt.Sprintf("Folder named '%s' already exists: ", val)
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
	repo := util.PromptUntilTrue(c.String("repository"), func(val string, ind byte) string {
		if len(strings.TrimSpace(val)) == 0 {
			if ind == 0 {
				return "Enter the repository to clone: "
			} else {
				return "Repository url can not be empty"
			}
		} else {
			return ""
		}
	})

	if strings.Contains(repo, ":/") {
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
		tree, err := repo.Worktree()
		util.Fatal(err, "Could not get repo tree")

		opts := &git.CheckoutOptions{}
		if hash != "" {
			opts.Branch = plumbing.NewBranchReferenceName(branch)
		} else if branch != "" {
			opts.Hash = plumbing.NewHash(hash)
		}

		err2 := tree.Checkout(opts)
		util.Fatal(err2, fmt.Sprintf("Could not checkout hash:'%s', branch:'%s'", hash, branch))
	}
}

func clearGitData(dest string) {
	os.RemoveAll(filepath.Join(dest, ".git"))
	os.Remove(filepath.Join(dest, "README.md"))
	os.Remove(filepath.Join(dest, ".gitignore"))
}
