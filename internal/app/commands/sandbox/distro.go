package sandbox

import (
	"fmt"
	"github.com/AlecAivazis/survey"
	"github.com/Masterminds/semver"
	"github.com/enonic/cli-enonic/internal/app/commands/common"
	"github.com/enonic/cli-enonic/internal/app/util"
	"github.com/enonic/cli-enonic/internal/app/util/system"
	"gopkg.in/cheggaaa/pb.v1"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

const DISTRO_FOLDER_NAME_REGEXP = "^enonic-xp-(?:windows|mac|linux)-(?:sdk|server)-([-_.a-zA-Z0-9]+)$"
const DISTRO_FOLDER_NAME_TPL = "enonic-xp-%s-sdk-%s"
const DISTRO_LIST_NAME_REGEXP = "^(?:windows|mac|linux)-(?:sdk|server)-([-_.a-zA-Z0-9]+)$"
const DISTRO_LIST_NAME_TPL = "%s-sdk-%s"
const REMOTE_DISTRO_URL = "http://repo.enonic.com/public/com/enonic/xp/enonic-xp-%s-sdk/%s/%s"
const REMOTE_VERSION_URL = "http://repo.enonic.com/api/search/versions?g=com.enonic.xp&a=enonic-xp-%s-sdk"
const SNAP_ENV_VAR = "SNAP_USER_COMMON"

type VersionResult struct {
	Version     string `json:version`
	Integration bool   `json:integration`
}

type VersionsResult struct {
	Results []VersionResult `json:results`
}

func EnsureDistroExists(distroName string) (string, bool) {
	distroVersion := parseDistroVersion(distroName, true)

	for _, distro := range listDistros() {
		if distroVer := parseDistroVersion(distro, false); distroVer == distroVersion {
			return filepath.Join(getDistrosDir(), distro), false
		}
	}

	osName := util.GetCurrentOs()
	zipPath := downloadDistro(osName, distroVersion)

	distroPath := unzipDistro(zipPath)

	if osName == "linux" {
		if snapCommon, snapExists := os.LookupEnv(SNAP_ENV_VAR); snapExists {
			createSymLink(snapCommon, distroName)
		}
	}

	err := os.Remove(zipPath)
	util.Warn(err, "Could not delete distro zip file: ")

	return distroPath, true
}

func createSymLink(snapCommon, distroName string) {
	symLink := getDistroSymLinkPath(snapCommon, distroName)
	distro := getDistroExecutablePath(distroName)

	symLinkDir := filepath.Dir(symLink)
	if _, err := os.Stat(symLinkDir); os.IsNotExist(err) {
		err = os.MkdirAll(symLinkDir, os.ModeDir)
		util.Fatal(err, fmt.Sprintf("Could not create directory '%s'", symLinkDir))
	}

	err := os.Symlink(distro, symLink)
	util.Fatal(err, fmt.Sprintf("Error creating a symbolic link to distro %s:", distroName))
}

func getAllVersions(osName string) []VersionResult {

	req, err := http.NewRequest("GET", fmt.Sprintf(REMOTE_VERSION_URL, osName), nil)
	resp := common.SendRequest(req, "Loading")
	util.Fatal(err, "Could not load latest version for os: "+osName)

	var versions VersionsResult
	common.ParseResponse(resp, &versions)

	var filteredVersions []VersionResult
	for _, value := range versions.Results {
		tempVersion, tempErr := semver.NewVersion(value.Version)
		util.Warn(tempErr, "Could not parse distro version: "+value.Version)

		// excluding only SNAPSHOTS
		if strings.ToUpper(tempVersion.Prerelease()) != "SNAPSHOT" {
			filteredVersions = append(filteredVersions, value)
		}
	}

	return filteredVersions
}

func findLatestVersion(versions []VersionResult) string {

	var latestVer *semver.Version
	for _, version := range versions {
		if tempVer, err := semver.NewVersion(version.Version); err == nil {
			if latestVer == nil || latestVer.LessThan(tempVer) {
				latestVer = tempVer
			}
		} else {
			util.Warn(err, "Could not parse remote distro version: ")
		}
	}

	return latestVer.String()
}

func createProgressBar(total int64) *pb.ProgressBar {
	bar := pb.New(int(total))
	bar.ShowSpeed = false
	bar.ShowCounters = false
	bar.ShowPercent = true
	bar.ShowTimeLeft = false
	bar.ShowElapsedTime = false
	bar.ShowFinalTime = false
	bar.Prefix("Downloading distro ").SetUnits(pb.U_BYTES_DEC).SetRefreshRate(200 * time.Millisecond).Start()
	return bar
}

func downloadDistro(osName, version string) string {
	distroName := formatDistroVersion(version, osName, false) + ".zip"

	fullPath := filepath.Join(getDistrosDir(), distroName)

	zipFile, err := os.Create(fullPath)
	util.Fatal(err, "Could not save distro: ")
	defer zipFile.Close()

	// Get the data

	url := fmt.Sprintf(REMOTE_DISTRO_URL, osName, version, distroName)

	req, err := http.NewRequest("GET", url, nil)
	util.Fatal(err, "Could not create request to: "+url)

	resp, err := common.SendRequestCustom(req, "Loading", 15)
	if err != nil || resp.StatusCode != 200 {
		message := resp.Status
		if err != nil {
			message = err.Error()
		}
		fmt.Fprintf(os.Stderr, "Could not load distro '%s' from remote server: %s\n", version, message)
		os.Exit(1)
	}

	pBar := createProgressBar(resp.ContentLength)
	defer resp.Body.Close()

	// Write the body to file
	_, err = io.Copy(zipFile, pBar.NewProxyReader(resp.Body))
	util.Fatal(err, "Could not save distro: ")

	pBar.Finish()

	return fullPath
}

func unzipDistro(zipFile string) string {
	fmt.Fprint(os.Stderr, "Unzipping distro...")

	zipDir := filepath.Dir(zipFile)
	unzippedFiles := util.Unzip(zipFile, zipDir)

	sourceName := unzippedFiles[0] // distro zip contains only 1 root dir which is the first unzipped file
	targetPath := filepath.Join(zipDir, sourceName)

	fmt.Fprintln(os.Stderr, "Done")

	return targetPath
}

func getDistroExecutablePath(distroName string) string {
	myOs := util.GetCurrentOs()
	var executable string
	if myOs == "windows" {
		executable = "server.bat"
	} else {
		executable = "server.sh"
	}
	version := parseDistroVersion(distroName, true)
	return filepath.Join(getDistrosDir(), formatDistroVersion(version, myOs, false), "bin", executable)
}

func getDistroSymLinkPath(snapCommon, distroName string) string {
	myOs := util.GetCurrentOs()
	version := parseDistroVersion(distroName, true)
	return filepath.Join(snapCommon, "dot-enonic", "distributions", formatDistroVersion(version, myOs, false), "startDistro")
}

func startDistro(distroName, sandbox string, detach, devMode bool) *exec.Cmd {
	myOs := util.GetCurrentOs()
	var argsTemplate, appPath string
	if myOs == "windows" {
		argsTemplate = `-Dxp.home="%s"` // quotes are needed for windows to understand spaces in path
		appPath = getDistroExecutablePath(distroName)
	} else {
		argsTemplate = `-Dxp.home=%s` // other OSes work ok without em
		if snapCommon, snapExists := os.LookupEnv(SNAP_ENV_VAR); snapExists {
			appPath = getDistroSymLinkPath(snapCommon, distroName)
		} else {
			appPath = getDistroExecutablePath(distroName)
		}
	}
	homePath := GetSandboxHomePath(sandbox)
	args := []string{fmt.Sprintf(argsTemplate, homePath)}
	if devMode {
		args = append(args, "dev")
	}

	return system.Start(appPath, args, detach)
}

func stopDistro(pid int) {
	err := system.KillAll(pid)
	util.Fatal(err, fmt.Sprintf("Could not stop process %d", pid))
}

func deleteDistro(distroName string) {
	myOs := util.GetCurrentOs()
	distroVersion := parseDistroVersion(distroName, true)
	err := os.RemoveAll(filepath.Join(getDistrosDir(), formatDistroVersion(distroVersion, myOs, false)))
	util.Warn(err, fmt.Sprintf("Could not delete distro '%s' folder: ", distroName))

	if myOs == "linux" {
		if snapCommon, snapExists := os.LookupEnv(SNAP_ENV_VAR); snapExists {
			// delete the symlink as well
			err = os.Remove(getDistroSymLinkPath(snapCommon, distroName))
			util.Warn(err, fmt.Sprintf("Could not delete symbolic link for '%s': ", distroName))
		}
	}

}

func listDistros() []string {
	distrosDir := getDistrosDir()
	files, err := ioutil.ReadDir(distrosDir)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Could not list distros: ", err)
	}
	return filterDistros(files, distrosDir)
}

