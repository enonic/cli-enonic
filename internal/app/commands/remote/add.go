package remote

import (
	"github.com/urfave/cli"
	"github.com/enonic/enonic-cli/internal/app/util"
	"strings"
	"fmt"
	"golang.org/x/crypto/bcrypt"
	"os"
)

var Add = cli.Command{
	Name: "add",
	Flags: []cli.Flag{
		cli.StringFlag{
			Name:  "url, u",
			Usage: "Remote url in the following format: [scheme]://[user:password]@[host]:[port]",
		},
	},
	Usage: "Add a new remote to list. Format: [name] [scheme]://[user:password]@[host]:[port]",
	Action: func(c *cli.Context) error {

		name := ensureUniqueNameArg(c)
		remoteUrl := ensureUrlFlag(c)

		userName := remoteUrl.User.Username()
		userPass, passSet := remoteUrl.User.Password()
		if passSet {
			userPass = generateHash(userPass)
		}

		data := readRemotesData()
		data.Remotes[name] = RemoteData{remoteUrl, userName, userPass}
		writeRemotesData(data)

		fmt.Fprintf(os.Stderr, "Remote '%s' created.", name)
		return nil
	},
}

func generateHash(s string) string {
	saltedBytes := []byte(s)

	hashedBytes, err := bcrypt.GenerateFromPassword(saltedBytes, bcrypt.DefaultCost)
	util.Fatal(err, fmt.Sprintf("Could not generate hash from '%s'", s))

	return string(hashedBytes[:])
}

func ensureUrlFlag(c *cli.Context) *MarshalledUrl {
	remoteText := c.String("url")
	var (
		parsedUrl *MarshalledUrl
		err       error
	)
	util.PromptUntilTrue(remoteText, func(val *string, i byte) string {
		if len(strings.TrimSpace(*val)) == 0 {
			if i == 0 {
				return "Enter remote URL: "
			} else {
				return "Remote URL can not be empty: "
			}
		} else {
			if parsedUrl, err = ParseMarshalledUrl(*val); err != nil {
				return "Incorrect URL. Format: [scheme]://[user:password]@[host]:[port]: "
			}
			return ""
		}
	})
	return parsedUrl
}

func ensureUniqueNameArg(c *cli.Context) string {
	var name string
	if c.NArg() > 0 {
		name = c.Args().First()
	}
	remotes := readRemotesData()
	return util.PromptUntilTrue(name, func(val *string, i byte) string {
		if len(strings.TrimSpace(*val)) == 0 {
			if i == 0 {
				return "Enter the name of the remote: "
			} else {
				return "Remote name can not be empty: "
			}
		} else {
			if _, exists := getRemoteByName(*val, remotes.Remotes); exists {
				return fmt.Sprintf("Remote '%s' already exists: ", *val)
			}
			return ""
		}
	})
}
