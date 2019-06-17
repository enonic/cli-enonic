package common

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/Masterminds/semver"
	"github.com/briandowns/spinner"
	"github.com/enonic/cli-enonic/internal/app/commands/remote"
	"github.com/enonic/cli-enonic/internal/app/util"
	"github.com/urfave/cli"
	"io"
	"io/ioutil"
	"math"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
)

var ENV_XP_HOME = "XP_HOME"
var ENV_JAVA_HOME = "JAVA_HOME"
var MARKET_URL = "https://market.enonic.com/api/graphql"
var SCOOP_MANIFEST_URL = "https://raw.githubusercontent.com/enonic/cli-scoop/master/enonic.json"
var JSESSIONID = "JSESSIONID"
var LATEST_CHECK_MSG = "Last version check was %d days ago. Run 'enonic latest' to check for newer CLI version"
var LATEST_VERSION_MSG = "Latest available version is %s. Run '%s' to update CLI"
var CLI_DOWNLOAD_URL = "https://repo.enonic.com/public/com/enonic/cli/enonic/%[1]s/enonic_%[1]s_%[2]s_64-bit.%[3]s"
var spin *spinner.Spinner

func init() {
	spin = spinner.New(spinner.CharSets[26], 300*time.Millisecond)
	spin.Writer = os.Stderr
}

var FLAGS = []cli.Flag{
	cli.StringFlag{
		Name:  "auth, a",
		Usage: "Authentication token for basic authentication (user:password)",
	},
}

type ProjectData struct {
	Sandbox string `toml:"sandbox"`
}

type RuntimeData struct {
	Running       string    `toml:"running"`
	PID           int       `toml:"PID"`
	SessionID     string    `toml:sessionID`
	LatestVersion string    `toml:latestVersion`
	LatestCheck   time.Time `toml:latestCheck`
}

func GetEnonicDir() string {
	return filepath.Join(util.GetHomeDir(), ".enonic")
}

func HasProjectData(prjPath string) bool {
	if stat, err := os.Stat(path.Join(prjPath, ".enonic")); err == nil && !stat.IsDir() {
		return true
	}
	return false
}

func ReadProjectData(prjPath string) *ProjectData {
	file := util.OpenOrCreateDataFile(filepath.Join(prjPath, ".enonic"), true)
	defer file.Close()

	var data ProjectData
	util.DecodeTomlFile(file, &data)
	return &data
}

func WriteProjectData(data *ProjectData, prjPath string) {
	file := util.OpenOrCreateDataFile(filepath.Join(prjPath, ".enonic"), false)
	defer file.Close()

	util.EncodeTomlFile(file, data)
}

func ReadRuntimeData() RuntimeData {
	path := filepath.Join(GetEnonicDir(), ".enonic")
	file := util.OpenOrCreateDataFile(path, true)
	defer file.Close()

	var data RuntimeData
	util.DecodeTomlFile(file, &data)
	return data
}

func WriteRuntimeData(data RuntimeData) {
	path := filepath.Join(GetEnonicDir(), ".enonic")
	file := util.OpenOrCreateDataFile(path, false)
	defer file.Close()

	util.EncodeTomlFile(file, data)
}

func EnsureAuth(authString string) (string, string) {
	var splitAuth []string
	util.PromptUntilTrue(authString, func(val *string, ind byte) string {
		if len(strings.TrimSpace(*val)) == 0 {
			switch ind {
			case 0:
				return "Enter authentication token (<user>:<password>): "
			default:
				return "Authentication token can not be empty (<user>:<password>): "
			}
		} else {
			splitAuth = strings.Split(*val, ":")
			if len(splitAuth) != 2 {
				return fmt.Sprintf("Authentication token '%s' must have the following format <user>:<password>: ", *val)
			} else {
				return ""
			}
		}
	})
	return splitAuth[0], splitAuth[1]
}

func CreateRequest(c *cli.Context, method, url string, body io.Reader) *http.Request {
	var auth, user, pass string
	if c != nil {
		auth = c.String("auth")
	}

	if url != MARKET_URL && url != SCOOP_MANIFEST_URL && (ReadRuntimeData().SessionID == "" || auth != "") {
		if auth == "" {
			activeRemote := remote.GetActiveRemote()
			if activeRemote.User != "" || activeRemote.Pass != "" {
				auth = fmt.Sprintf("%s:%s", activeRemote.User, activeRemote.Pass)
			}
		}
		user, pass = EnsureAuth(auth)
	}

	return doCreateRequest(method, url, user, pass, body)
}

func doCreateRequest(method, reqUrl, user, pass string, body io.Reader) *http.Request {
	var (
		host, scheme, port, path string
	)

	parsedUrl, err := url.Parse(reqUrl)
	util.Fatal(err, "Not a valid url: "+reqUrl)

	if parsedUrl.IsAbs() {
		host = parsedUrl.Hostname()
		port = parsedUrl.Port()
		scheme = parsedUrl.Scheme
		path = parsedUrl.Path
	} else {
		activeRemote := remote.GetActiveRemote()
		host = activeRemote.Url.Hostname()
		port = activeRemote.Url.Port()
		scheme = activeRemote.Url.Scheme

		runeUrl := []rune(reqUrl)
		if runeUrl[0] == '/' {
			// absolute path
			path = reqUrl
		} else {
			// relative path
			path = activeRemote.Url.Path + "/" + reqUrl
		}
	}

	req, err := http.NewRequest(method, fmt.Sprintf("%s://%s:%s%s", scheme, host, port, path), body)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Params error: ", err)
		os.Exit(1)
	}
	req.Header.Set("Content-Type", "application/json")

	rData := ReadRuntimeData()
	if user != "" {
		req.SetBasicAuth(user, pass)

		if rData.SessionID != "" {
			rData.SessionID = ""
			WriteRuntimeData(rData)
		}
	} else if rData.SessionID != "" {
		req.AddCookie(&http.Cookie{
			Name:  JSESSIONID,
			Value: rData.SessionID,
		})
	}

	return req
}