func filterDistros(vs []os.FileInfo, distrosDir string) []string {
	vsf := make([]string, 0)
	for _, v := range vs {
		if isDistro(v) {
			vsf = append(vsf, v.Name())
		} else {
			if err := os.RemoveAll(filepath.Join(distrosDir, v.Name())); err != nil {
				fmt.Fprintln(os.Stderr, "Could not remove invalid distro: ", err)
			}
		}
	}
	return vsf
}

func isDistro(v os.FileInfo) bool {
	distroRegexp := regexp.MustCompile(DISTRO_FOLDER_NAME_REGEXP)
	return v.IsDir() && distroRegexp.MatchString(v.Name())
}

func parseDistroVersion(distro string, isDisplay bool) string {
	var distroRegexp *regexp.Regexp
	if isDisplay {
		distroRegexp = regexp.MustCompile(DISTRO_LIST_NAME_REGEXP)
	} else {
		distroRegexp = regexp.MustCompile(DISTRO_FOLDER_NAME_REGEXP)
	}
	match := distroRegexp.FindStringSubmatch(distro)
	if len(match) == 2 {
		return match[1]
	} else {
		return ""
	}
}

func formatDistroVersion(version, myOs string, isDisplay bool) string {
	var tpl string
	if isDisplay {
		tpl = DISTRO_LIST_NAME_TPL
	} else {
		tpl = DISTRO_FOLDER_NAME_TPL
	}
	return fmt.Sprintf(tpl, myOs, version)
}

