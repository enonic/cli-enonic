# Enonic XP CLI

Enonic XP CLI is a command-line tool built for management of installations and projects of [Enonic XP](https://github.com/enonic/xp).

In order to build and develop the CLI, you need to have the Go environment available.

## Installing Go build and release environment:

##### Mac OS

1. Install [Go](https://go.dev/dl/)
 
2. Install Goreleaser:

   `brew install goreleaser`

1. Install Go dependency management tool:

   `brew install dep`

1. Install Snapcraft

   `brew install snapcraft`

##### Windows

Recommended way is to use [scoop](https://scoop.sh/) command line installer

1. Install scoop if needed

   *Make sure Powershell 3 (or later) and .NET Framework 4.5 (or later) are installed. Then run:*

   `iex (new-object net.webclient).downloadstring('https://get.scoop.sh')`
1. Install go

   `scoop install go`
1. Install goreleaser

   `scoop bucket add goreleaser https://github.com/goreleaser/scoop-bucket.git`

   `scoop install goreleaser`
1.  Install Go dependency management tool

    `scoop install dep`

For other OSes, please see [Goreleaser](https://goreleaser.com).

## Building project

1. Check out [XP CLI](https://github.com/enonic/cli-enonic) from GitHub
1. Run `dep ensure` in the project folder.  -  This will download all dependencies for the project.
1. Run `goreleaser --clean --snapshot` in the project folder to build a snapshot of latest code. Binaries for all supported platforms will be built in corresponding folders of the `dist` folder.

## Releasing a new version

1. Ensure [release notes](docs%2Freleases.adoc) are updated (for a feature release)
1. Change value of `:xp_version:` variable in the [installation guide](docs%2Finstall.adoc) to `X.Y.Z` (version of the upcoming release)
1. Commit all the uncommitted changes. Make sure HEAD is not in dirty state (no uncommitted changes).
1. Run `git tag vX.Y.Z`
1. Run `git push origin vX.Y.Z` to trigger the release via Github actions.
