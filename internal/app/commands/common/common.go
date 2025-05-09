package common

import (
	"bytes"
	"cli-enonic/internal/app/commands/remote"
	"cli-enonic/internal/app/util"
	"cli-enonic/internal/app/util/system"
	"crypto/tls"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"github.com/Masterminds/semver"
	"github.com/briandowns/spinner"
	"github.com/magiconair/properties"
	"github.com/mitchellh/go-ps"
	"github.com/urfave/cli"
	"io"
	"math"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"
)

const MIN_XP_VERSION = "7.0.0"
const ENV_XP_HOME = "XP_HOME"
const ENV_JAVA_HOME = "JAVA_HOME"
const MARKET_URL = "https://market.enonic.com/api/graphql"
const SCOOP_MANIFEST_URL = "https://raw.githubusercontent.com/enonic/cli-scoop/master/enonic.json"
const JSESSIONID = "JSESSIONID"
const LATEST_CHECK_MSG = "Last version check was %d days ago. Run 'enonic latest' to check for newer CLI version"
const LATEST_VERSION_MSG = "\nLatest available version is %s. Run '%s' to update CLI"
const SNAP_ENV_VAR = "SNAP_USER_COMMON"
const FORCE_COOKIE = "forceFlag"
const HTTP_PORT = 8080
const INFO_PORT = 2609
const MGMT_PORT = 4848
const MODE_DEV = "dev"
const MODE_DEFAULT = "default"

var spin *spinner.Spinner

func init() {
	spin = spinner.New(spinner.CharSets[26], 300*time.Millisecond)
	spin.Writer = os.Stderr
}

var AUTH_FLAG = cli.StringFlag{
	Name:  "auth, a",
	Usage: "Authentication token for basic authentication (user:password)",
}

var CRED_FILE_FLAG = cli.StringFlag{
	Name:  "cred-file",
	Usage: "The path to the service account key file (in JSON format). This is only available for XP version 7.15 and later. Key file can be generated by Users application for System ID Provider users (aka Service Accounts). If specified, the flag \"--auth\" or \"-a\" will be ignored",
}

var FORCE_FLAG = cli.BoolFlag{
	Name:  "force, f",
	Usage: "Accept default answers to all prompts and run non-interactively",
}

var CLIENT_KEY_FLAG = cli.StringFlag{
	Name:  "client-key",
	Usage: "Specifies the private key file for client certificate authentication. This option is used in conjunction with --client-cert to establish a mutual TLS (mTLS) session.",
}

var CLIENT_CERT_FLAG = cli.StringFlag{
	Name:  "client-cert",
	Usage: "Specifies the client certificate file to use for authentication with the remote server. Requires --client-key to be specified as well when establishing a mutual TLS (mTLS) session.",
}

var AUTH_AND_TLS_FLAGS = []cli.Flag{
	AUTH_FLAG,
	CRED_FILE_FLAG,
	CLIENT_KEY_FLAG,
	CLIENT_CERT_FLAG,
}

func IsForceMode(c *cli.Context) bool {
	return c != nil && c.Bool("force")
}

type ProjectData struct {
	Sandbox string `toml:"sandbox"`
}

type RuntimeData struct {
	Running       string    `toml:"running"`
	Mode          string    `toml:"mode"`
	PID           int       `toml:"PID"`
	SessionID     string    `toml:sessionID`
	LatestVersion string    `toml:latestVersion`
	LatestCheck   time.Time `toml:latestCheck`
}

type MarketResponse[K any] struct {
	Data struct {
		Market struct {
			Query []K
		}
	}
}

