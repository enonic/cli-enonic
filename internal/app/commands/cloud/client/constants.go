package client

var (
	api          = "<SET_DURING_BUILD>"
	graphQLURL   = apiURL("/")
	appUploadURL = apiURL("/app")
)

func apiURL(path string) string {
	return api + path
}
