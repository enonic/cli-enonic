package sandbox

import (
	"path/filepath"
	"fmt"
	"net/http"
	"io"
	"io/ioutil"
	"os"
	"regexp"
	"github.com/enonic/xp-cli/internal/app/util"
	"time"
	"runtime"
	"os/exec"
	"strings"
	"github.com/Masterminds/semver"
	"github.com/enonic/xp-cli/internal/app/commands/common"
	"gopkg.in/cheggaaa/pb.v1"
	"github.com/AlecAivazis/survey"
)

const DISTRO_REGEXP = "^enonic-xp-([-_.a-zA-Z0-9]+)$"
const DISTRO_TEMPLATE = "enonic-xp-%s"
const REMOTE_DISTRO_URL = "http://repo.enonic.com/public/com/enonic/xp/enonic-xp-%s-sdk/%s/%s"
const REMOTE_VERSION_URL = "http://repo.enonic.com/api/search/versions?g=com.enonic.xp&a=enonic-xp-%s-sdk"
const REMOTE_DISTRO_NAME = "enonic-xp-%s-sdk-%s.zip"

type VersionResult struct {
	Version     string `json:version`
	Integration bool   `json:integration`
}

type VersionsResult struct {
	Results []VersionResult `json:results`
}

func EnsureDistroExists(version string) (string, bool) {
	osName := util.GetCurrentOs()

	for _, distro := range listDistros() {
		if distroVer := getDistroVersion(distro); distroVer == version {
			return filepath.Join(getDistrosDir(), distro), false
		}
	}

	zipPath := downloadDistro(osName, version)

	distroPath := unzipDistro(zipPath, version)

	err := os.Remove(zipPath)
	util.Warn(err, "Could not delete distro zip file: ")

	return distroPath, true
}

func getAllVersions() []VersionResult {
	osName := util.GetCurrentOs()
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
	distroName := fmt.Sprintf(REMOTE_DISTRO_NAME, osName, version)

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

func unzipDistro(zipFile, version string) string {
	fmt.Fprint(os.Stderr, "Unzipping distro...")

	zipDir := filepath.Dir(zipFile)
	unzippedFiles := util.Unzip(zipFile, zipDir)

	sourceName := unzippedFiles[0] // distro zip contains only 1 root dir which is the first unzipped file
	sourcePath := filepath.Join(zipDir, sourceName)
	targetName := fmt.Sprintf(DISTRO_TEMPLATE, version)
	targetPath := filepath.Join(zipDir, targetName)

	if sourcePath != targetPath {
		if _, err := os.Stat(targetPath); !os.IsNotExist(err) {
			os.RemoveAll(targetPath)
		}

		err := os.Rename(sourcePath, targetPath)
		util.Fatal(err, fmt.Sprintf("Could not rename '%s' to '%s'", sourceName, targetName))
	}

	fmt.Fprintln(os.Stderr, "Done")

	return targetPath
}

func startDistro(version, sandbox string) *exec.Cmd {
	executable := "server.sh"
	if runtime.GOOS == "windows" {
		executable = "server.bat"
	}
	appPath := filepath.Join(getDistrosDir(), fmt.Sprintf(DISTRO_TEMPLATE, version), "bin", executable)
	homePath := GetSandboxHomePath(sandbox)

	cmd := exec.Command(appPath, fmt.Sprintf("-Dxp.home=%s", homePath))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	err := cmd.Start()
	util.Fatal(err, fmt.Sprintf("Could not start distro '%s': ", version))

	return cmd
}

func deleteDistro(version string) {
	err := os.RemoveAll(filepath.Join(getDistrosDir(), fmt.Sprintf(DISTRO_TEMPLATE, version)))
	util.Warn(err, fmt.Sprintf("Could not delete distro '%s' folder: ", version))
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
	distroRegexp := regexp.MustCompile(DISTRO_REGEXP)
	return v.IsDir() && distroRegexp.MatchString(v.Name())
}

func getDistroVersion(distro string) string {
	distroRegexp := regexp.MustCompile(DISTRO_REGEXP)
	match := distroRegexp.FindStringSubmatch(distro)
	if len(match) == 2 {
		return match[1]
	} else {
		return ""
	}
}

func GetDistroJdkPath(version string) string {
	return filepath.Join(getDistrosDir(), fmt.Sprintf(DISTRO_TEMPLATE, version), "jdk")
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

	versions := getAllVersions()
	textVersions := make([]string, len(versions))
	for key, value := range versions {
		textVersions[key] = value.Version
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
			Message:  "Distro version:",
			Options:  textVersions,
			Default:  defaultVersion,
			PageSize: 10,
		}, &distro, nil)
		util.Fatal(err, "Distro select error: ")
		return distro
	}

}

func getDistrosDir() string {
	return filepath.Join(util.GetHomeDir(), ".enonic", "distributions")
}
