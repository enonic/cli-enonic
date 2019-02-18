package util

import (
	"bytes"
	"encoding/json"
	"bufio"
	"os"
	"fmt"
	"runtime"
	"github.com/mitchellh/go-homedir"
	"archive/zip"
	"path/filepath"
	"io"
	"net"
	"github.com/BurntSushi/toml"
	"time"
	"strings"
	"github.com/AlecAivazis/survey"
)

func PrettyPrintJSONBytes(b []byte) ([]byte, error) {
	var out bytes.Buffer
	err := json.Indent(&out, b, "", "    ")
	return out.Bytes(), err
}

func PrettyPrintJSON(data interface{}) string {
	var out = new(bytes.Buffer)
	enc := json.NewEncoder(out)
	enc.SetIndent("", "    ")
	if err := enc.Encode(data); err != nil {
		return "Not a valid JSON: " + err.Error()
	}
	return out.String()
}

func PromptString(text, val, defaultVal string, validator func(val interface{}) error) string {
	if err := validator(val); err == nil {
		return val
	}

	prompt := &survey.Input{
		Message: text,
		Default: defaultVal,
	}

	survey.AskOne(prompt, &val, validator)

	return val
}

func PromptBool(text string, defaultVal bool) bool {
	var val bool

	prompt := &survey.Confirm{
		Message: text,
		Default: defaultVal,
	}

	survey.AskOne(prompt, &val, nil)

	return val
}

func PromptUntilTrue(val string, assessFunc func(val *string, i byte) string) string {
	index := byte(0)
	text := assessFunc(&val, index)
	for text != "" {
		reader := bufio.NewScanner(os.Stdin)
		fmt.Fprint(os.Stderr, text)
		reader.Scan()
		val = reader.Text()
		index += 1
		text = assessFunc(&val, index)
	}
	return val
}

func checkError(err error, msg string, fatal bool) {
	if err != nil {
		fmt.Fprintln(os.Stderr, msg, err.Error())
		if fatal {
			os.Exit(1)
		}
	}
}

func Warn(err error, msg string) {
	checkError(err, msg, false)
}

func Fatal(err error, msg string) {
	checkError(err, msg, true)
}

func GetCurrentOs() string {
	osName := runtime.GOOS
	if osName == "darwin" {
		osName = "mac"
	}
	return strings.ToLower(osName)
}

func GetHomeDir() string {
	home, err := homedir.Dir()
	Fatal(err, "Could not get user home dir: ")
	return home
}

func Unzip(zipFile, destFolder string) []string {
	reader, err := zip.OpenReader(zipFile)
	Fatal(err, "Could not open zip file: ")
	defer reader.Close()

	unzipped := make([]string, 0)
	for _, f := range reader.File {
		cloneZipItem(f, destFolder)
		unzipped = append(unzipped, f.Name)
	}

	return unzipped
}

func cloneZipItem(f *zip.File, destFolder string) {
	destPath := filepath.Join(destFolder, f.Name)
	err := os.MkdirAll(filepath.Dir(destPath), os.ModeDir|os.ModePerm)
	Fatal(err, fmt.Sprintf("Could not create folder '%s': ", destPath))

	// Clone if item is a file
	reader, err := f.Open()
	Fatal(err, fmt.Sprintf("Could not read file '%s'", f.Name))

	if !f.FileInfo().IsDir() {
		fileCopy, err := os.OpenFile(destPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.FileInfo().Mode())
		Fatal(err, fmt.Sprintf("Could not create file '%s", f.Name))

		_, err = io.Copy(fileCopy, reader)
		fileCopy.Close()
		Fatal(err, fmt.Sprintf("Could not write file '%s", f.Name))
	}
	reader.Close()
}

func IsPortAvailable(port uint16) bool {
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err == nil {
		defer ln.Close()
	}
	return err == nil
}

func IndexOf(element string, data []string) (int) {
	for k, v := range data {
		if element == v {
			return k
		}
	}
	return -1
}

func OpenOrCreateDataFile(path string, readOnly bool) *os.File {
	flags := os.O_CREATE
	if readOnly {
		flags |= os.O_RDONLY
	} else {
		flags |= os.O_WRONLY | os.O_TRUNC
	}
	file, err := os.OpenFile(path, flags, 0640)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Could not open file: ", err)
		os.Exit(1)
	}
	return file
}

func DecodeTomlFile(file *os.File, data interface{}) {
	if _, err := toml.DecodeReader(bufio.NewReader(file), data); err != nil {
		fmt.Fprintln(os.Stderr, "Could not parse toml file: ", err)
		os.Exit(1)
	}
}

func EncodeTomlFile(file *os.File, data interface{}) {
	if err := toml.NewEncoder(bufio.NewWriter(file)).Encode(data); err != nil {
		fmt.Fprintln(os.Stderr, "Could not encode toml file: ", err)
		os.Exit(1)
	}
}

func TimeFromNow(start time.Time) time.Duration {
	return time.Now().Round(time.Second).Sub(start.Round(time.Second))
}
