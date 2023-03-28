package project

import (
	"bufio"
	"bytes"
	"cli-enonic/internal/app/commands/common"
	"cli-enonic/internal/app/util"
	"encoding/json"
	"fmt"
	"github.com/Masterminds/semver"
	"github.com/fatih/color"
	"github.com/otiai10/copy"
	"github.com/pkg/browser"
	"github.com/pkg/errors"
	"github.com/urfave/cli"
	"gopkg.in/AlecAivazis/survey.v1"
	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/config"
	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4/plumbing/transport/http"
	"net/url"
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
var UPSTREAM_NAME = "origin"
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
var MARKET_DOCS_QUERY_TPL = "data.gitUrl='%s' AND data.version.supportedVersions LIKE '7.*'"
var MARKET_DOCS_REQUEST = `query($query: String!){
  market {
    query(
      query: $query
    ) {
      displayName
      ... on com_enonic_app_market_Starter {
        data {
          ... on com_enonic_app_market_Starter_Data {
            documentationUrl
            gitUrl
            version {
              supportedVersions
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
		cli.StringFlag{
			Name:  "name, n",
			Usage: "Project name.",
		},
		common.AUTH_FLAG,
		common.FORCE_FLAG,
	},
	Action: func(c *cli.Context) error {
		fmt.Fprint(os.Stderr, "\n")

		branch := c.String("branch")
		hash := c.String("checkout")
		gitUrl, starter := ensureGitRepositoryUri(c, &hash, &branch)
		name := ensureNameArg(c)
		dest := ensureDestination(c, name)
		version := ensureVersion(c)

		var user, pass string
		if authString := c.String("auth"); authString != "" {
			user, pass = common.EnsureAuth(authString, common.IsForceMode(c))
		}

		fmt.Fprintln(os.Stderr, "\nInitializing project...")
		cloneAndProcessRepo(gitUrl, dest, user, pass, branch, hash)
		fmt.Fprint(os.Stderr, "\n")

		propsFile := filepath.Join(dest, "gradle.properties")
		processGradleProperties(propsFile, name, version)

		absDest, err := filepath.Abs(dest)
		util.Fatal(err, "Error creating project")

		pData := ensureProjectDataExists(c, dest, "A sandbox is required for your project, create one?", false)

		if pData == nil || pData.Sandbox == "" {
			fmt.Fprintf(os.Stdout, "\nProject created in '%s'\n", absDest)
		} else {
			fmt.Fprintf(os.Stdout, "Project created in '%s' and linked to '%s'\n", absDest, pData.Sandbox)
		}
		fmt.Fprint(os.Stderr, "\n")

		if starter == nil {
			// see if we have docs for that url
			starter = lookupStarterDocs(c, gitUrl)
		}

		if starter != nil {
			if !common.IsForceMode(c) && util.PromptBool(fmt.Sprintf("Open %s docs in the browser ?", starter.DisplayName), false) {
				err := browser.OpenURL(starter.Data.DocumentationUrl)
				util.Warn(err, "Could not open documentation at: "+starter.Data.DocumentationUrl)
			} else {
				fmt.Fprintf(os.Stderr, "%s docs: %s\n", starter.DisplayName, starter.Data.DocumentationUrl)
			}
		}

		fmt.Print("\nYour new Enonic application has been successfully bootstrapped. Deploy it by running:\n\n")

		boldCyan := color.New(color.FgCyan, color.Bold)
		boldCyan.Printf("cd %s\nenonic project deploy\n\n", dest)

		return nil
	},
}

func ensureVersion(c *cli.Context) string {
	force := common.IsForceMode(c)
	var versionValidator = func(val interface{}) error {
		str := val.(string)
		if _, err := semver.NewVersion(str); err != nil {
			if force {
				// assume DEFAULT_VERSION in non-interactive mode instead of error
				if str == "" {
					fmt.Fprintf(os.Stderr, "Version was not supplied. Using default: %s\n", DEFAULT_VERSION)
				} else {
					fmt.Fprintf(os.Stderr, "Version '%s' is not valid. Using default: %s\n", str, DEFAULT_VERSION)
				}
				return nil
			}
			return errors.Errorf("Version '%s' is not valid: ", str)
		}
		return nil
	}

	version := util.PromptString("Project version", c.String("version"), DEFAULT_VERSION, versionValidator)
	if !force || version != "" {
		return version
	} else {
		return DEFAULT_VERSION
	}
}

func ensureDestination(c *cli.Context, name string) string {
	dest := c.String("destination")
	force := common.IsForceMode(c)
	var defaultDest string
	if dest == "" && name != "" {
		lastDot := strings.LastIndex(name, ".")
		defaultDest = name[lastDot+1:]
	}

	var destValidator func(val interface{}) error
	destValidator = func(val interface{}) error {
		str := val.(string)
		if val == "" || len(strings.TrimSpace(str)) < 2 {
			if force {
				// Assume defaultDest in non-interactive mode
				if val == "" {
					fmt.Fprintf(os.Stderr, "Destination folder was not supplied. Using default: %s\n", defaultDest)
				} else {
					fmt.Fprintf(os.Stderr, "Destination folder '%s' must be at least 2 characters long. Using default: %s\n", str, defaultDest)
				}
				// validate defaultDest as well in case it already exists
				return destValidator(defaultDest)
			}
			return errors.New("Destination folder must be at least 2 characters long: ")
		} else if stat, err := os.Stat(str); stat != nil {
			if force {
				fmt.Fprintf(os.Stderr, "Destination folder '%s' already exists.\n", str)
				os.Exit(1)
			}
			return errors.Errorf("Destination folder '%s' already exists: ", str)
		} else if os.IsNotExist(err) {
			return nil
		} else if err != nil {
			if force {
				fmt.Fprintf(os.Stderr, "Folder '%s' could not be created: %s\n", str, err.Error())
				os.Exit(1)
			}
			return errors.Errorf("Folder '%s' could not be created: ", str)
		}
		return nil
	}

	userDest := util.PromptString("Destination folder", dest, defaultDest, destValidator)
	if !force || userDest != "" {
		return userDest
	} else {
		return defaultDest
	}
}

func ensureNameArg(c *cli.Context) string {
	name := c.String("name")
	force := common.IsForceMode(c)
	if name == "" && c.NArg() > 0 {
		name = c.Args().First()
	}
	appNameRegex, _ := regexp.Compile("^[a-z0-9.]{3,}$")

	var nameValidator = func(val interface{}) error {
		str := val.(string)
		if !appNameRegex.MatchString(str) {
			if force {
				if str == "" {
					fmt.Fprintf(os.Stderr, "Name was not supplied. Using default: %s\n", DEFAULT_NAME)
				} else {
					fmt.Fprintf(os.Stderr, "Name '%s' is not valid. Using default: %s\n", val, DEFAULT_NAME)
				}
				return nil
			}
			return errors.Errorf("Name '%s' is not valid. It must be min 3 characters long and only contain lowercase letters, digits, and periods [a-z0-9.]", val)
		}
		return nil
	}

	projectName := util.PromptString("Project name", name, DEFAULT_NAME, nameValidator)
	if !force || projectName != "" {
		return projectName
	} else {
		return DEFAULT_NAME
	}
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

func ensureGitRepositoryUri(c *cli.Context, hash *string, branch *string) (string, *Starter) {
	var (
		customRepoOption = "Custom repo"
		starterList      []string
		selectedOption   string
		starter          *Starter
	)
	repo := c.String("repository")
	if repo != "" {
		if parsedRepo, err := expandToAbsoluteURl(repo, true); err != nil {
			repo = ""
		} else {
			repo = parsedRepo
		}
	}

	if repo == "" {
		if common.IsForceMode(c) {
			fmt.Fprintln(os.Stderr, "Repository flag can not be empty in non-interactive mode.")
			os.Exit(1)
		}

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
		util.Fatal(err, "Exiting: ")

		if selectedOption != customRepoOption {
			for _, st := range starters {
				if fmt.Sprintf(STARTER_LIST_TPL, st.DisplayName, st.Data.ShortDescription) == selectedOption {
					repo = st.Data.GitUrl
					starter = &st
					if *hash == "" && *branch == "" {
						*hash = findLatestHash(&st.Data.Version)
					}
					break
				}
			}
		} else {
			var repoValidator = func(val interface{}) error {
				str := val.(string)
				if str == "" {
					return errors.New("Repository name can not be empty")
				} else if _, err := expandToAbsoluteURl(str, true); err != nil {
					return err
				}
				return nil
			}
			repo = util.PromptString("Custom Git repository", "", "", repoValidator)
		}
		repo, _ = expandToAbsoluteURl(repo, true) // Safe to ignore error cuz it's either was validated or predefined starter
	}

	return repo, starter
}

func expandToAbsoluteURl(repo string, guessShortUrls bool) (string, error) {
	if parsedUrl, err := url.ParseRequestURI(repo); err == nil {
		return parsedUrl.String(), nil
	} else if guessShortUrls {
		repo = strings.TrimSuffix(repo, ".git")
		splitRepo := strings.Split(repo, "/")
		switch len(splitRepo) {
		case 2:
			repo = fmt.Sprintf(GITHUB_REPO_TPL, splitRepo[0], splitRepo[1])
		case 1:
			repo = fmt.Sprintf(GITHUB_REPO_TPL, "enonic", repo)
		}
		return expandToAbsoluteURl(repo, false)
	} else {
		return "", errors.New("Not a valid repository")
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
	semver7 = semver.MustParse(common.MIN_XP_VERSION)
	for _, ver := range versions {
		latestSemver, err = semver.NewVersion(ver)
		util.Warn(err, fmt.Sprintf("Could not parse version '%s'", ver))

		if !latestSemver.LessThan(semver7) {
			return true
		}
	}
	return false
}

func lookupStarterDocs(c *cli.Context, repo string) *Starter {

	gitUrl := strings.TrimSuffix(repo, ".git")
	body := new(bytes.Buffer)
	params := map[string]interface{}{
		"query": MARKET_DOCS_REQUEST,
		"variables": map[string]string{
			"query": fmt.Sprintf(MARKET_DOCS_QUERY_TPL, gitUrl),
		},
	}
	json.NewEncoder(body).Encode(params)

	req := common.CreateRequest(c, "POST", common.MARKET_URL, body)
	res, err := common.SendRequestCustom(req, "", 1)
	if err != nil {
		return nil
	}

	var result StartersResponse
	common.ParseResponse(res, &result)

	starters := result.Data.Market.Query
	if len(starters) == 1 {
		return &starters[0]
	} else {
		return nil
	}
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
		Auth:       auth,
		URL:        url,
		Progress:   os.Stderr,
		RemoteName: UPSTREAM_NAME,
	})
	util.Fatal(err, fmt.Sprintf("Could not connect to a remote repository '%s':", url))

	if branch != "" || hash != "" {

		if err2 := repo.Fetch(&git.FetchOptions{
			RemoteName: UPSTREAM_NAME,
			RefSpecs:   []config.RefSpec{"+refs/*:refs/*"},
		}); err2 != nil && err2.Error() != git.NoErrAlreadyUpToDate.Error() {
			fmt.Fprintf(os.Stderr, "Could not fetch remote repo: %s", err2.Error())
			os.Exit(1)
		}

		tree, err := repo.Worktree()
		util.Fatal(err, "Could not get repo tree")

		err3 := tree.Checkout(getCheckoutOpts(repo, hash, branch))
		util.Fatal(err3, fmt.Sprintf("Could not checkout hash [%s] and branch [%s]:", hash, branch))
	}
}

func getCheckoutOpts(repo *git.Repository, hash, branch string) *git.CheckoutOptions {
	if hash != "" {
		// verify hash exists
		if _, err := repo.CommitObject(plumbing.NewHash(hash)); err != nil {
			fmt.Fprintf(os.Stderr, "Could not find commit with hash %s: %s\n", hash, err.Error())
			os.Exit(1)
		}

		return &git.CheckoutOptions{
			Hash: plumbing.NewHash(hash),
		}
	}
	if branch != "" {
		// verify branch exists
		if exist, err := isRemoteBranchExist(repo, branch); !exist {
			fmt.Fprintf(os.Stderr, "Could not find branch %s: %v\n", branch, err)
			os.Exit(1)
		}

		return &git.CheckoutOptions{
			Branch: plumbing.NewBranchReferenceName(branch),
		}
	}
	return &git.CheckoutOptions{}
}

func isRemoteBranchExist(repo *git.Repository, branch string) (bool, error) {
	origin, err := repo.Remote(UPSTREAM_NAME)
	if err != nil {
		return false, err
	}

	refs, err := origin.List(&git.ListOptions{})
	if err != nil {
		return false, err
	}

	for _, ref := range refs {
		if ref.Name().IsBranch() && ref.Name().Short() == branch {
			return true, nil
		}
	}

	return false, nil
}

func clearGitData(dest string) {
	os.RemoveAll(filepath.Join(dest, ".git"))
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
