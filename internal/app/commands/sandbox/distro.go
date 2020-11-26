package sandbox

import (
	"fmt"
	"github.com/AlecAivazis/survey"
	"github.com/Masterminds/semver"
	"github.com/enonic/cli-enonic/internal/app/commands/common"
	"github.com/enonic/cli-enonic/internal/app/commands/remote"
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

// old CLI used this format to store distro name in sandbox, new format will apply after first sandbox distro change
const OLD_DISTRO_FOLDER_NAME_REGEXP = "^(?:windows|mac|linux)-(?:sdk|server)-([-_.a-zA-Z0-9]+)$"

const DISTRO_LIST_NAME_REGEXP = "^(?:windows|mac|linux)-(?:sdk|server)-([-_.a-zA-Z0-9]+) \\([ ,a-zA-Z]+\\)$"
const SANDBOX_LIST_NAME_TPL = "%s (%s-sdk-%s)"
const DISTRO_LIST_NAME_TPL = "%s-sdk-%s (%s)"
const REMOTE_DISTRO_URL = "https://repo.enonic.com/public/com/enonic/xp/enonic-xp-%s-sdk/%s/%s"
const REMOTE_VERSION_URL = "https://repo.enonic.com/api/search/versions?g=com.enonic.xp&a=enonic-xp-%s-sdk"

type VersionResult struct {
	Version     string `json:version`
	Integration bool   `json:integration`
}

type VersionsResult struct {
	Results []VersionResult `json:results`
}

func EnsureDistroExists(distroName string) (string, bool) {
	distroVersion := parseDistroVersion(distroName, false)

	for _, distro := range listDistros() {
		if distroVer := parseDistroVersion(distro, false); distroVer == distroVersion {
			return filepath.Join(getDistrosDir(), distro), false
		}
	}

	osName := util.GetCurrentOs()
	zipPath := downloadDistro(osName, distroVersion)

	distroPath := unzipDistro(zipPath)

	err := os.Remove(zipPath)
	util.Warn(err, "Could not delete distro zip file: ")

	return distroPath, true
}

func getAllVersions(osName, minDistro string, includeUnstable bool) ([]VersionResult, VersionResult) {

	req, err := http.NewRequest("GET", fmt.Sprintf(REMOTE_VERSION_URL, osName), nil)
	resp := common.SendRequest(req, "Loading")
	util.Fatal(err, "Could not load latest version for os: "+osName)

	minDistroVer, _ := semver.NewVersion(minDistro)

	var versions VersionsResult
	common.ParseResponse(resp, &versions)

	var filteredVersions []VersionResult
	var latestVersionResult VersionResult
	var latestVersion *semver.Version
	for _, value := range versions.Results {
		tempVersion, tempErr := semver.NewVersion(value.Version)
		util.Warn(tempErr, "Could not parse distro version: "+value.Version)

		// excluding only SNAPSHOTS
		if (minDistroVer == nil || !tempVersion.LessThan(minDistroVer)) &&
			strings.ToUpper(tempVersion.Prerelease()) != "SNAPSHOT" &&
			(includeUnstable || tempVersion.Prerelease() == "") {

			filteredVersions = append(filteredVersions, value)
			if latestVersion == nil || latestVersion.LessThan(tempVersion) {
				latestVersion = tempVersion
				latestVersionResult = value
			}
		}
	}

	return filteredVersions, latestVersionResult
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
	distroName := formatDistroVersion(version, osName) + ".zip"

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

func startDistro(distroName, sandbox string, detach, devMode bool) *exec.Cmd {
	myOs := util.GetCurrentOs()
	var executable, homeTemplate string
	if myOs == "windows" {
		executable = "server.bat"
		homeTemplate = `-Dxp.home="%s"` // quotes are needed for windows to understand spaces in path
	} else {
		executable = "server.sh"
		homeTemplate = `-Dxp.home=%s` // other OSes work ok without em
	}
	version := parseDistroVersion(distroName, false)
	appPath := filepath.Join(getDistrosDir(), formatDistroVersion(version, myOs), "bin", executable)
	homePath := GetSandboxHomePath(sandbox)
	args := []string{fmt.Sprintf(homeTemplate, homePath)}
	if devMode {
		args = append(args, "dev")
	}

	if proxy := remote.GetActiveRemote().Proxy; proxy != nil {
		args = append(args,
			fmt.Sprintf(`-Dhttp.proxyHost=%s`, proxy.Hostname()),
			fmt.Sprintf(`-Dhttp.proxyPort=%s`, proxy.Port()),
			fmt.Sprintf(`-Dhttps.proxyHost=%s`, proxy.Hostname()),
			fmt.Sprintf(`-Dhttps.proxyPort=%s`, proxy.Port()),
		)
	}

	return system.Start(appPath, args, detach)
}

func stopDistro(pid int) {
	err := system.KillAll(pid)
	util.Fatal(err, fmt.Sprintf("Could not stop process %d", pid))
}

func deleteDistro(distroName string) {
	myOs := util.GetCurrentOs()
	distroVersion := parseDistroVersion(distroName, false)
	err := os.RemoveAll(filepath.Join(getDistrosDir(), formatDistroVersion(distroVersion, myOs)))
	util.Warn(err, fmt.Sprintf("Could not delete distro '%s' folder: ", distroName))
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
		if !distroRegexp.MatchString(distro) {
			// check old format in case there's no match
			distroRegexp = regexp.MustCompile(OLD_DISTRO_FOLDER_NAME_REGEXP)
		}
	}
	match := distroRegexp.FindStringSubmatch(distro)
	if len(match) == 2 {
		return match[1]
	} else {
		return ""
	}
}

func formatDistroVersion(version, myOs string) string {
	return fmt.Sprintf(DISTRO_FOLDER_NAME_TPL, myOs, version)
}

func formatSandboxListItemName(boxName, version, myOs string) string {
	var tpl = SANDBOX_LIST_NAME_TPL

	return fmt.Sprintf(tpl, boxName, myOs, version)
}

func formatDistroVersionDisplay(version, myOs, latestVersion string) string {
	var tpl = DISTRO_LIST_NAME_TPL

	stability := assessStability(version, latestVersion)
	return fmt.Sprintf(tpl, myOs, version, stability)
}

func assessStability(versionStr, latestVersion string) string {
	version, err := semver.NewVersion(versionStr)
	if err != nil {
		return "unknown"
	}

	var stability []string
	if versionStr == latestVersion {
		stability = append(stability, "latest")
	}
	if version.Prerelease() != "" {
		stability = append(stability, "unstable")
	} else {
		stability = append(stability, "stable")
	}
	return strings.Join(stability[:], ",")
}

func GetDistroJdkPath(distroName string) string {
	myOs := util.GetCurrentOs()
	distroVersion := parseDistroVersion(distroName, false)
	return filepath.Join(getDistrosDir(), formatDistroVersion(distroVersion, myOs), "jdk")
}

func EnsureSanboxSupportsProjectVersion(sBox *Sandbox, minDistroVersion *semver.Version) {
	sandboxDistroVer := semver.MustParse(parseDistroVersion(sBox.Distro, false))
	if sandboxDistroVer.LessThan(minDistroVersion) {
		fmt.Fprintf(os.Stderr, "The project requires XP %v or higher. Associated sandbox '%s' uses XP %v.\nUpgrade sandbox XP version with 'enonic sandbox upgrade'\nor set a different project sandbox with 'enonic project sandbox'.\n", minDistroVersion, sBox.Name, sandboxDistroVer)
		os.Exit(1)
	}
}

func ensureVersionCorrect(versionStr, minDistroVer string, includeUnstable bool) string {
	var (
		version       *semver.Version
		versionErr    error
		versionExists bool
	)

	if len(strings.TrimSpace(versionStr)) > 0 {
		if version, versionErr = semver.NewVersion(versionStr); versionErr != nil {
			fmt.Fprintf(os.Stderr, "'%s' is not a valid distro version.\n", versionStr)
		}
	}

	myOs := util.GetCurrentOs()
	versions, latest := getAllVersions(myOs, minDistroVer, includeUnstable || version != nil && version.Prerelease() != "")
	textVersions := make([]string, len(versions))
	for key, value := range versions {
		textVersions[key] = formatDistroVersionDisplay(value.Version, myOs, latest.Version)
		if version != nil && version.String() == value.Version {
			versionExists = true
		}
	}

	if version != nil && versionExists {
		return version.String()
	} else {

		defaultVersion := findLatestVersion(versions)
		var distro string
		err := survey.AskOne(&survey.Select{
			Message:  "Enonic XP distribution:",
			Options:  textVersions,
			Default:  formatDistroVersionDisplay(defaultVersion, myOs, latest.Version),
			PageSize: 10,
		}, &distro, nil)
		util.Fatal(err, "Exiting: ")

		return parseDistroVersion(distro, true)
	}

}

func getDistrosDir() string {
	return filepath.Join(util.GetHomeDir(), ".enonic", "distributions")
}
