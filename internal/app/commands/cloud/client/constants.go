package client

import (
	"os"
	"strings"
)

const CLI_CLOUD_API_URL_VAR = "ENONIC_CLI_CLOUD_API_URL"
const CLI_CLOUD_API_URL_DEFAULT = "https://cloud.enonic.com/api"

var (
	graphQLURL   = apiURL("")
	appUploadURL = apiURL("app")
)

func apiURL(path string) string {
	url := CLI_CLOUD_API_URL_DEFAULT
	if userUrl := os.Getenv(CLI_CLOUD_API_URL_VAR); userUrl != "" {
		url = userUrl
	}

	if strings.HasSuffix(url, "/") {
		return url + path
	}
	return url + "/" + path
}
