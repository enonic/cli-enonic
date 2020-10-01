package client

import (
	"os"
	"strings"
)

const CLI_CLOUD_API_URL = "ENONIC_CLI_CLOUD_API_URL"

var (
	graphQLURL   = apiURL("")
	appUploadURL = apiURL("app")
)

func apiURL(path string) string {
	url := os.Getenv(CLI_CLOUD_API_URL)
	if strings.HasSuffix(url, "/") {
		return url + path
	}
	return url + "/" + path
}
