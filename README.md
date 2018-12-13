# Enonic XP CLI

In order to build and develop the XP CLI, you need to have the go environment available.

## Installing Go build and release environment on Mac:

1. `brew install goreleaser`
1. `brew install dep` - This is the Go dependency management tool.

## Building project

1. Check out [XP CLI](https://github.com/enonic/xp-cli) from GitHub
1. Run `dep ensure` in the project folder.  -  This will download all dependencies for the project.
1. Run `goreleaser --rm-dist --snapshot` in the project folder to build a snapshot of latest code.  A binary installation, ready for use will be put in the dist folder.

## Publishing

`goreleaser` requires the current commit to be tagged in GitHub in order to be published, so if you want to publish the latest code, commit it and tag the commit.  If you want to publish an earlier version, check out the version.  Then a build (`goreleaser --rm-dist`) will publish `xp-cli` to GitHub and our own Artifactory repo, as long as it is set up correctly:
1. GitHub - To publish to GitHub, you must have publishing rights on the xp-cli project, and a personal Access Code to identify yourself.  This can be set up on GitHub by going to your personal Settings / Developer Settings / Personal Access token.  Create a token with all rights to repo and put it in `~/.config/goreleaser/github_token`
1. repo.enonic.com - This repo use the Artifactory general `ci` user.  The API key for the `ci` user must be put in a local environment variable called `ARTIFACTORY_REPO_SECRET`.
