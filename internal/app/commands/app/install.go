package app

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/enonic/cli-enonic/internal/app/commands/common"
	"github.com/enonic/cli-enonic/internal/app/util"
	"github.com/pkg/errors"
	"github.com/urfave/cli"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var Install = cli.Command{
	Name:    "install",
	Aliases: []string{"i"},
	Usage:   "Install an application from URL or file",
	Flags: append([]cli.Flag{
		cli.StringFlag{
			Name:  "url, u",
			Usage: "The URL of the application",
		},
		cli.StringFlag{
			Name:  "file, f",
			Usage: "Application file",
		},
	}, common.AUTH_FLAG, common.FORCE_FLAG),
	Action: func(c *cli.Context) error {

		file, url := ensureURLOrFileFlag(c)

		installApp(c, file, url)

		return nil
	},
}

func installApp(c *cli.Context, file, url string) InstallResult {
	req := createInstallRequest(c, file, url)

	resp := common.SendRequest(req, fmt.Sprintf("Installing \"%s\"", file))

	var result InstallResult
	common.ParseResponse(resp, &result)
	if fail := result.Failure; fail != "" {
		fmt.Fprintln(os.Stderr, fail)
	} else {
		fmt.Fprintln(os.Stderr, "Done")
	}
	fmt.Fprintln(os.Stdout, util.PrettyPrintJSON(result))

	return result
}

func InstallFromFile(c *cli.Context, file string) InstallResult {
	return installApp(c, file, "")
}

func InstallFromUrl(c *cli.Context, url string) InstallResult {
	return installApp(c, "", url)
}

func ensureURLOrFileFlag(c *cli.Context) (string, string) {
	urlString := strings.TrimSpace(c.String("u"))
	fileString := strings.TrimSpace(c.String("f"))

	if urlString == "" && fileString == "" || fileString != "" {
		return ensureFileFlag(c), ""
	} else {
		return "", ensureURLFlag(c)
	}
}

func ensureURLFlag(c *cli.Context) string {
	force := common.IsForceMode(c)
	urlValidator := func(val interface{}) error {
		str := val.(string)
		if len(strings.TrimSpace(str)) == 0 {
			if force {
				fmt.Fprintln(os.Stderr, "URL can not be empty in non-interactive mode.")
				os.Exit(1)
			}
			return errors.New("URL can not be empty")
		} else {
			if _, err := url.ParseRequestURI(str); err != nil {
				if force {
					fmt.Fprintf(os.Stderr, "URL '%s' is not valid", str)
					os.Exit(1)
				}
				return errors.Errorf("URL '%s' is not valid: ", str)
			}
			return nil
		}
	}

	return util.PromptString("Enter URL", c.String("u"), "", urlValidator)
}

func ensureFileFlag(c *cli.Context) string {
	return util.PromptProjectJar(c.String("f"), common.IsForceMode(c))
}

func createInstallRequest(c *cli.Context, filePath, urlParam string) *http.Request {
	body := new(bytes.Buffer)
	var baseUrl, contentType string

	if filePath != "" {
		baseUrl = "app/install"

		file, err := os.Open(filePath)
		util.Fatal(err, "Error opening file:")
		defer file.Close()

		writer := multipart.NewWriter(body)
		part, _ := writer.CreateFormFile("file", filepath.Base(file.Name()))
		io.Copy(part, file)
		contentType = writer.FormDataContentType()
		writer.Close()
	} else if urlParam != "" {
		baseUrl = "app/installUrl"
		contentType = "application/json"
		params := map[string]string{
			"URL": urlParam,
		}
		json.NewEncoder(body).Encode(params)
	} else {
		panic("Either file or URL is required")
	}

	req := common.CreateRequest(c, "POST", baseUrl, body)
	req.Header.Set("Content-Type", contentType)
	return req
}

type InstallResult struct {
	ApplicationInstalledJson struct {
		Application struct {
			DisplayName      string
			Key              string
			Deletable        bool
			Editable         bool
			Local            bool
			MaxSystemVersion string
			MinSystemVersion string
			ModifiedTime     time.Time
			State            string
			Url              string
			VendorName       string
			VendorUrl        string
			Version          string
		}
	}
	Failure string
}
