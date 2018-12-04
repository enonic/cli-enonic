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
	"strings"
	"github.com/Masterminds/semver"
	"os/exec"
	"github.com/enonic/xp-cli/internal/app/util/system"
)

const DISTRO_REGEXP = "^enonic-xp-([-_.a-zA-Z0-9]+)$"
const DISTRO_TEMPLATE = "enonic-xp-%s"
const REMOTE_DISTRO_URL = "http://repo.enonic.com/public/com/enonic/xp/distro-%s-jdk/%s/%s"
const REMOTE_DISTRO_NAME = "distro-%s-jdk-%s.zip"
const REMOTE_DISTRO_HASH = "distro-%s-jdk-%s.zip.md5"

func ensureDistroPresent(version string) string {
	var (
		latestHash string
		data       SandboxesData
		outdated   bool
	)
	osName := util.GetCurrentOs()

	if version == "latest" {
		fmt.Fprint(os.Stderr, "Checking remote latest distro version...")
		latestHash = downloadDistroHash(osName, "latest")
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
				return filepath.Join(getDistrosDir(), distro)
			}
		}
		fmt.Fprintln(os.Stderr, "Not found")
	}

	zipPath := downloadDistro(osName, version)

	distroPath := unzipDistro(zipPath, version)

	err := os.Remove(zipPath)
	util.Warn(err, "Could not delete distro zip file: ")

	if outdated {
		// save latest hash after everything's done
		data.Latest = latestHash
		writeSandboxesData(data)
	}

	return distroPath
}

func downloadDistroHash(osName, version string) string {
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

func printDownloadProgress(path string, total int64) {
	for {
		file, err := os.Open(path)
		util.Fatal(err, "Could not open download file: ")

		fi, err2 := file.Stat()
		util.Fatal(err2, "Could not read download file: ")

		size := fi.Size()
		if total <= size {
			fmt.Fprintln(os.Stderr, "Done")
			break
		}

		var percent = float64(size) / float64(total) * 100

		fmt.Fprintf(os.Stderr, "\rDownloading distro (%.0f %% of %d bytes)...", percent, total)

		time.Sleep(time.Second)
	}
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

	go printDownloadProgress(fullPath, resp.ContentLength)
	defer resp.Body.Close()

	// Write the body to file
	_, err = io.Copy(zipFile, resp.Body)
	util.Fatal(err, "Could not save distro: ")

	time.Sleep(time.Second) // Make sure printDownloadProgress finished because it refreshes every 1 sec

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

func startDistro(version, sandbox string, detach bool) *exec.Cmd {
	executable := "server.sh"
	if runtime.GOOS == "windows" {
		executable = "server.bat"
	}
	appPath := filepath.Join(getDistrosDir(), fmt.Sprintf(DISTRO_TEMPLATE, version), "bin", executable)
	homePath := GetSandboxHomePath(sandbox)
	args := []string{fmt.Sprintf("-Dxp.home=%s", homePath)}

	return system.Start(appPath, args, detach)
}

func stopDistro(pid int) {
	err := system.KillAll(pid)
	util.Fatal(err, fmt.Sprintf("Could not stop process %d", pid))
}

func deleteDistro(version string) {
	err := os.RemoveAll(filepath.Join(getDistrosDir(), fmt.Sprintf(DISTRO_TEMPLATE, version)))
	util.Warn(err, fmt.Sprintf("Could not delete distro '%s' folder: ", version))

	if version == "latest" {
		data := readSandboxesData()
		data.Latest = ""
		writeSandboxesData(data)
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

func ensureVersionCorrect(version string) string {
	return util.PromptUntilTrue(version, func(val string, i byte) string {
		if len(strings.TrimSpace(val)) == 0 {
			if i == 0 {
				return "Enter distro version for this sandbox: "
			} else {
				return "Distro version can not be empty: "
			}
		} else {
			if val != "latest" {
				if version, err := semver.NewVersion(val); err != nil || version == nil {
					return fmt.Sprintf("Version '%s' does not seem to be a valid version: ", val)
				}
			}
			return ""
		}
	})
}

func getDistrosDir() string {
	return filepath.Join(util.GetHomeDir(), ".enonic", "distributions")
}
