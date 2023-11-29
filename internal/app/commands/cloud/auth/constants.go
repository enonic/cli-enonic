package auth

import (
	"os"
)

const CLI_CLOUD_AUTH_CLIENT_VAR = "ENONIC_CLI_CLOUD_AUTH_CLIENT"
const CLI_CLOUD_AUTH_CLIENT_DEFAULT = "WMq5N474JWbzIHY5RanLa5z4mConLz6C"

const CLI_CLOUD_AUTH_URL_VAR = "ENONIC_CLI_CLOUD_AUTH_URL"
const CLI_CLOUD_AUTH_URL_DEFAULT = "https://auth.enonic.com"

var (
	clientID = getAuthClient()
	scope    = "openid profile email offline_access"
	authURL  = getAuthUrl()
)

func getAuthUrl() string {
	if userUrl := os.Getenv(CLI_CLOUD_AUTH_URL_VAR); userUrl != "" {
		return userUrl
	}
	return CLI_CLOUD_AUTH_URL_DEFAULT
}

func getAuthClient() string {
	if userClient := os.Getenv(CLI_CLOUD_AUTH_CLIENT_VAR); userClient != "" {
		return userClient
	}
	return CLI_CLOUD_AUTH_CLIENT_DEFAULT
}
