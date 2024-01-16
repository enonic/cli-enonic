package auth

import (
	"os"
)

const CLI_CLOUD_AUTH_CLIENT_VAR = "ENONIC_CLI_CLOUD_AUTH_CLIENT"
const CLI_CLOUD_AUTH_CLIENT_DEFAULT = "WMq5N474JWbzIHY5RanLa5z4mConLz6C"

const CLI_CLOUD_AUTH_SECRET_VAR = "ENONIC_CLI_CLOUD_AUTH_SECRET"
const CLI_CLOUD_AUTH_SECRET_DEFAULT = "llBw3NdElDoDNp4k--OtIzCggzYWmHKwjvYFIZD1Zq6Gya398_I-2CEQTSgBpnHm"

const CLI_CLOUD_AUTH_URL_VAR = "ENONIC_CLI_CLOUD_AUTH_URL"
const CLI_CLOUD_AUTH_URL_DEFAULT = "https://auth.enonic.com"

const CLI_CLOUD_AUTH_AUD_VAR = "ENONIC_CLI_CLOUD_AUTH_AUD"
const CLI_CLOUD_AUTH_AUD_DEFAULT = "https://cloud.enonic.com/api"

var (
	clientID     = getAuthClient()
	clientSecret = getAuthSecret()
	scope        = "openid profile email offline_access"
	audience     = getAuthAud()
	authURL      = getAuthUrl()
)

func getAuthUrl() string {
	if userUrl := os.Getenv(CLI_CLOUD_AUTH_URL_VAR); userUrl != "" {
		return userUrl
	}
	return CLI_CLOUD_AUTH_URL_DEFAULT
}

func getAuthClient() string {
	if clientID := os.Getenv(CLI_CLOUD_AUTH_CLIENT_VAR); clientID != "" {
		return clientID
	}
	return CLI_CLOUD_AUTH_CLIENT_DEFAULT
}

func getAuthSecret() string {
	if clientSecret := os.Getenv(CLI_CLOUD_AUTH_SECRET_VAR); clientSecret != "" {
		return clientSecret
	}
	return CLI_CLOUD_AUTH_SECRET_DEFAULT
}

func getAuthAud() string {
	if audience := os.Getenv(CLI_CLOUD_AUTH_AUD_VAR); audience != "" {
		return audience
	}
	return CLI_CLOUD_AUTH_AUD_DEFAULT
}
