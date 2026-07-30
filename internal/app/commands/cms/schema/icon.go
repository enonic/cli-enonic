package schema

import (
	"bytes"
	"cli-enonic/internal/app/commands/common"
	"cli-enonic/internal/app/util"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"github.com/urfave/cli"
)

var Icon = cli.Command{
	Name:  "icon",
	Usage: "Schema icon commands",
	Subcommands: []cli.Command{
		IconSet,
		IconDelete,
	},
}

var IconSet = cli.Command{
	Name:      "set",
	Usage:     "Set the icon of a deployed schema",
	ArgsUsage: "<file>",
	Flags: append([]cli.Flag{
		KEY_FLAG,
		KIND_FLAG,
		common.FORCE_FLAG,
	}, common.AUTH_AND_TLS_FLAGS...),
	Action: func(c *cli.Context) error {

		k, namespace, name := ensureIconTarget(c)
		file := ensureIconFileArg(c)

		if err := setIcon(c, k, namespace, name, file); err != nil {
			fmt.Fprintf(os.Stderr, "Failure: %s\n", err)
			os.Exit(1)
		}

		fmt.Fprintf(os.Stderr, "Icon of %s '%s' set\n", k.name, k.target(namespace, name))
		return nil
	},
}

var IconDelete = cli.Command{
	Name:  "delete",
	Usage: "Delete the icon of a schema",
	Flags: append([]cli.Flag{
		KEY_FLAG,
		KIND_FLAG,
		common.FORCE_FLAG,
	}, common.AUTH_AND_TLS_FLAGS...),
	Action: func(c *cli.Context) error {

		k, namespace, name := ensureIconTarget(c)
		target := k.target(namespace, name)

		req := common.CreateRequest(c, "DELETE", k.url(namespace, name)+"/icon", nil)
		res := common.SendRequest(c, req, fmt.Sprintf("Deleting icon of %s '%s'", k.name, target))

		if err := parseResponseErr(res, new(interface{})); err != nil {
			fmt.Fprintf(os.Stderr, "Failure: %s\n", err)
			os.Exit(1)
		}

		fmt.Fprintf(os.Stderr, "Icon of %s '%s' deleted\n", k.name, target)
		return nil
	},
}

// ensureIconTarget resolves the kind and key flags of the icon commands,
// exiting if the kind does not support icons
func ensureIconTarget(c *cli.Context) (*kind, string, string) {
	k := ensureKindFlag(c)
	if !k.icon {
		fmt.Fprintf(os.Stderr, "Icons are not supported for kind '%s': must be one of %s.\n", k.name, strings.Join(iconKindNames(), ", "))
		os.Exit(1)
	}
	namespace, name := ensureKeyFlag(c, k)
	return k, namespace, name
}

// iconKindNames returns the names of the kinds that support icons
func iconKindNames() []string {
	var names []string
	for _, k := range kinds {
		if k.icon {
			names = append(names, k.name)
		}
	}
	return names
}

// iconMimeType resolves the mime type of an icon file from its extension;
// the server accepts SVG and PNG icons only
func iconMimeType(path string) (string, error) {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".svg":
		return "image/svg+xml", nil
	case ".png":
		return "image/png", nil
	default:
		return "", errors.Errorf("icon '%s' must be a .svg or .png file", path)
	}
}

// checkIcon validates an icon upload without sending it: the kind supports
// icons, the extension maps to a supported mime type and the file exists.
// Used before deploying a descriptor so a bad icon fails the whole document
func checkIcon(k *kind, path string) error {
	if !k.icon {
		return errors.Errorf("icons are not supported for kind '%s': must be one of %s", k.name, strings.Join(iconKindNames(), ", "))
	}
	if _, err := iconMimeType(path); err != nil {
		return err
	}
	if info, err := os.Stat(path); err != nil || info.IsDir() {
		return errors.Errorf("icon file '%s' does not exist", path)
	}
	return nil
}

// setIcon uploads an icon for a deployed schema: the raw image bytes are sent
// with the mime type in the Content-Type header. The descriptor must already
// exist on the server
func setIcon(c *cli.Context, k *kind, namespace, name, iconPath string) error {
	if err := checkIcon(k, iconPath); err != nil {
		return err
	}
	mimeType, err := iconMimeType(iconPath)
	if err != nil {
		return err
	}
	data, err := os.ReadFile(iconPath)
	if err != nil {
		return errors.Errorf("could not read icon file '%s': %v", iconPath, err)
	}
	if len(data) == 0 {
		return errors.Errorf("icon file '%s' is empty", iconPath)
	}

	target := k.target(namespace, name)
	req := common.CreateRequest(c, "PUT", k.url(namespace, name)+"/icon", bytes.NewReader(data))
	// CreateRequest hardcodes application/json, which the server rejects for icons
	req.Header.Set("Content-Type", mimeType)
	res := common.SendRequest(c, req, fmt.Sprintf("Setting icon of %s '%s'", k.name, target))

	notFound := res.StatusCode == http.StatusNotFound
	if err := parseResponseErr(res, new(interface{})); err != nil {
		if notFound {
			return errors.Errorf("%s: deploy the schema first", err)
		}
		return err
	}
	return nil
}

// ensureIconFileArg returns the icon file argument, prompting if missing
func ensureIconFileArg(c *cli.Context) string {
	var file string
	if c.NArg() > 0 {
		file = c.Args().First()
	}

	fileValidator := func(val interface{}) error {
		str := strings.TrimSpace(val.(string))
		var message string
		if str == "" {
			message = "Icon file can not be empty"
		} else if _, err := iconMimeType(str); err != nil {
			message = fmt.Sprintf("Icon file '%s' must be a .svg or .png file", str)
		} else if info, err := os.Stat(str); err != nil || info.IsDir() {
			message = fmt.Sprintf("File '%s' does not exist", str)
		} else {
			return nil
		}
		if common.IsForceMode(c) {
			fmt.Fprintln(os.Stderr, message+" in non-interactive mode.")
			os.Exit(1)
		}
		return errors.New(message + ": ")
	}

	return strings.TrimSpace(util.PromptString("Enter path to an icon file (.svg or .png)", file, "", fileValidator))
}