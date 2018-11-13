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
)

const DISTRO_REGEXP = "^enonic-xp-([-_.a-zA-Z0-9]+)$"
const DISTRO_TEMPLATE = "enonic-xp-%s"
const REMOTE_DISTRO_URL = "http://repo.enonic.com/public/com/enonic/xp/distro-%s-jdk/%s/%s"
const REMOTE_DISTRO_NAME = "distro-%s-jdk-%s.zip"
const REMOTE_DISTRO_HASH = "distro-%s-jdk-%s.zip.md5"

func ensureDistroPresent(version string) {
	var (
		latestHash string
		data       SandboxesData
		outdated   bool
	)
	osName := util.GetCurrentOs()

	if version == "latest" {
		fmt.Fprint(os.Stderr, "Checking remote latest distro version...")
		latestHash = downloadLatestHash(osName, "latest")
		data = readSandboxesData()
		if data.Latest == latestHash {
			fmt.Fprintln(os.Stderr, "Up to date")
		} else {
			fmt.Fprintln(os.Stderr, "Outdated")
			outdated = true
		}
	}

	if !outdated {
		fmt.Fprintf(os.Stderr, "Looking for local distro '%s'...", version)
		for _, distro := range listDistros() {
			if distroVer := getDistroVersion(distro); distroVer == version {
				fmt.Fprintln(os.Stderr, "Found")
				return
			}
		}
		fmt.Fprintln(os.Stderr, "Not found")
	}

	downloadDistro(osName, version)

	if outdated {
		data.Latest = latestHash
		writeSandboxesData(data)
	}
}

func downloadLatestHash(osName, version string) string {
	hashName := fmt.Sprintf(REMOTE_DISTRO_HASH, osName, version)
	url := fmt.Sprintf(REMOTE_DISTRO_URL, osName, version, hashName)

	resp, err := http.Get(url)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Could not access remote server: ", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	bodyBytes, err2 := ioutil.ReadAll(resp.Body)
	if err2 != nil {
		fmt.Fprintln(os.Stderr, "Could not read hash from remote server: ", err)
		os.Exit(1)
	}
	return string(bodyBytes)
}

func downloadDistro(osName, version string) {
	distroName := fmt.Sprintf(REMOTE_DISTRO_NAME, osName, version)

	fullPath := filepath.Join(util.GetHomeDir(), ".enonic", "distributions", distroName)

	zipFile, err := os.Create(fullPath)
	util.Fatal(err, "Could not save distro: ")
	defer zipFile.Close()

	// Get the data

	url := fmt.Sprintf(REMOTE_DISTRO_URL, osName, version, distroName)
	resp, err := http.Get(url)
	util.Fatal(err, "Could not load distro: ")

	fmt.Fprint(os.Stderr, "Downloading distro (it may take several minutes)...")
	defer resp.Body.Close()

	// Write the body to file
	_, err = io.Copy(zipFile, resp.Body)
	util.Fatal(err, "Could not save distro: ")

	fmt.Fprintln(os.Stderr, "Done")

	unzipDistro(fullPath, version)
}

func unzipDistro(zipFile, version string) {
	fmt.Fprint(os.Stderr, "Unzipping distro...")

	zipDir := filepath.Dir(zipFile)
	unzipped := util.Unzip(zipFile, zipDir)

	sourceName := unzipped[0]
	targetName := fmt.Sprintf(DISTRO_TEMPLATE, version)
	err := os.Rename(filepath.Join(zipDir, unzipped[0]), filepath.Join(zipDir, targetName))
	util.Fatal(err, fmt.Sprintf("Could not rename '%s' to '%s'", sourceName, targetName))

	fmt.Fprintln(os.Stderr, "Done")

	err2 := os.Remove(zipFile)
	util.Warn(err2, "Could not delete distro zip file: ")
}

func listDistros() []string {
	distrosDir := filepath.Join(util.GetHomeDir(), ".enonic", "distributions")
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

// Returns os and version
func getDistroVersion(distro string) string {
	distroRegexp := regexp.MustCompile(DISTRO_REGEXP)
	match := distroRegexp.FindStringSubmatch(distro)
	if len(match) == 2 {
		return match[1]
	} else {
		return ""
	}
}