func GetInEnonicDir(children ...string) string {
	var joinArgs []string
	if util.GetCurrentOs() == "linux" {
		if snapCommon, snapExists := os.LookupEnv(SNAP_ENV_VAR); snapExists {
			joinArgs = []string{snapCommon, "dot-enonic"}
		}
	}
	if joinArgs == nil {
		joinArgs = []string{util.GetHomeDir(), ".enonic"}
	}
	if len(children) > 0 {
		joinArgs = append(joinArgs, children...)
	}
	return filepath.Join(joinArgs...)
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

func ReadGradlePropertiesFile(path string) (*properties.Properties, error) {
	return properties.LoadFile(filepath.Join(path, "gradle.properties"), properties.UTF8)
}

func ReadProjectDistroVersion(prjPath string) string {
	if props, _ := ReadGradlePropertiesFile(prjPath); props != nil {
		return props.GetString("xpVersion", MIN_XP_VERSION)
	} else {
		return MIN_XP_VERSION
	}
}

func ReadProjectName(prjPath string) string {
	if props, _ := ReadGradlePropertiesFile(prjPath); props != nil {
		return props.GetString("projectName", "")
	}
	return ""
}

func WriteProjectData(data *ProjectData, prjPath string) {
	file := util.OpenOrCreateDataFile(filepath.Join(prjPath, ".enonic"), false)
	defer file.Close()

	util.EncodeTomlFile(file, data)
}

func ReadRuntimeData() RuntimeData {
	enonicPath := GetInEnonicDir(".enonic")
	file := util.OpenOrCreateDataFile(enonicPath, true)
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
	enonicPath := GetInEnonicDir(".enonic")
	file := util.OpenOrCreateDataFile(enonicPath, false)
	defer file.Close()

	util.EncodeTomlFile(file, data)
}

func EnsureAuth(authString string, force bool) (string, string) {
	var splitAuth []string
	util.PromptPassword("Authentication token (<user>:<password>): ", authString, func(val interface{}) error {
		str := val.(string)
		if len(strings.TrimSpace(str)) == 0 {
			if force {
				fmt.Fprintln(os.Stderr, "Authentication token can not be empty in non-interactive mode")
				os.Exit(1)
			}
			return errors.New("authentication token can not be empty")
		} else {
			splitAuth = strings.Split(str, ":")
			if len(splitAuth) != 2 || len(splitAuth[0]) == 0 {
				if force {
					fmt.Fprintln(os.Stderr, "Authentication token must have the following format <user>:<password>")
					os.Exit(1)
				}
				return errors.New("authentication token must have the following format <user>:<password>")
			}
		}
		return nil
	})
	return splitAuth[0], splitAuth[1]
}

func getValueOrDefault(path string, defaultValue string) string {
	if path == "" {
		path = defaultValue
	}
	return path
}

func resolveCredFilePath(path string) string {
	return getValueOrDefault(path, os.Getenv("ENONIC_CLI_CRED_FILE"))
}

func CreateRequest(c *cli.Context, method, url string, body io.Reader) *http.Request {
	var auth, user, pass, credFilePath string
	if c != nil {
		auth = c.String("auth")
		credFilePath = resolveCredFilePath(c.String("cred-file"))
	}

	if url != MARKET_URL && url != SCOOP_MANIFEST_URL && (ReadRuntimeData().SessionID == "" || auth != "" || credFilePath != "") {
		if credFilePath != "" {
			jwtToken := generateServiceAccountJwtToken(credFilePath)
			return doCreateRequestBearerAuthRequest(method, url, jwtToken, body)
		} else {
			if auth == "" {
				activeRemote := remote.GetActiveRemote()
				if activeRemote.User != "" || activeRemote.Pass != "" {
					auth = fmt.Sprintf("%s:%s", activeRemote.User, activeRemote.Pass)
				}
			}
			user, pass = EnsureAuth(auth, IsForceMode(c))
		}
	}

	return doCreateBasicAuthRequest(method, url, user, pass, body, IsForceMode(c))
}

func doCreateSimpleRequest(method, reqUrl string, body io.Reader) *http.Request {
	var (
		host, scheme, port, restPath string
	)

	parsedUrl, err := url.Parse(reqUrl)
	util.Fatal(err, fmt.Sprintf("Not a valid url: %s", reqUrl))

	if parsedUrl.IsAbs() {
		host = parsedUrl.Hostname()
		port = parsedUrl.Port()
		scheme = parsedUrl.Scheme
		restPath = parsedUrl.Path
	} else {
		activeRemote := remote.GetActiveRemote()
		host = activeRemote.Url.Hostname()
		port = activeRemote.Url.Port()
		scheme = activeRemote.Url.Scheme

		runeUrl := []rune(reqUrl)
		if runeUrl[0] == '/' {
			// absolute path
			restPath = reqUrl
		} else {
			// relative path
			restPath = activeRemote.Url.Path + "/" + reqUrl
		}
	}

	req, err := http.NewRequest(method, fmt.Sprintf("%s://%s:%s%s", scheme, host, port, restPath), body)
	util.Fatal(err, "Params error: ")

	return req
}

func doCreateRequestBearerAuthRequest(method, url, jwtToken string, body io.Reader) *http.Request {
	req := doCreateSimpleRequest(method, url, body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", jwtToken))
	return req
}

func doCreateBasicAuthRequest(method, reqUrl, user, pass string, body io.Reader, force bool) *http.Request {
	req := doCreateSimpleRequest(method, reqUrl, body)

	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: FORCE_COOKIE, Value: strconv.FormatBool(force)})

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

func SendRequest(c *cli.Context, req *http.Request, message string) *http.Response {
	res, err := SendRequestCustom(c, req, message, 1)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Unable to connect to remote service: ", err)
		os.Exit(1)
	}
	return res
}

