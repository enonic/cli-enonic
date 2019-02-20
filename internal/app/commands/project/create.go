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
	"encoding/json"
	"bytes"
)

var GITHUB_URL = "https://github.com/"
var ENONIC_REPOSITORY_PREFIX = "enonic/"
var GIT_REPOSITORY_SUFFIX = ".git"
var DEFAULT_NAME = "com.enonic.app.mytest"
var DEFAULT_VERSION = "1.0.0-SNAPSHOT"
var MARKET_STARTERS_REQUEST = `{
  market {
    query(
      query: "type='com.enonic.app.market:starter' AND ngram('data.version.supportedVersions', '7.0')"
    ) {
      displayName
      ... on com_enonic_app_market_Starter {
        data {
          ... on com_enonic_app_market_Starter_Data {
            gitUrl
            version {
              supportedVersions
              gitTag
            }
          }
        }
      }
    }
  }
}`

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

		branch := c.String("branch")
		hash := c.String("checkout")
		gitUrl := ensureGitRepositoryUri(c, &hash)
		name := ensureNameArg(c)
		dest := ensureDestination(c, name)
		version := ensureVersion(c)

		var user, pass string
		if authString := c.String("auth"); authString != "" {
			user, pass = common.EnsureAuth(authString)
		}

		fmt.Fprintln(os.Stderr, "\nInitializing project...")
		cloneAndProcessRepo(gitUrl, dest, user, pass, branch, hash)

		propsFile := filepath.Join(dest, "gradle.properties")
		processGradleProperties(propsFile, name, version)

		absDest, err := filepath.Abs(dest)
		util.Fatal(err, "Error creating project")
		fmt.Fprintf(os.Stderr, "Project created in '%s'\n\n", absDest)

		ensureProjectDataExists(nil, dest, "A sandbox is required for your project, create one?")

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

	return util.PromptString("Application version", c.String("version"), DEFAULT_VERSION, versionValidator)
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

	return util.PromptString("Destination folder", dest, defaultDest, destValidator)
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

	return util.PromptString("Project name", name, DEFAULT_NAME, nameValidator)
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

func ensureGitRepositoryUri(c *cli.Context, hash *string) string {
	var (
		customRepoOption        = "Custom repo"
		starterList             []string
		starter, defaultStarter string
	)
	repo := c.String("repository")

	if repo == "" {
		starters := fetchStarters(c)
		for i, st := range starters {
			if i == 0 {
				defaultStarter = st.DisplayName
			}
			starterList = append(starterList, st.DisplayName)
		}

		err := survey.AskOne(&survey.Select{
			Message:  "Starter",
			Options:  append([]string{customRepoOption}, starterList...),
			Default:  defaultStarter,
			PageSize: 10,
		}, &starter, nil)
		util.Fatal(err, "Starter select error: ")

		if starter != customRepoOption {
			for _, st := range starters {
				if st.DisplayName == starter {
					repo = st.Data.GitUrl
					if *hash == "" {
						*hash = findLatestHash(&st.Data.Version)
					}
					break
				}
			}
		} else {
			repo = customRepoOption
		}
	}

	if repo == customRepoOption {
		var repoValidator = func(val interface{}) error {
			str := val.(string)
			if str == "" || len(strings.TrimSpace(str)) < 3 {
				return errors.Errorf("Repository name can not be empty", val)
			}
			return nil
		}
		repo = util.PromptString("Custom repository", "", "", repoValidator)
	}

	if strings.Contains(repo, "://") {
		return repo
	} else if strings.Contains(repo, "/") {
		return GITHUB_URL + repo + GIT_REPOSITORY_SUFFIX
	} else {
		return GITHUB_URL + ENONIC_REPOSITORY_PREFIX + repo + GIT_REPOSITORY_SUFFIX
	}
}

func findLatestHash(versions *[]StarterVersion) string {
	var (
		latestSemver, maxSemver *semver.Version
		maxHash                 string
	)
	for _, v := range *versions {
		latestSemver = findLatestVersion(v.SupportedVersions)

		if maxSemver == nil || latestSemver.GreaterThan(maxSemver) {
			maxSemver = latestSemver
			maxHash = v.GitTag
		}
	}
	return maxHash
}
func findLatestVersion(versions []string) *semver.Version {
	var (
		latestSemver, maxSemver *semver.Version
		err                     error
	)
	for _, ver := range versions {
		latestSemver, err = semver.NewVersion(ver)
		util.Warn(err, fmt.Sprintf("Could not parse version '%s'", ver))

		if maxSemver == nil || latestSemver.GreaterThan(maxSemver) {
			maxSemver = latestSemver
		}
	}
	return maxSemver
}

func fetchStarters(c *cli.Context) []Starter {
	body := new(bytes.Buffer)
	params := map[string]string{
		"query": MARKET_STARTERS_REQUEST,
	}
	json.NewEncoder(body).Encode(params)

	req := common.CreateRequest(c, "POST", common.MARKET_URL, body)
	res, err := common.SendRequestCustom(req, "Loading starters from enonic market", 1)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error, check your internet connection.")
		return []Starter{}
	}

	var result StartersResponse
	common.ParseResponse(res, &result)

	fmt.Fprintln(os.Stderr, "Done.")
	return result.Data.Market.Query
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
		if hash != "" {
			opts.Hash = plumbing.NewHash(hash)
		} else if branch != "" {
			opts.Branch = plumbing.NewBranchReferenceName(branch)
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

type StarterVersion struct {
	SupportedVersions []string
	GitTag            string
}

type Starter struct {
	DisplayName string
	Data struct {
		GitUrl  string
		Version []StarterVersion
	}
}

type StartersResponse struct {
	Data struct {
		Market struct {
			Query []Starter
		}
	}
}
