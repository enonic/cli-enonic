package sandbox

import (
	"cli-enonic/internal/app/commands/common"
	"cli-enonic/internal/app/commands/remote"
	"cli-enonic/internal/app/util"
	"cli-enonic/internal/app/util/system"
	"encoding/xml"
	"fmt"
	"github.com/Masterminds/semver"
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

const DISTRO_FOLDER_NAME_REGEXP = "^enonic-xp-(?:windows|mac|mac-arm64|linux|linux-arm64)-(?:sdk|server)-([-_.a-zA-Z0-9]+)$"
const DISTRO_FOLDER_NAME_TPL = "enonic-xp-%s-sdk-%s"

// old CLI used this format to store distro name in sandbox, new format will apply after first sandbox distro change
const OLD_DISTRO_FOLDER_NAME_REGEXP = "^(?:windows|mac|linux)-(?:sdk|server)-([-_.a-zA-Z0-9]+)$"

const DISTRO_LIST_NAME_REGEXP = "^(?:windows|mac|mac-arm64|linux|linux-arm64)-(?:sdk|server)-([-_.a-zA-Z0-9]+) \\([ ,a-zA-Z]+\\)$"
const SANDBOX_LIST_NAME_TPL = "%s (%s-sdk-%s)"
const DISTRO_LIST_NAME_TPL = "%s-sdk-%s %s"

const REMOTE_DISTRO_URL = "https://repo.enonic.com/public/com/enonic/xp/enonic-xp-%s-sdk/%s/%s"
const REMOTE_VERSION_URL = "https://repo.enonic.com/public/com/enonic/xp/enonic-xp-%s-sdk/maven-metadata.xml"

const TGZ_SUPPORTED_FROM_VERSION = "7.6.0"
const TGZ_MAC_SUPPORTED_FROM_VERSION = "7.10.0"

type Metadata struct {
	XMLName    xml.Name   `xml:"metadata"`
	GroupId    string     `xml:"groupId"`
	ArtifactId string     `xml:"artifactId"`
	Versioning Versioning `xml:"versioning"`
}

type Versioning struct {
	XMLName     xml.Name `xml:"versioning"`
	Latest      string   `xml:"latest"`
	Release     string   `xml:"release"`
	LastUpdated string   `xml:"lastUpdated"`
	Versions    []string `xml:"versions>version"`
}

func EnsureDistroExists(distroName string) (string, bool) {
	for _, distro := range listDistros() {
		if distroName == distro {
			return filepath.Join(getDistrosDir(), distro), false
		}
	}

	distroVersion := parseDistroVersion(distroName, false)
	zipPath := downloadDistro(distroVersion)

	distroPath := unzipDistro(zipPath)

	err := os.Remove(zipPath)
	util.Warn(err, "Could not delete distro zip file: ")

	return distroPath, true
}

func getAllVersions(osName, minDistro string, includeMinVer, includeUnstable bool) ([]string, string) {

	req, err := http.NewRequest("GET", fmt.Sprintf(REMOTE_VERSION_URL, osName), nil)
	resp := common.SendRequest(req, "Loading")
	util.Fatal(err, "Could not load latest version for os: "+osName)
	fmt.Fprintln(os.Stderr, "Done")

	var minDistroVer *semver.Version
	if minDistro != "" {
		minDistroVer, err = semver.NewVersion(minDistro)
	}

	var metadata Metadata
	common.ParseResponseXml(resp, &metadata)

	var filteredVersions []string
	var latestVersionResult string
	var latestVersion *semver.Version
	for _, version := range metadata.Versioning.Versions {
		tempVersion, tempErr := semver.NewVersion(version)
		util.Warn(tempErr, "Could not parse distro version: "+version)

		minVersionPasses := minDistroVer == nil ||
			tempVersion.GreaterThan(minDistroVer) ||
			includeMinVer && tempVersion.Equal(minDistroVer)
		// excluding only SNAPSHOTS
		if minVersionPasses &&
			strings.ToUpper(tempVersion.Prerelease()) != "SNAPSHOT" &&
			(includeUnstable || tempVersion.Prerelease() == "") {

			filteredVersions = append(filteredVersions, version)
			if latestVersion == nil || latestVersion.LessThan(tempVersion) {
				latestVersion = tempVersion
				latestVersionResult = version
			}
		}
	}

	return filteredVersions, latestVersionResult
}

func findLatestVersion(versions []string) string {

	var latestVer *semver.Version
	for _, version := range versions {
		if tempVer, err := semver.NewVersion(version); err == nil {
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

func resolveArchiveExtension(version string) string {
	currentOS := util.GetCurrentOs()
	if currentOS == "windows" {
		return ".zip"
	}

	tgzVersion := semver.MustParse(TGZ_SUPPORTED_FROM_VERSION)
	macTgzVersion := semver.MustParse(TGZ_MAC_SUPPORTED_FROM_VERSION)
	currentVersion := semver.MustParse(version)

	if (currentOS == "mac" && currentVersion.LessThan(macTgzVersion)) || currentVersion.LessThan(tgzVersion) {
		return ".zip"
	} else {
		return ".tgz"
	}
}

func downloadDistro(version string) string {
	distroName := formatDistroVersion(version) + resolveArchiveExtension(version)

	fullPath := filepath.Join(getDistrosDir(), distroName)

	zipFile, err := os.Create(fullPath)
	util.Fatal(err, "Could not save distro: ")
	defer zipFile.Close()

	// Get the data
	url := fmt.Sprintf(REMOTE_DISTRO_URL, util.GetCurrentOsWithArch(), version, distroName)

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

	var unzippedFiles []string
	if strings.HasSuffix(zipFile, ".zip") {
		unzippedFiles = util.Unzip(zipFile, zipDir)
	} else {
		unzippedFiles = util.Untar(zipFile, zipDir)
	}

	sourceName := unzippedFiles[0] // distro zip contains only 1 root dir which is the first unzipped file
	targetPath := filepath.Join(zipDir, sourceName)

	fmt.Fprintln(os.Stderr, "Done")

	return targetPath
}

func startDistro(distroName, sandbox string, detach, devMode, debug bool) *exec.Cmd {
	var executable, homeTemplate string
	if util.GetCurrentOs() == "windows" {
		executable = "server.bat"
		homeTemplate = `-Dxp.home="%s"` // quotes are needed for windows to understand spaces in path
	} else {
		executable = "server.sh"
		homeTemplate = `-Dxp.home=%s` // other OSes work ok without em
	}
	version := parseDistroVersion(distroName, false)
	appPath := filepath.Join(getDistrosDir(), formatDistroVersion(version), "bin", executable)
	homePath := GetSandboxHomePath(sandbox)
	var args []string
	if debug {
		// should go as 1st param !
		args = append(args, "debug")
	}
	if devMode {
		args = append(args, "dev")
	}
	args = append(args, fmt.Sprintf(homeTemplate, homePath))

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
	distroVersion := parseDistroVersion(distroName, false)
	err := os.RemoveAll(filepath.Join(getDistrosDir(), formatDistroVersion(distroVersion)))
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
	return (v.IsDir() || (v.Mode()&os.ModeSymlink == os.ModeSymlink)) && distroRegexp.MatchString(v.Name())
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

func formatDistroVersion(version string) string {
	return fmt.Sprintf(DISTRO_FOLDER_NAME_TPL, util.GetCurrentOsWithArch(), version)
}

func formatSandboxListItemName(boxName, version, myOs string) string {
	var tpl = SANDBOX_LIST_NAME_TPL

	return fmt.Sprintf(tpl, boxName, myOs, version)
}

func formatDistroVersionDisplay(version, myOs, latestVersion string, markStability bool) string {
	var tpl = DISTRO_LIST_NAME_TPL

	stability := assessVersionStability(version, latestVersion, markStability)
	return fmt.Sprintf(tpl, myOs, version, stability)
}

func assessVersionStability(versionStr, latestVersion string, markStability bool) string {
	version, err := semver.NewVersion(versionStr)
	if err != nil {
		return "unknown"
	}

	var stability []string
	if versionStr == latestVersion {
		stability = append(stability, "latest")
	}
	if markStability {
		if version.Prerelease() != "" {
			stability = append(stability, "unstable")
		} else {
			stability = append(stability, "stable")
		}
	}
	if len(stability) > 0 {
		return fmt.Sprintf("(%s)", strings.Join(stability, ", "))
	} else {
		return ""
	}
}

func GetDistroJdkPath(distroName string) string {
	distroVersion := parseDistroVersion(distroName, false)
	return filepath.Join(getDistrosDir(), formatDistroVersion(distroVersion), "jdk")
}

func EnsureSanboxSupportsProjectVersion(sBox *Sandbox, minDistroVersion *semver.Version) {
	sandboxDistroVer := semver.MustParse(parseDistroVersion(sBox.Distro, false))
	if sandboxDistroVer.LessThan(minDistroVersion) {
		fmt.Fprintf(os.Stderr, "The project requires XP %v or higher. Associated sandbox '%s' uses XP %v.\nUpgrade sandbox XP version with 'enonic sandbox upgrade'\nor set a different project sandbox with 'enonic project sandbox'.\n", minDistroVersion, sBox.Name, sandboxDistroVer)
		os.Exit(1)
	}
}

func ensureVersionCorrect(versionStr, minDistroVer string, includeMinVer, includeUnstable, force bool) (string, int) {
	var (
		version       *semver.Version
		versionErr    error
		versionExists bool
	)

	if len(strings.TrimSpace(versionStr)) > 0 {
		if version, versionErr = semver.NewVersion(versionStr); versionErr != nil {
			fmt.Fprintf(os.Stderr, "'%s' is not a valid distro version.\n", versionStr)
			if force {
				os.Exit(1)
			}
		}
	}

	currentOsWithArch := util.GetCurrentOsWithArch()
	shouldIncludeUnstable := includeUnstable || version != nil && version.Prerelease() != ""
	versions, latestVersion := getAllVersions(currentOsWithArch, minDistroVer, includeMinVer, shouldIncludeUnstable)
	totalVersions := len(versions)

	if totalVersions == 0 {
		return "", 0
	} else if totalVersions == 1 {
		return versions[0], totalVersions
	}

	textVersions := make([]string, len(versions))
	for key, value := range versions {
		textVersions[key] = formatDistroVersionDisplay(value, currentOsWithArch, latestVersion, shouldIncludeUnstable)
		if version != nil && version.String() == value {
			versionExists = true
		}
	}

	if version != nil && versionExists {
		return version.String(), totalVersions
	} else {
		if force {
			if version == nil {
				fmt.Fprintln(os.Stderr, "Version flag can not be empty in non-interactive mode.")
			} else {
				fmt.Fprintf(os.Stderr, "Version '%s' can not be found.\n", versionStr)
			}
			os.Exit(1)
		}

		useLatest := util.PromptBool(fmt.Sprintf("Do you want to use Enonic XP %s %s",
			latestVersion, assessVersionStability(latestVersion, latestVersion, shouldIncludeUnstable)),
			true)
		if useLatest {
			return latestVersion, totalVersions
		}

		distro, _, err := util.PromptSelect(&util.SelectOptions{
			Message:  "Enonic XP distribution",
			Default:  formatDistroVersionDisplay(latestVersion, currentOsWithArch, latestVersion, shouldIncludeUnstable),
			Options:  textVersions,
			PageSize: 10,
		})
		util.Fatal(err, "Could not select distro: ")

		return parseDistroVersion(distro, true), totalVersions
	}

}

func getDistrosDir() string {
	return common.GetInEnonicDir("distributions")
}
