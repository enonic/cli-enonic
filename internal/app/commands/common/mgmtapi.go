package common

import (
	"cli-enonic/internal/app/commands/remote"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"github.com/urfave/cli"
)

// SERVER_API_PROBE is answered by XP 8.2 and later, which serve the management APIs as /server:<api>; older XP
// answers it with 404
const SERVER_API_PROBE = "/server:repo/"

var serverApis = struct {
	sync.Mutex
	byEndpoint map[string]bool
}{byEndpoint: map[string]bool{}}

// ServerApis reports whether commands call XP's management APIs (/server:<api>, XP 8.2 and later) rather than
// the legacy REST resources, which a management endpoint can switch off. --compat decides when it is given;
// otherwise the endpoint is asked once per run
func ServerApis(c *cli.Context) bool {
	if c != nil {
		if compat := c.String("compat"); compat != "" {
			return compatAtLeast(compat, 8, 2)
		}
	}

	endpoint := remote.GetActiveRemote().Url.String()
	serverApis.Lock()
	defer serverApis.Unlock()
	if known, ok := serverApis.byEndpoint[endpoint]; ok {
		return known
	}
	res := SendRequest(c, CreateRequest(c, "GET", SERVER_API_PROBE, nil), "")
	res.Body.Close()
	served := res.StatusCode != http.StatusNotFound
	serverApis.byEndpoint[endpoint] = served
	return served
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

// ServerApiPath is the path of a management API, e.g. ServerApiPath("dump", name, "load") is /server:dump/<name>/load
func ServerApiPath(api string, segments ...string) string {
	return "/server:" + api + "/" + strings.Join(segments, "/")
}