func GetDistroJdkPath(distroName string) string {
	myOs := util.GetCurrentOs()
	distroVersion := parseDistroVersion(distroName, true)
	return filepath.Join(getDistrosDir(), formatDistroVersion(distroVersion, myOs, false), "jdk")
}

func ensureVersionCorrect(versionStr string) string {
	var (
		version       *semver.Version
		versionErr    error
		versionExists bool
	)

	if len(strings.TrimSpace(versionStr)) > 0 {
		if version, versionErr = semver.NewVersion(versionStr); versionErr != nil {
			fmt.Fprintf(os.Stderr, "'%s' is not a valid distro version.", versionStr)
		}
	}

	myOs := util.GetCurrentOs()
	versions := getAllVersions(myOs)
	textVersions := make([]string, len(versions))
	for key, value := range versions {
		textVersions[key] = formatDistroVersion(value.Version, myOs, true)
		if version != nil && version.String() == value.Version {
			versionExists = true
		}
	}

	if version != nil || versionExists {

		return version.String()
	} else {

		defaultVersion := findLatestVersion(versions)
		var distro string
		err := survey.AskOne(&survey.Select{
			Message:  "Enonic XP distribution:",
			Options:  textVersions,
			Default:  formatDistroVersion(defaultVersion, myOs, true),
			PageSize: 10,
		}, &distro, nil)
		util.Fatal(err, "Distribution select error: ")

		return parseDistroVersion(distro, true)
	}

}

func getDistrosDir() string {
	return filepath.Join(util.GetHomeDir(), ".enonic", "distributions")
}
