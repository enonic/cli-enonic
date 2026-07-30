package namespace

import (
	"cli-enonic/internal/app/commands/common"
	"cli-enonic/internal/app/util"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/pkg/errors"
	"github.com/urfave/cli"
)

func All() []cli.Command {
	return []cli.Command{
		List,
		Get,
		Create,
		Update,
		Delete,
	}
}

func namespaceUrl(key string) string {
	return "/server:schema/namespaces/" + url.PathEscape(key)
}

func ensureKeyArg(c *cli.Context) string {
	var key string
	if c.NArg() > 0 {
		key = c.Args().First()
	}

	keyValidator := func(val interface{}) error {
		str := val.(string)
		if len(strings.TrimSpace(str)) == 0 {
			if common.IsForceMode(c) {
				fmt.Fprintln(os.Stderr, "Namespace key can not be empty in non-interactive mode.")
				os.Exit(1)
			}
			return errors.New("Namespace key can not be empty: ")
		}
		return nil
	}

	return util.PromptString("Enter namespace key", key, "", keyValidator)
}

// parseResponse accepts any 2xx status as success, unlike common.ParseResponse
// which requires 200. A body-less success (e.g. 204 No Content) leaves target unset.
func parseResponse(resp *http.Response, target interface{}) {
	defer resp.Body.Close()
	decoder := json.NewDecoder(resp.Body)
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		if err := decoder.Decode(target); err != nil && err != io.EOF {
			fmt.Fprint(os.Stderr, "Error parsing response ", err)
			os.Exit(1)
		}
	} else {
		var enonicError common.EnonicError
		if err := decoder.Decode(&enonicError); err == nil && enonicError.Message != "" {
			fmt.Fprintf(os.Stderr, "Failure: %s\n", enonicError.Message)
		} else {
			fmt.Fprintln(os.Stderr, resp.Status)
		}
		os.Exit(1)
	}
}
