package common

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Masterminds/semver"
	"github.com/briandowns/spinner"
	"github.com/enonic/cli-enonic/internal/app/commands/remote"
	"github.com/enonic/cli-enonic/internal/app/util"
	"github.com/enonic/cli-enonic/internal/app/util/system"
	"github.com/magiconair/properties"
	"github.com/mitchellh/go-ps"
	"github.com/urfave/cli"
	"io"
	"io/ioutil"
	"math"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

const MIN_XP_VERSION = "7.0.0"
const ENV_XP_HOME = "XP_HOME"
const ENV_JAVA_HOME = "JAVA_HOME"
const MARKET_URL = "https://market.enonic.com/api/graphql"
const SCOOP_MANIFEST_URL = "https://raw.githubusercontent.com/enonic/cli-scoop/master/enonic.json"
const JSESSIONID = "JSESSIONID"
const LATEST_CHECK_MSG = "Last version check was %d days ago. Run 'enonic latest' to check for newer CLI version"
const LATEST_VERSION_MSG = "Latest available version is %s. Run '%s' to update CLI"
const CLI_DOWNLOAD_URL = "https://repo.enonic.com/public/com/enonic/cli/enonic/%[1]s/enonic_%[1]s_%[2]s_64-bit.%[3]s"

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

func ReadProjectDistroVersion(prjPath string) string {
	if props, _ := properties.LoadFile(filepath.Join(prjPath, "gradle.properties"), properties.UTF8); props != nil {
		return props.GetString("xpVersion", MIN_XP_VERSION)
	} else {
		return MIN_XP_VERSION
	}
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

func VerifyRuntimeData(rData *RuntimeData) bool {
	if rData.PID == 0 {
		if rData.Running != "" {
			rData.Running = ""
			WriteRuntimeData(*rData)
		}
		return false
	} else {
		// make sure that process is still alive and has the same name
		proc, _ := ps.FindProcess(rData.PID)
		if proc != nil {
			detachedName := system.GetDetachedProcName()
			if match, _ := regexp.MatchString("^(?:enonic|"+detachedName+")(?:.exe)?$", proc.Executable()); match {
				return true
			}
		}
		// process is either nil, or PID is taken by other process already, so erase its info
		rData.PID = 0
		rData.Running = ""
		WriteRuntimeData(*rData)
		return false
	}
}

func WriteRuntimeData(data RuntimeData) {
	path := filepath.Join(GetEnonicDir(), ".enonic")
	file := util.OpenOrCreateDataFile(path, false)
	defer file.Close()

	util.EncodeTomlFile(file, data)
}

func EnsureAuth(authString string) (string, string) {
	var splitAuth []string
	util.PromptPassword("Authentication token (<user>:<password>): ", authString, func(val interface{}) error {
		str := val.(string)
		if len(strings.TrimSpace(str)) == 0 {
			return errors.New("authentication token can not be empty")
		} else {
			splitAuth = strings.Split(str, ":")
			if len(splitAuth) != 2 || len(splitAuth[0]) == 0 {
				return errors.New("authentication token must have the following format <user>:<password>")
			}
		}
		return nil
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
	activeRemote := remote.GetActiveRemote()
	var client *http.Client
	if activeRemote.Proxy != nil {
		client = &http.Client{
			Timeout:   timeoutMin * time.Minute,
			Transport: &http.Transport{Proxy: http.ProxyURL(&activeRemote.Proxy.URL)},
		}
	} else {
		client = &http.Client{
			Timeout: timeoutMin * time.Minute,
		}
	}
	if message != "" {
		spin.Prefix = message
		spin.FinalMSG = "\r" + message + "..." //r fixes empty spaces before final msg on windows
		spin.Start()
	}

	// make a copy of request body prior to sending cuz it vanishes after!
	bodyCopy := copyBody(req)

	res, err := client.Do(req)
	if message != "" {
		spin.Stop()
	}
	if err != nil {
		return nil, err
	}

	rData := ReadRuntimeData()
	switch res.StatusCode {
	case http.StatusOK:
		for _, cookie := range res.Cookies() {
			if cookie.Name == JSESSIONID && cookie.Value != rData.SessionID {
				rData.SessionID = cookie.Value
				WriteRuntimeData(rData)
			}
		}
	case http.StatusForbidden:
		if rData.SessionID != "" {
			fmt.Fprint(os.Stderr, "User session is not valid.")
			rData.SessionID = ""
			WriteRuntimeData(rData)
		}

		var auth string
		user, pass, _ := res.Request.BasicAuth()
		activeRemote := remote.GetActiveRemote()
		if user == "" && pass == "" {
			if activeRemote.User != "" {
				fmt.Fprintln(os.Stderr, "Using environment defined user and password.")
				auth = fmt.Sprintf("%s:%s", activeRemote.User, activeRemote.Pass)
			} else {
				fmt.Fprintln(os.Stderr, "")
			}
		} else {
			if activeRemote.User != "" {
				fmt.Fprintln(os.Stderr, "Environment defined user and password are not valid.")
			} else {
				fmt.Fprintln(os.Stderr, "User and password are not valid.")
			}
			auth = ""
		}
		user, pass = EnsureAuth(auth)
		fmt.Fprintln(os.Stderr, "")

		newReq := doCreateRequest(req.Method, req.URL.String(), user, pass, bodyCopy)
		// need to set it for install requests, because their content type may vary
		newReq.Header.Set("Content-Type", req.Header.Get("Content-Type"))
		res, err = SendRequestCustom(newReq, message, timeoutMin)
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
	enonicErr, err := ParseResponseCustom(resp, target)
	if enonicErr != nil {
		fmt.Fprintf(os.Stderr, "%d %s\n", enonicErr.Status, enonicErr.Message)
		os.Exit(1)
	} else if err != nil {
		fmt.Fprint(os.Stderr, "Error parsing response ", err)
		os.Exit(1)
	}
}

func ParseResponseCustom(resp *http.Response, target interface{}) (*EnonicError, error) {
	defer resp.Body.Close()
	decoder := json.NewDecoder(resp.Body)
	if resp.StatusCode == http.StatusOK {
		if err := decoder.Decode(target); err != nil {
			return nil, err
		}
	} else {
		var enonicError EnonicError
		if err := decoder.Decode(&enonicError); err == nil && enonicError.Message != "" {
			return &enonicError, nil
		} else {
			return nil, errors.New(resp.Status)
		}
	}
	return nil, nil
}

func ProduceCheckVersionFunction(appVersion string) func() string {
	return func() string {
		var message string

		rData := ReadRuntimeData()

		if rData.LatestVersion == "" {
			rData.LatestCheck = time.Now()
			rData.LatestVersion = appVersion
			WriteRuntimeData(rData)
		}

		daysSinceLastCheck := time.Since(rData.LatestCheck).Hours() / 24
		if daysSinceLastCheck > 30 {
			message = fmt.Sprintf(LATEST_CHECK_MSG, int(math.Floor(daysSinceLastCheck)))
		} else {
			latestVer := semver.MustParse(rData.LatestVersion)
			currentVer := semver.MustParse(appVersion)
			if latestVer.GreaterThan(currentVer) {
				message = FormatLatestVersionMessage(rData.LatestVersion)
			}
		}

		return message
	}
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
		return "brew upgrade enonic"
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