func SendRequestCustom(c *cli.Context, req *http.Request, message string, timeoutMin time.Duration) (*http.Response, error) {
	var isCredFileAbsent bool
	if c != nil {
		isCredFileAbsent = resolveCredFilePath(c.String("cred-file")) == ""
	}

	tlsKey := getValueOrDefault(c.String(CLIENT_KEY_FLAG.Name), os.Getenv("ENONIC_CLI_CLIENT_KEY"))
	tlsCert := getValueOrDefault(c.String(CLIENT_CERT_FLAG.Name), os.Getenv("ENONIC_CLI_CLIENT_CERT"))

	var tlsConfig *tls.Config
	if tlsKey != "" && tlsCert != "" {
		cert, err := tls.LoadX509KeyPair(tlsCert, tlsKey)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Failed to load client certificate: ", err)
			os.Exit(1)
		}

		tlsConfig = &tls.Config{
			Certificates: []tls.Certificate{cert},
		}
	}

	var transport *http.Transport

	activeRemote := remote.GetActiveRemote()

	client := &http.Client{
		Timeout: timeoutMin * time.Minute,
	}
	if activeRemote.Proxy != nil {
		transport = &http.Transport{
			Proxy: http.ProxyURL(&activeRemote.Proxy.URL),
		}
		if tlsConfig != nil {
			transport.TLSClientConfig = tlsConfig
		}
		client.Transport = transport
	} else {
		if tlsConfig != nil {
			transport = &http.Transport{
				TLSClientConfig: tlsConfig,
			}
		}
	}

	if transport != nil {
		client.Transport = transport
	}

	if message != "" {
		StartSpinner(message)
	}

	// make a copy of request body prior to sending cuz it vanishes after!
	bodyCopy := copyBody(req)

	res, err := client.Do(req)
	if message != "" {
		StopSpinner()
	}
	if err != nil {
		return nil, err
	}

	rData := ReadRuntimeData()
	if res.StatusCode >= 200 && res.StatusCode < 300 {
		if isCredFileAbsent {
			for _, cookie := range res.Cookies() {
				if cookie.Name == JSESSIONID && cookie.Value != rData.SessionID {
					rData.SessionID = cookie.Value
					WriteRuntimeData(rData)
				}
			}
		}
	} else if res.StatusCode == http.StatusForbidden {
		if isCredFileAbsent {
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
			forceCookie, cookieError := res.Request.Cookie(FORCE_COOKIE)
			util.Warn(cookieError, fmt.Sprintf("Could not read '%s' cookie", FORCE_COOKIE))
			forceBool, boolError := strconv.ParseBool(forceCookie.Value)
			util.Warn(boolError, fmt.Sprintf("Could not parse '%s' cookie value: %s", FORCE_COOKIE, forceCookie.Value))

			if forceBool {
				// Just exit cuz there's no way we can ask new auth in non-interactive mode
				os.Exit(1)
			}

			user, pass = EnsureAuth(auth, forceBool)
			fmt.Fprintln(os.Stderr, "")

			newReq := CreateRequest(c, req.Method, req.URL.String(), bodyCopy)
			// need to set it for install requests, because their content type may vary
			newReq.Header.Set("Content-Type", req.Header.Get("Content-Type"))
			res, err = SendRequestCustom(c, newReq, message, timeoutMin)
		}
	}

	return res, err
}

