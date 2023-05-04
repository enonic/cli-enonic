package remote

import (
	"cli-enonic/internal/app/util"
	"fmt"
	"github.com/pkg/errors"
	"github.com/urfave/cli"
	"golang.org/x/crypto/bcrypt"
	"os"
	"strings"
)

var Add = cli.Command{
	Name: "add",
	Flags: []cli.Flag{
		cli.StringFlag{
			Name:  "url, u",
			Usage: "Remote url in the following format: [scheme]://[user:password]@[host]:[port]",
		},
		cli.StringFlag{
			Name:  "proxy, p",
			Usage: "Proxy url in the following format: [scheme]://[user:password]@[host]:[port]",
		},
	},
	Usage: "Add a new remote to list. Format: [name] -u [url] -p [proxy]",
	Action: func(c *cli.Context) error {

		name := ensureUniqueNameArg(c)
		remoteUrl := ensureUrl(c.String("url"), "Remote url")

		userName := remoteUrl.User.Username()
		userPass, passSet := remoteUrl.User.Password()
		if passSet {
			userPass = generateHash(userPass)
		}

		var proxyUrl *MarshalledUrl
		if proxyText := c.String("proxy"); proxyText != "" {
			proxyUrl = ensureUrl(proxyText, "Proxy url")
		}

		data := readRemotesData()
		data.Remotes[name] = RemoteData{remoteUrl, userName, userPass, proxyUrl}
		writeRemotesData(data)

		fmt.Fprintf(os.Stdout, "Remote '%s' created.", name)
		return nil
	},
}

func generateHash(s string) string {
	saltedBytes := []byte(s)

	hashedBytes, err := bcrypt.GenerateFromPassword(saltedBytes, bcrypt.DefaultCost)
	util.Fatal(err, fmt.Sprintf("Could not generate hash from '%s'", s))

	return string(hashedBytes[:])
}

func ensureUrl(urlString, promptString string) *MarshalledUrl {
	var (
		parsedUrl *MarshalledUrl
		err       error
	)

	var urlValidator = func(val interface{}) error {
		str := val.(string)
		if len(strings.TrimSpace(str)) == 0 {
			return errors.New("URL can not be empty: ")
		} else {
			if parsedUrl, err = ParseMarshalledUrl(str); err != nil {
				return errors.New("Incorrect URL. Format: [scheme]://[user:password]@[host]:[port]: ")
			}
		}
		return nil
	}

	util.PromptString(promptString, urlString, "", urlValidator)

	return parsedUrl
}

func ensureUniqueNameArg(c *cli.Context) string {
	var name string
	if c.NArg() > 0 {
		name = c.Args().First()
	}
	remotes := readRemotesData()

	validator := func(val interface{}) error {
		str := val.(string)
		if len(strings.TrimSpace(str)) == 0 {
			return errors.New("Remote name can not be empty: ")
		} else {
			if _, exists := getRemoteByName(str, remotes.Remotes); exists {
				return errors.Errorf("Remote '%s' already exists: ", str)
			}
		}
		return nil
	}

	return util.PromptString("Enter the name of the remote", name, "", validator)
}
