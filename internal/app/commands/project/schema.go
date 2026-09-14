package project

import (
	"archive/zip"
	"cli-enonic/internal/app/commands/app"
	"cli-enonic/internal/app/commands/common"
	"cli-enonic/internal/app/commands/remote"
	"cli-enonic/internal/app/commands/sandbox"
	"cli-enonic/internal/app/util"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/urfave/cli"
)

const XP_READY_TIMEOUT = 3 * time.Minute

// schemaJarPath returns build/libs/<short name>.jar, where the short name is the last segment
// of the application name (e.g. "myapp" for "com.example.myapp"). Schema applications have no
// version, so the file name carries none either.
func schemaJarPath(prjPath, appName string) string {
	return filepath.Join(prjPath, "build", "libs", destFromName(appName)+".jar")
}

func cmsDescriptorRequiredError(prjPath string) error {
	absPath, err := filepath.Abs(prjPath)
	if err != nil {
		absPath = prjPath
	}
	return fmt.Errorf("%s/%s is required for a schema application but was not found in '%s'. "+
		"Applications that provide schemas must contain a %s/%s (or %s/%s) descriptor",
		common.CMS_DIR_NAME, common.CMS_DESCRIPTOR_FILES[0], absPath,
		common.CMS_DIR_NAME, common.CMS_DESCRIPTOR_FILES[0], common.CMS_DIR_NAME, common.CMS_DESCRIPTOR_FILES[1])
}

// loadSchemaApp reads the application descriptor and checks that the project can be packaged:
// the application name must be valid and the CMS descriptor must be present.
func loadSchemaApp(prjPath string) (*common.AppDescriptor, error) {
	descriptor, err := common.ReadAppDescriptor(prjPath)
	if err != nil {
		return nil, err
	}
	if descriptor == nil {
		absPath, absErr := filepath.Abs(prjPath)
		if absErr != nil {
			absPath = prjPath
		}
		return nil, fmt.Errorf("application descriptor %s not found in '%s'", common.APP_DESCRIPTOR_FILES[0], absPath)
	}
	if err := common.ValidateAppName(descriptor.Name); err != nil {
		return nil, fmt.Errorf("%w (see %s)", err, common.APP_DESCRIPTOR_FILES[0])
	}
	if common.FindCmsDescriptorFile(prjPath) == "" {
		return nil, cmsDescriptorRequiredError(prjPath)
	}
	return descriptor, nil
}

func addZipDir(zw *zip.Writer, name string, info fs.FileInfo) error {
	// directory entries let OSGi find resources by folder (Bundle.findEntries) and keep empty folders
	header := &zip.FileHeader{Name: name, Method: zip.Store, Modified: info.ModTime()}
	header.SetMode(info.Mode())
	_, err := zw.CreateHeader(header)
	return err
}

func addZipFile(zw *zip.Writer, name, path string, info fs.FileInfo) error {
	header := &zip.FileHeader{Name: name, Method: zip.Deflate, Modified: info.ModTime()}
	header.SetMode(0644)
	w, err := zw.CreateHeader(header)
	if err != nil {
		return err
	}
	src, err := os.Open(path)
	if err != nil {
		return err
	}
	defer src.Close()
	_, err = io.Copy(w, src)
	return err
}

// writeSchemaJar packs a schema application into a jar: the manifest as the first entry, followed by the
// application descriptor, the application icon (when present) and the cms folder, all at the jar root.
// Nothing else from the project folder is included.
func writeSchemaJar(prjPath, jarPath string, manifest Manifest) error {
	descriptorFile := common.FindAppDescriptorFile(prjPath)
	if descriptorFile == "" {
		return fmt.Errorf("application descriptor %s not found in '%s'", common.APP_DESCRIPTOR_FILES[0], prjPath)
	}
	for _, name := range common.APP_DESCRIPTOR_FILES[1:] {
		if other := filepath.Join(prjPath, name); other != descriptorFile {
			if stat, err := os.Stat(other); err == nil && stat.Mode().IsRegular() {
				fmt.Fprintf(os.Stderr, "Both %s and %s exist, packaging %s\n", filepath.Base(descriptorFile), name, filepath.Base(descriptorFile))
			}
		}
	}

	cmsDir := filepath.Join(prjPath, common.CMS_DIR_NAME)
	if stat, err := os.Stat(cmsDir); err != nil || !stat.IsDir() {
		return cmsDescriptorRequiredError(prjPath)
	}

	if err := os.MkdirAll(filepath.Dir(jarPath), 0755); err != nil {
		return err
	}
	file, err := os.Create(jarPath)
	if err != nil {
		return err
	}
	defer file.Close()

	zw := zip.NewWriter(file)

	// manifest must be the first entry of a jar
	mw, err := zw.CreateHeader(&zip.FileHeader{Name: MANIFEST_PATH, Method: zip.Deflate, Modified: time.Now()})
	if err != nil {
		return err
	}
	if err := manifest.Write(mw); err != nil {
		return err
	}

	info, err := os.Stat(descriptorFile)
	if err != nil {
		return err
	}
	if err := addZipFile(zw, filepath.Base(descriptorFile), descriptorFile, info); err != nil {
		return err
	}

	iconFile := filepath.Join(prjPath, common.APP_ICON_FILE)
	if info, err := os.Stat(iconFile); err == nil {
		if info.Mode().IsRegular() {
			if err := addZipFile(zw, common.APP_ICON_FILE, iconFile, info); err != nil {
				return err
			}
		} else {
			fmt.Fprintf(os.Stderr, "Skipping '%s': not a regular file\n", iconFile)
		}
	}

	hasCmsDescriptor := false
	err = filepath.WalkDir(cmsDir, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(prjPath, path)
		if err != nil {
			return err
		}
		// zip entries always use forward slashes
		name := filepath.ToSlash(rel)

		info, err := entry.Info()
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return addZipDir(zw, name+"/", info)
		}
		if !info.Mode().IsRegular() {
			fmt.Fprintf(os.Stderr, "Skipping '%s': not a regular file\n", path)
			return nil
		}
		if isCmsDescriptorEntry(name) {
			hasCmsDescriptor = true
		}
		return addZipFile(zw, name, path, info)
	})
	if err != nil {
		return err
	}
	if !hasCmsDescriptor {
		return cmsDescriptorRequiredError(prjPath)
	}

	if err := zw.Close(); err != nil {
		return err
	}
	return file.Close()
}

