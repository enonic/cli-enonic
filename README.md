# Enonic XP CLI

Enonic XP CLI is a command-line tool built for management of installations and projects of [Enonic XP](https://github.com/enonic/xp).

In order to build and develop the CLI, you need to have the Go environment available.

## Installing Go build and release environment:

##### Mac OS

1. `brew install goreleaser`
1. `brew install dep` - This is the Go dependency management tool.

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

1. Check out [XP CLI](https://github.com/enonic/xp-cli) from GitHub
1. Run `dep ensure` in the project folder.  -  This will download all dependencies for the project.
1. Run `goreleaser --rm-dist --snapshot` in the project folder to build a snapshot of latest code.  A binary installation, ready for use will be put in the dist folder.

## Publishing

`goreleaser` requires the current commit to be tagged in GitHub in order to be published, so if you want to publish the latest code, commit it and tag the commit.  If you want to publish an earlier version, check out the version.  Then a build (`goreleaser --rm-dist`) will publish `xp-cli` to GitHub and our own Artifactory repo, as long as it is set up correctly:
1. GitHub - To publish to GitHub, you must have publishing rights on the xp-cli project, and a personal Access Code to identify yourself.  This can be set up on GitHub by going to your personal Settings / Developer Settings / Personal Access token.  Create a token with all rights to repo and put it in `~/.config/goreleaser/github_token`
1. repo.enonic.com - This repo use the Artifactory general `ci` user.  The API key for the `ci` user must be put in a local environment variable called `ARTIFACTORY_REPO_SECRET`.

If you build a snapshot with `goreleaser --rm-dist --snapshot`, it may be uploaded to our repo by executing this command for each created distro:
* `curl -u ci:$ARTIFACTORY_REPO_SECRET -X PUT "http://repo.enonic.com/public/com/enonic/cli/eonic/next/enonic_0.1.9-next_Windows_64-bit.zip" -T enonic_0.1.9-next_Windows_64-bit.zip`
