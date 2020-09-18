package system

import (
	"fmt"
	"github.com/enonic/cli-enonic/internal/app/commands/common"
	"github.com/urfave/cli"
	"strings"
	"time"
)

var Changelog = cli.Command{
	Name:    "changelog",
	Aliases: []string{"log"},
	Usage:   "Release notes history",
	Flags:   common.FLAGS,
	Action: func(c *cli.Context) error {

		req := common.CreateRequest(c, "GET", "http://api.github.com/repos/enonic/cli-enonic/releases", nil)
		req.Header.Add("Accept", "application/vnd.github.v3+json")
		res := common.SendRequest(req, "Loading")

		var result ReleasesResponse
		common.ParseResponse(res, &result)

		for i := 0; i < len(result); i++ {
			curr := result[i]
			if !(curr.Draft || curr.Prerelease) {
				fmt.Println("\r\n\r\n***************************")
				fmt.Printf("*   Enonic CLI v%s\r\n", strings.TrimSpace(curr.Tag))
				fmt.Println("***************************")
				fmt.Println("\r\n", strings.TrimSpace(curr.Body))
			}
		}

		return nil
	},
}

type AuthorInfo struct {
	Login string `json:login`
}

type ReleaseInfo struct {
	Id          uint32     `json:id`
	Url         string     `json:url`
	Tag         string     `json:"tag_name"`
	Name        string     `json:name`
	Body        string     `json:body`
	Draft       bool       `json:draft`
	Prerelease  bool       `json:prerelease`
	PublishedAt time.Time  `json:"published_at"`
	Author      AuthorInfo `json:author`
}

type ReleasesResponse []ReleaseInfo