func SendRequest(req *http.Request, message string) *http.Response {
	res, err := SendRequestCustom(req, message, 1)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Unable to connect to remote service: ", err)
		os.Exit(1)
	}
	return res
}

func SendRequestCustom(req *http.Request, message string, timeoutMin time.Duration) (*http.Response, error) {
	client := &http.Client{
		Timeout: timeoutMin * time.Minute,
	}
	if message != "" {
		spin.Prefix = message
		spin.FinalMSG = "\r" + message + "..." //r fixes empty spaces before final msg on windows
		spin.Start()
		defer spin.Stop()
	}
	bodyCopy := copyBody(req)
	res, err := client.Do(req)

	rData := ReadRuntimeData()
	switch res.StatusCode {
	case http.StatusForbidden:

		if rData.SessionID != "" {
			fmt.Fprintln(os.Stderr, "Session is no longer valid.")
			rData.SessionID = ""
			WriteRuntimeData(rData)
		}

		var auth string
		user, pass, _ := res.Request.BasicAuth()
		if user == "" && pass == "" {
			activeRemote := remote.GetActiveRemote()
			if activeRemote.User != "" || activeRemote.Pass != "" {
				auth = fmt.Sprintf("%s:%s", activeRemote.User, activeRemote.Pass)
			}
		} else {
			fmt.Fprintln(os.Stderr, "Environment defined user and password are not valid.")
		}
		user, pass = EnsureAuth(auth)

		newReq := doCreateRequest(req.Method, req.URL.String(), user, pass, bodyCopy)
		res, err = SendRequestCustom(newReq, message, timeoutMin)

	case http.StatusOK:

		for _, cookie := range res.Cookies() {
			if cookie.Name == JSESSIONID && cookie.Value != rData.SessionID {
				rData.SessionID = cookie.Value
				WriteRuntimeData(rData)
			}
		}
	}

	return res, err
}

func copyBody(req *http.Request) io.ReadCloser {
	if req.Body == nil {
		return nil
	}
	buf, _ := ioutil.ReadAll(req.Body)
	req.Body = ioutil.NopCloser(bytes.NewBuffer(buf))
	return ioutil.NopCloser(bytes.NewBuffer(buf))
}

func ParseResponse(resp *http.Response, target interface{}) {
	defer resp.Body.Close()
	decoder := json.NewDecoder(resp.Body)
	if resp.StatusCode == http.StatusOK {
		if err := decoder.Decode(target); err != nil {
			fmt.Fprint(os.Stderr, "Error parsing response ", err)
			os.Exit(1)
		}
	} else {
		var enonicError EnonicError
		if err := decoder.Decode(&enonicError); err == nil && enonicError.Message != "" {
			fmt.Fprintf(os.Stderr, "%d %s\n", enonicError.Status, enonicError.Message)
		} else {
			fmt.Fprintln(os.Stderr, resp.Status)
		}
		os.Exit(1)
	}
}

func PopulateMeta(app *cli.App) map[string]interface{} {
	meta := make(map[string]interface{})
	var message string

	rData := ReadRuntimeData()
	if rData.LatestCheck.IsZero() {
		// this is the first check so set it to now
		rData.LatestCheck = time.Now()
		rData.LatestVersion = app.Version
		WriteRuntimeData(rData)
	}

	daysSinceLastCheck := time.Since(rData.LatestCheck).Hours() / 24
	if daysSinceLastCheck > 30 {
		message = fmt.Sprintf(LATEST_CHECK_MSG, int(math.Floor(daysSinceLastCheck)))
	} else {
		latestVer := semver.MustParse(rData.LatestVersion)
		currentVer := semver.MustParse(app.Version)
		if latestVer.GreaterThan(currentVer) {
			message = FormatLatestVersionMessage(rData.LatestVersion)
		}
	}

	if message != "" {
		meta["Message"] = message
	}

	return meta
}

func FormatLatestVersionMessage(latest string) string {
	return fmt.Sprintf(LATEST_VERSION_MSG, latest, getOSUpdateCommand())
}

func getOSDownloadUrl(version string) string {
	os := util.GetCurrentOs()
	var ext string
	switch os {
	case "windows":
		ext = "zip"
	default:
		ext = "tar.gz"
	}
	return fmt.Sprintf(CLI_DOWNLOAD_URL, version, strings.Title(os), ext)
}

func getOSUpdateCommand() string {
	switch util.GetCurrentOs() {
	case "windows":
		return "scoop update enonic"
	case "mac":
		return "brew update enonic"
	case "linux":
		return "snap refresh enonic"
	default:
		return ""
	}
}

type EnonicError struct {
	Status  uint16 `json:status`
	Message string `json:message`
	Context struct {
		Authenticated bool     `json:authenticated`
		Principals    []string `json:principals`
	} `json:context`
}
