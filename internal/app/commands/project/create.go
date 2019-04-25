package project

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/AlecAivazis/survey"
	"github.com/Masterminds/semver"
	"github.com/enonic/enonic-cli/internal/app/commands/common"
	"github.com/enonic/enonic-cli/internal/app/util"
	"github.com/otiai10/copy"
	"github.com/pkg/browser"
	"github.com/pkg/errors"
	"github.com/urfave/cli"
	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/config"
	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4/plumbing/transport/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

var GITHUB_REPO_TPL = "https://github.com/%s/%s.git"
var STARTER_LIST_TPL = "%s (%s)"
var DEFAULT_NAME = "com.example.myproject"
var DEFAULT_VERSION = "1.0.0-SNAPSHOT"
var MARKET_STARTERS_REQUEST = `{
  market {
    query(
      query: "type='com.enonic.app.market:starter' AND data.version.supportedVersions LIKE '7.*'"
    ) {
      displayName
      ... on com_enonic_app_market_Starter {
        data {
          ... on com_enonic_app_market_Starter_Data {
			shortDescription
			documentationUrl
            gitUrl
            version {
              supportedVersions
			  versionNumber
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
		gitUrl, starter := ensureGitRepositoryUri(c, &hash)
		name := ensureNameArg(c)
		dest := ensureDestination(c, name)
		version := ensureVersion(c)

		var user, pass string
		if authString := c.String("auth"); authString != "" {
			user, pass = common.EnsureAuth(authString)
		}

		fmt.Fprintln(os.Stderr, "\nInitializing project...")
		cloneAndProcessRepo(gitUrl, dest, user, pass, branch, hash)
		fmt.Fprint(os.Stderr, "\n")

		propsFile := filepath.Join(dest, "gradle.properties")
		processGradleProperties(propsFile, name, version)

		absDest, err := filepath.Abs(dest)
		util.Fatal(err, "Error creating project")

		pData := ensureProjectDataExists(nil, dest, "A sandbox is required for your project, create one?")

		if pData == nil {
			fmt.Fprintf(os.Stderr, "\nProject created in '%s'\n", absDest)
		} else {
			fmt.Fprintf(os.Stderr, "Project created in '%s' and linked to '%s'\n", absDest, pData.Sandbox)
		}
		fmt.Fprint(os.Stderr, "\n")

		if starter != nil && util.PromptBool(fmt.Sprintf("Open %s docs in the browser ?", starter.DisplayName), true) {
			err := browser.OpenURL(starter.Data.DocumentationUrl)
			util.Warn(err, "Could not open documentation at: "+starter.Data.DocumentationUrl)
		}

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

	return util.PromptString("Project version", c.String("version"), DEFAULT_VERSION, versionValidator)
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

func ensureGitRepositoryUri(c *cli.Context, hash *string) (string, *Starter) {
	var (
		customRepoOption  = "Custom repo"
		starterList      []string
		selectedOption   string
		starter          *Starter
	)
	repo := c.String("repository")

	if repo == "" {
		starters := fetchStarters(c)
		sort.SliceStable(starters, func(i, j int) bool {
			return starters[i].DisplayName < starters[j].DisplayName
		})
		for _, st := range starters {
			starterList = append(starterList, fmt.Sprintf(STARTER_LIST_TPL, st.DisplayName, st.Data.ShortDescription))
		}

		err := survey.AskOne(&survey.Select{
			Message:  "Starter",
			Options:  append(starterList, customRepoOption),
			PageSize: 10,
		}, &selectedOption, nil)
		util.Fatal(err, "Starter select error: ")

		if selectedOption != customRepoOption {
			for _, st := range starters {
				if fmt.Sprintf(STARTER_LIST_TPL, st.DisplayName, st.Data.ShortDescription) == selectedOption {
					repo = st.Data.GitUrl
					starter = &st
					if *hash == "" {
						*hash = findLatestHash(&st.Data.Version)
					}
					break
				}
			}
		} else {
			var repoValidator = func(val interface{}) error {
				str := val.(string)
				if str == "" || len(strings.TrimSpace(str)) < 3 {
					return errors.Errorf("Repository name can not be empty", val)
				}
				return nil
			}
			repo = util.PromptString("Custom repository", "", "", repoValidator)
		}
	}

	if strings.Contains(repo, "://") {
		return repo, starter
	} else if splitRepo := strings.Split(repo, "/"); len(splitRepo) == 2 {
		return fmt.Sprintf(GITHUB_REPO_TPL, splitRepo[0], splitRepo[1]), starter
	} else {
		return fmt.Sprintf(GITHUB_REPO_TPL, "enonic", repo), starter
	}
}

func findLatestHash(versions *[]StarterVersion) string {
	var (
		latestSemver, maxSemver *semver.Version
		maxHash                 string
		err                     error
	)
	for _, v := range *versions {
		latestSemver, err = semver.NewVersion(v.VersionNumber)
		util.Warn(err, fmt.Sprintf("Could not parse version number: %s", v.VersionNumber))

		if isSupports7(v.SupportedVersions) && (maxSemver == nil || latestSemver.GreaterThan(maxSemver)) {
			maxSemver = latestSemver
			maxHash = v.GitTag
		}
	}
	return maxHash
}
func isSupports7(versions []string) bool {
	var (
		latestSemver, semver7 *semver.Version
		err                   error
	)
	semver7, _ = semver.NewVersion("7.0.0")
	for _, ver := range versions {
		latestSemver, err = semver.NewVersion(ver)
		util.Warn(err, fmt.Sprintf("Could not parse version '%s'", ver))

		if !latestSemver.LessThan(semver7) {
			return true
		}
	}
	return false
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
			// verify hash exists
			if _, err := repo.CommitObject(plumbing.NewHash(hash)); err == nil {
				opts.Hash = plumbing.NewHash(hash)
			}
		}
		// use branch if hash is not set only
		if opts.Hash.IsZero() && branch != "" {
			// verify branch exists
			if _, err := repo.Branch(hash); err == nil {
				opts.Branch = plumbing.NewBranchReferenceName(branch)
			}
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
	VersionNumber     string
}

type Starter struct {
	DisplayName string
	Data        struct {
		GitUrl           string
		ShortDescription string
		DocumentationUrl string
		Version          []StarterVersion
	}
}

type StartersResponse struct {
	Data struct {
		Market struct {
			Query []Starter
		}
	}
}
