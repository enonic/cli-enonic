package app

import (
	"github.com/urfave/cli"
	"enonic.com/xp-cli/commands/common"
	"fmt"
	"os"
	"net/http"
	"bytes"
	"encoding/json"
	"enonic.com/xp-cli/util"
	"strings"
	"net/url"
	"mime/multipart"
	"path/filepath"
	"io"
	"time"
)

var Install = cli.Command{
	Name:    "install",
	Aliases: []string{"i"},
	Usage:   "Install an application from URL or file",
	Flags: append([]cli.Flag{
		cli.StringFlag{
			Name:  "u",
			Usage: "The URL of the application",
		},
		cli.StringFlag{
			Name:  "f",
			Usage: "Application file",
		},
	}, common.FLAGS...),
	Action: func(c *cli.Context) error {

		ensureURLOrFileFlag(c)

		req := createInstallRequest(c)

		fmt.Fprint(os.Stderr, "Installing...")
		resp := common.SendRequest(req)

		var result InstallResult
		common.ParseResponse(resp, &result)
		if fail := result.ApplicationInstalledJson.Failure; fail != "" {
			fmt.Fprintf(os.Stderr, "Error occurred: %s", fail)
		} else {
			fmt.Fprintf(os.Stderr, "Installed '%s' v.%s", result.ApplicationInstalledJson.Application.DisplayName, result.ApplicationInstalledJson.Application.Version)
		}

		return nil
	},
}

func ensureURLOrFileFlag(c *cli.Context) {
	urlString := strings.TrimSpace(c.String("u"))
	fileString := strings.TrimSpace(c.String("f"))

	if (urlString == "") == (fileString == "") {
		var val string
		val = util.PromptUntilTrue(val, func(val string, ind byte) string {
			if val == "" && ind == 0 {
				return "Must provide either URL [u] or file [f] option: "
			} else if upper := strings.ToUpper(val); upper != "U" && upper != "F" {
				return "Please type [u] for URL or [f] for file:  "
			} else {
				return ""
			}
		})
		switch val {
		case "U", "u":
			ensureURLFlag(c)
		case "F", "f":
			ensureFileFlag(c)
		}
	}
}

func ensureURLFlag(c *cli.Context) {
	val := c.String("u")
	val = util.PromptUntilTrue(val, func(val string, ind byte) string {
		if len(strings.TrimSpace(val)) == 0 {
			switch ind {
			case 0:
				return "Enter URL: "
			default:
				return "URL can not be empty: "
			}
		} else {
			_, err := url.Parse(val)
			if err != nil {
				return fmt.Sprintf("URL '%s' is not valid: ", val)
			}
			return ""
		}
	})
	c.Set("u", val)
}

func ensureFileFlag(c *cli.Context) {
	val := c.String("f")
	val = util.PromptUntilTrue(val, func(val string, ind byte) string {
		if len(strings.TrimSpace(val)) == 0 {
			switch ind {
			case 0:
				return "Enter path to file: "
			default:
				return "Path to file can not be empty: "
			}
		} else {
			if _, err := os.Stat(val); os.IsNotExist(err) {
				return fmt.Sprintf("File '%s' does not exist: ", val)
			}
			return ""
		}
	})
	c.Set("f", val)
}

func createInstallRequest(c *cli.Context) *http.Request {
	body := new(bytes.Buffer)
	var baseUrl, contentType string

	if filePath := c.String("f"); filePath != "" {
		baseUrl = "api/app/install"
		file, _ := os.Open(filePath)
		defer file.Close()

		writer := multipart.NewWriter(body)
		part, _ := writer.CreateFormFile("file", filepath.Base(file.Name()))
		io.Copy(part, file)
		contentType = writer.FormDataContentType()
		writer.Close()
	} else if urlParam := c.String("u"); urlParam != "" {
		baseUrl = "api/app/installUrl"
		contentType = "application/json"
		params := map[string]string{
			"URL": urlParam,
		}
		json.NewEncoder(body).Encode(params)
	}

	req := common.CreateRequest(c, "POST", baseUrl, body)
	req.Header.Set("Content-Type", contentType)
	return req
}

type InstallResult struct {
	ApplicationInstalledJson struct {
		Application struct {
			DisplayName      string    `json:displayName`
			Key              string    `json:key`
			Deletable        bool      `json:deletable`
			Editable         bool      `json:editable`
			Local            bool      `json:local`
			MaxSystemVersion string    `json:maxSystemVersion`
			MinSystemVersion string    `json:minSystemVersion`
			ModifiedTime     time.Time `json:modifiedTime`
			State            string    `json:state`
			Url              string    `json:url`
			VendorName       string    `json:vendorName`
			VendorUrl        string    `json:verndorUrl`
			Version          string    `json:version`
		}
		Failure string `json:failure`
	} `json:applicationInstalledJson`
}
