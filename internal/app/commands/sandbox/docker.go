package sandbox

import (
	"cli-enonic/internal/app/commands/common"
	"cli-enonic/internal/app/util"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"
)

const DOCKER_DISTRO_PREFIX = "docker:"
const DOCKER_IMAGE_ENONIC_XP = "enonic/xp"
const DOCKER_HUB_TAGS_URL = "https://hub.docker.com/v2/repositories/enonic/xp/tags/?page_size=100&ordering=last_updated"
const DOCKER_CONTAINER_PREFIX = "enonic-sandbox-"
const DOCKER_XP_HOME = "/enonic-xp/home"

// IsDockerDistro checks if a distro name represents a docker image
func IsDockerDistro(distro string) bool {
	return strings.HasPrefix(distro, DOCKER_DISTRO_PREFIX)
}

// GetDockerImageName extracts the docker image name from a distro name
func GetDockerImageName(distro string) string {
	return strings.TrimPrefix(distro, DOCKER_DISTRO_PREFIX)
}

// FormatDockerDistro formats a docker image name as a distro name
func FormatDockerDistro(imageName string) string {
	return DOCKER_DISTRO_PREFIX + imageName
}

// GetDockerContainerName returns the docker container name for a sandbox
func GetDockerContainerName(sandboxName string) string {
	return DOCKER_CONTAINER_PREFIX + strings.ToLower(sandboxName)
}

// IsDockerAvailable checks if docker is installed and available
func IsDockerAvailable() bool {
	return util.IsCommandAvailable("docker")
}

// EnsureDockerAvailable ensures docker is installed, exits if not
func EnsureDockerAvailable() {
	if !IsDockerAvailable() {
		fmt.Fprintln(os.Stderr, "Docker is not installed or not available in PATH. Please install Docker first.")
		os.Exit(1)
	}
}

// IsDockerImagePulled checks if a docker image is already pulled locally
func IsDockerImagePulled(imageName string) bool {
	cmd := exec.Command("docker", "image", "inspect", imageName)
	return cmd.Run() == nil
}

// PullDockerImage pulls a docker image from the registry
func PullDockerImage(imageName string) error {
	fmt.Fprintf(os.Stderr, "Pulling docker image '%s'...\n", imageName)
	cmd := exec.Command("docker", "pull", imageName)
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// EnsureDockerImageExists ensures a docker image exists locally, pulling it if needed
func EnsureDockerImageExists(imageName string) {
	EnsureDockerAvailable()
	if !IsDockerImagePulled(imageName) {
		if err := PullDockerImage(imageName); err != nil {
			fmt.Fprintf(os.Stderr, "Could not pull docker image '%s': %v\n", imageName, err)
			os.Exit(1)
		}
	}
}

// IsDockerContainerRunning checks if a docker container with the given name is currently running
func IsDockerContainerRunning(containerName string) bool {
	cmd := exec.Command("docker", "ps", "--filter", fmt.Sprintf("name=^/%s$", containerName), "--format", "{{.Names}}")
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	for _, line := range strings.Split(strings.TrimSpace(string(output)), "\n") {
		if line == containerName {
			return true
		}
	}
	return false
}

// startDockerSandbox starts a sandbox using docker and returns the exec.Cmd
func startDockerSandbox(imageName, sandboxName string, detach, devMode, debug bool, httpPort uint16) *exec.Cmd {
	homePath := GetSandboxHomePath(sandboxName)
	containerName := GetDockerContainerName(sandboxName)

	// Ensure home directory exists
	if _, err := os.Stat(homePath); os.IsNotExist(err) {
		if mkErr := os.MkdirAll(homePath, 0755); mkErr != nil {
			fmt.Fprintf(os.Stderr, "Could not create sandbox home directory: %v\n", mkErr)
			os.Exit(1)
		}
	}

	args := []string{"run", "--rm", "--name", containerName}

	if detach {
		args = append(args, "-d")
	}

	// Port mappings
	args = append(args,
		"-p", fmt.Sprintf("%d:8080", httpPort),
		"-p", fmt.Sprintf("%d:%d", common.MGMT_PORT, common.MGMT_PORT),
		"-p", fmt.Sprintf("%d:%d", common.INFO_PORT, common.INFO_PORT),
	)

	// Mount home directory
	args = append(args, "-v", fmt.Sprintf("%s:%s", homePath, DOCKER_XP_HOME))

	// Image name
	args = append(args, imageName)

	// Pass mode arguments to the container entrypoint
	if debug {
		// debug should go as 1st param
		args = append(args, "debug")
	}
	if devMode {
		args = append(args, "dev")
	}

	cmd := exec.Command("docker", args...)
	if !detach {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin
	}

	if err := cmd.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "Could not start docker container: %v\n", err)
		os.Exit(1)
	}

	return cmd
}

// stopDockerContainer stops a running docker container by name
func stopDockerContainer(containerName string) {
	cmd := exec.Command("docker", "stop", containerName)
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Could not stop docker container '%s': %v\n", containerName, err)
	}
}

type dockerHubTagsResponse struct {
	Count    int            `json:"count"`
	Next     string         `json:"next"`
	Previous string         `json:"previous"`
	Results  []dockerHubTag `json:"results"`
}

type dockerHubTag struct {
	Name string `json:"name"`
}

// FetchDockerTags fetches available tags for the enonic/xp image from Docker Hub
func FetchDockerTags() ([]string, error) {
	resp, err := http.Get(DOCKER_HUB_TAGS_URL)
	if err != nil {
		return nil, fmt.Errorf("could not fetch docker tags: %v", err)
	}
	defer resp.Body.Close()

	var tagsResponse dockerHubTagsResponse
	if err := json.NewDecoder(resp.Body).Decode(&tagsResponse); err != nil {
		return nil, fmt.Errorf("could not parse docker tags response: %v", err)
	}

	var tags []string
	for _, tag := range tagsResponse.Results {
		if tag.Name != "" {
			tags = append(tags, DOCKER_IMAGE_ENONIC_XP+":"+tag.Name)
		}
	}
	return tags, nil
}

// promptDockerImage prompts the user to select or enter a docker image
func promptDockerImage(imageStr string, force bool) string {
	if imageStr != "" {
		return imageStr
	}

	if force {
		fmt.Fprintln(os.Stderr, "Docker image must be specified with --image flag in non-interactive mode.")
		os.Exit(1)
	}

	fmt.Fprint(os.Stderr, "Loading available docker images from Docker Hub...")
	tags, err := FetchDockerTags()
	if err != nil || len(tags) == 0 {
		fmt.Fprint(os.Stderr, "Failed\n")
	} else {
		fmt.Fprint(os.Stderr, "Done\n")
	}

	var options []string
	if len(tags) > 0 {
		options = append(options, tags...)
	}
	options = append(options, "Custom image")

	_, idx, selectErr := util.PromptSelect(&util.SelectOptions{
		Message:           "Select Enonic XP docker image",
		Options:           options,
		Default:           options[0],
		PageSize:          10,
		StartInSearchMode: len(tags) > 0,
	})
	util.Fatal(selectErr, "Could not select docker image: ")

	if options[idx] == "Custom image" {
		imageStr = util.PromptString("Enter docker image name", "", DOCKER_IMAGE_ENONIC_XP+":latest-sdk", func(val interface{}) error {
			str := val.(string)
			if len(strings.TrimSpace(str)) == 0 {
				return fmt.Errorf("docker image name can not be empty")
			}
			return nil
		})
	} else {
		imageStr = options[idx]
	}

	return imageStr
}