func StartSpinner(message string) {
	spin.Prefix = message
	spin.FinalMSG = "\r" + message + "..." //r fixes empty spaces before final msg on windows
	spin.Start()
}

func StopSpinner() {
	spin.Stop()
}

func copyBody(req *http.Request) io.ReadCloser {
	if req.Body == nil {
		return nil
	}
	buf, _ := io.ReadAll(req.Body)
	req.Body = io.NopCloser(bytes.NewBuffer(buf))
	return io.NopCloser(bytes.NewBuffer(buf))
}

func ParseResponse(resp *http.Response, target interface{}) {
	enonicErr, err := ParseResponseCustom(resp, target)
	if enonicErr != nil {
		fmt.Fprintf(os.Stderr, "Failure: %s\n", enonicErr.Message)
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

func ParseResponseXml(resp *http.Response, target interface{}) {
	enonicErr, err := ParseResponseXmlCustom(resp, target)
	if enonicErr != nil {
		fmt.Fprintf(os.Stderr, "%d %s\n", enonicErr.Status, enonicErr.Message)
		os.Exit(1)
	} else if err != nil {
		fmt.Fprint(os.Stderr, "Error parsing response ", err)
		os.Exit(1)
	}
}

func ParseResponseXmlCustom(resp *http.Response, target interface{}) (*EnonicError, error) {
	defer resp.Body.Close()

	decoder := xml.NewDecoder(resp.Body)
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

func IsInstalledViaNPM() bool {
	_, _, code := getCommandResult("npm", "list", "-g", "@enonic/cli")
	// npm list return exit code 1 if package is not installed and 0 if it is
	return code == 0
}

func GetLatestNPMVersion() string {
	version, _, _ := getCommandResult("npm", "view", "@enonic/cli", "version")
	// the output is version followed by a newline: 2.4.0-RC2\n
	return strings.Trim(version, "\r\n")
}

func getCommandResult(command string, args ...string) (string, string, int) {
	var stdout, stderr bytes.Buffer
	var exitCode int

	cmd := exec.Command(command, args...)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			waitStatus := exitError.Sys().(syscall.WaitStatus)
			exitCode = waitStatus.ExitStatus()
		} else {
			exitCode = 1
		}
	}
	outStr, errStr := string(stdout.Bytes()), string(stderr.Bytes())

	return outStr, errStr, exitCode
}

func FormatLatestVersionMessage(latest string) string {
	upgradeCmd := "enonic"
	if util.GetCurrentOs() == "windows" {
		upgradeCmd += ".exe"
	}
	upgradeCmd += " upgrade"
	return fmt.Sprintf(LATEST_VERSION_MSG, latest, upgradeCmd)
}

func GetOSUpdateCommand(isNPM bool) string {
	if isNPM {
		return "npm upgrade -g @enonic/cli"
	}

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

func GetOSUninstallCommand(isNPM bool) string {
	if isNPM {
		return "npm uninstall -g @enonic/cli"
	}

	switch util.GetCurrentOs() {
	case "windows":
		return "scoop uninstall enonic"
	case "mac":
		return "brew uninstall enonic"
	case "linux":
		return "snap remove enonic"
	default:
		return ""
	}
}

type EnonicError struct {
	Status  uint16 `json:"status"`
	Message string `json:"message"`
	Context struct {
		Authenticated bool     `json:"authenticated"`
		Principals    []string `json:"principals"`
	} `json:"context"`
}