func isCmsDescriptorEntry(name string) bool {
	for _, file := range common.CMS_DESCRIPTOR_FILES {
		if name == common.CMS_DIR_NAME+"/"+file {
			return true
		}
	}
	return false
}

func buildSchemaJar(prjPath string, descriptor *common.AppDescriptor) string {
	manifest, err := buildSchemaManifest(descriptor)
	util.Fatal(err, "Could not create application manifest: ")

	jarPath := schemaJarPath(prjPath, descriptor.Name)
	fmt.Fprintf(os.Stderr, "Building schema application '%s'...\n", descriptor.Name)

	err = writeSchemaJar(prjPath, jarPath, manifest)
	util.Fatal(err, "Could not build application: ")

	fmt.Fprintf(os.Stderr, "Built %s\n", jarPath)
	return jarPath
}

// waitForXpReady polls the management API until the application install endpoint is available.
// GET on app/install answers 405 once the ApplicationResource is registered, 404 before that and
// connection errors before Jetty is up.
func waitForXpReady(c *cli.Context, sandboxName string, timeout time.Duration) {
	req := common.CreateRequest(c, "GET", "app/install", nil)

	common.StartSpinner(fmt.Sprintf("Waiting for sandbox '%s' to become ready", sandboxName))
	deadline := time.Now().Add(timeout)
	lastResponse := "no response"
	for {
		res, err := common.SendRequestRaw(c, req, 1)
		if err == nil {
			res.Body.Close()
			lastResponse = res.Status
			if res.StatusCode == http.StatusMethodNotAllowed {
				common.StopSpinner()
				fmt.Fprintln(os.Stderr, "Done")
				return
			}
		}
		if time.Now().After(deadline) {
			common.StopSpinner()
			fmt.Fprintf(os.Stderr, "\nTimed out waiting for sandbox '%s' to become ready (last response: %s)\n", sandboxName, lastResponse)
			os.Exit(1)
		}
		time.Sleep(time.Second)
	}
}

// ensureSchemaProjectData resolves project data for a schema application. Unlike gradle projects a sandbox
// is optional: it is used only when already linked to the project; otherwise the application is installed
// to the active remote (e.g. an XP instance not managed by CLI).
func ensureSchemaProjectData(c *cli.Context) *common.ProjectData {
	ensureValidProjectFolder(".")
	projectData := common.ReadProjectData(".")

	if sandbox.Exists(projectData.Sandbox) {
		// regular flow: validates the sandbox, XP version and distro
		projectData, _ = ensureProjectDataExists(c, ".", "", "A sandbox is required to install the project, "+
			"do you want to create one")
		return projectData
	}

	if projectData.Sandbox != "" {
		fmt.Fprintf(os.Stderr, "Sandbox '%s' linked to the project does not exist.\n", projectData.Sandbox)
	}
	return projectData
}

func mustLoadSchemaApp(prjPath string) *common.AppDescriptor {
	descriptor, err := loadSchemaApp(prjPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	return descriptor
}

// installSchema builds a schema application jar and installs it over HTTP: into the sandbox linked to the project
// (starting it in detached mode when needed) or, without a linked sandbox, to the active remote.
func installSchema(c *cli.Context) error {
	// validate the project before any sandbox prompt
	descriptor := mustLoadSchemaApp(".")

	projectData := ensureSchemaProjectData(c)
	if projectData == nil {
		return nil
	}

	jarPath := buildSchemaJar(".", descriptor)
	fmt.Fprintln(os.Stderr, "")

	sandboxExists := sandbox.Exists(projectData.Sandbox)
	if sandboxExists && !c.Bool("skip-start") {
		if !sandbox.AskToStartSandboxDetached(c, projectData.Sandbox) {
			fmt.Fprintf(os.Stderr, "Sandbox '%s' is not running. Start it and run 'enonic project install' again, "+
				"or install the application manually:\n  enonic app install --file %s\n", projectData.Sandbox, jarPath)
			return nil
		}
		waitForXpReady(c, projectData.Sandbox, XP_READY_TIMEOUT)
	}

	if sandboxExists {
		fmt.Fprintf(os.Stderr, "Installing to sandbox '%s'...\n", projectData.Sandbox)
	} else {
		fmt.Fprintf(os.Stderr, "No sandbox linked to the project, installing to the active remote (%s)...\n", remote.GetActiveRemote().Url.String())
		fmt.Fprintln(os.Stderr, "Run 'enonic project sandbox <name>' to install to a sandbox instead.")
	}
	app.InstallFromFile(c, jarPath)

	return nil
}
