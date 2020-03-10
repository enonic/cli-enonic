package client

var (
	api          = "<TO_BE_DECIDED>"
	graphQLURL   = apiURL("/graphql")
	appUploadURL = apiURL("/app-upload")
)

func apiURL(path string) string {
	return api + path
}
