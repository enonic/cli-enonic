package common

import (
	"cli-enonic/internal/app/util"
	"net/url"
	"strconv"
	"strings"

	"github.com/urfave/cli"
)

// ServerApis reports whether commands call XP's management APIs (/server:<api>, XP 8.2 and later) rather than
// the legacy REST resources, which a management endpoint can switch off. Beta: without ENONIC_CLI_BETA=true
// the REST resources are always used, as before. With beta, --compat decides; without it the management APIs
// are used
func ServerApis(c *cli.Context) bool {
	if !util.IsBeta() {
		return false
	}
	if c != nil {
		util.Fatal(ValidateCompatFlag(c), "Invalid argument")
		if compat := c.String("compat"); compat != "" {
			return compatAtLeast(compat, 8, 2)
		}
	}
	return true
}

// WithBetaCompat adds --compat to the flags of a command that calls the management APIs, so XP 8.0 and 8.1
// can be targeted with --compat 8.1. Beta, as the management APIs are: without it the flags stay as before
func WithBetaCompat(flags []cli.Flag) []cli.Flag {
	if util.IsBeta() {
		return append(flags, COMPAT_FLAG)
	}
	return flags
}

func compatAtLeast(compat string, major, minor int) bool {
	parts := strings.SplitN(compat, ".", 2)
	x, err := strconv.Atoi(parts[0])
	if err != nil || x != major {
		return err == nil && x > major
	}
	y := 0
	if len(parts) == 2 {
		y, _ = strconv.Atoi(parts[1])
	}
	return y >= minor
}

// ServerApiPath is the path of a management API, e.g. ServerApiPath("dump", name, "load") is /server:dump/<name>/load.
// Each segment is escaped, so a name with '/', '#', '?' or '%' stays one segment
func ServerApiPath(api string, segments ...string) string {
	escaped := make([]string, len(segments))
	for i, segment := range segments {
		escaped[i] = url.PathEscape(segment)
	}
	return "/server:" + api + "/" + strings.Join(escaped, "/")
}
