package sandbox

import (
	"path/filepath"
	"fmt"
	"net/http"
	"io"
	"io/ioutil"
	"os"
	"regexp"
	"github.com/enonic/enonic-cli/internal/app/util"
	"time"
	"os/exec"
	"strings"
	"github.com/Masterminds/semver"
	"github.com/enonic/enonic-cli/internal/app/commands/common"
	"gopkg.in/cheggaaa/pb.v1"
	"github.com/AlecAivazis/survey"
)

const DISTRO_FOLDER_NAME_REGEXP = "^enonic-xp-(?:windows|mac|linux)-(?:sdk|server)-([-_.a-zA-Z0-9]+)$"
const DISTRO_FOLDER_NAME_TPL = "enonic-xp-%s-sdk-%s"
const DISTRO_LIST_NAME_REGEXP = "^(?:windows|mac|linux)-(?:sdk|server)-([-_.a-zA-Z0-9]+)$"
const DISTRO_LIST_NAME_TPL = "%s-sdk-%s"
const REMOTE_DISTRO_URL = "http://repo.enonic.com/public/com/enonic/xp/enonic-xp-%s-sdk/%s/%s"
const REMOTE_VERSION_URL = "http://repo.enonic.com/api/search/versions?g=com.enonic.xp&a=enonic-xp-%s-sdk"

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

	err := os.Remove(zipPath)
	util.Warn(err, "Could not delete distro zip file: ")

	return distroPath, true
}

func getAllVersions(osName string) []VersionResult {
	resp, err := http.Get(fmt.Sprintf(REMOTE_VERSION_URL, osName))
	util.Fatal(err, "Could not load latest version for os: "+osName)

	var versions VersionsResult
	common.ParseResponse(resp, &versions)
	return versions.Results
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

	resp, err := http.Get(url)
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

func startDistro(distroName, sandbox string) *exec.Cmd {
	myOs := util.GetCurrentOs()
	executable := "server.sh"
	if myOs == "windows" {
		executable = "server.bat"
	}
	version := parseDistroVersion(distroName, true)
	appPath := filepath.Join(getDistrosDir(), formatDistroVersion(version, myOs, false), "bin", executable)
	homePath := GetSandboxHomePath(sandbox)

	cmd := exec.Command(appPath, fmt.Sprintf("-Dxp.home=%s", homePath))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	err := cmd.Start()
	util.Fatal(err, fmt.Sprintf("Could not start distro '%s': ", distroName))

	return cmd
}

func deleteDistro(distroName string) {
	myOs := util.GetCurrentOs()
	distroVersion := parseDistroVersion(distroName, true)
	err := os.RemoveAll(filepath.Join(getDistrosDir(), formatDistroVersion(distroVersion, myOs, false)))
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
