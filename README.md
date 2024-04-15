# Enonic CLI

<p align="left">
  <img alt="" src="https://img.shields.io/npm/v/@enonic/cli?style=flat">
  <img alt="" src="https://img.shields.io/npm/l/@enonic/cli.svg?style=flat">
  <a aria-label="Join the Enonic community on Slack" href="https://slack.enonic.com/">
    <img alt="" src="https://img.shields.io/badge/Join%20Slack-f03e2f?logo=Slack&style=flat">
  </a>
  <a aria-label="Follow Enonic on Twitter" href="https://twitter.com/enonichq">
    <img alt="" src="https://img.shields.io/twitter/follow/enonichq?style=flat&color=blue">
  </a>
</p>

The official development and management CLI tool for Enonic XP.

## About

Enonic offers an API-first content platform featuring structured content, powerful search, tree structures, preview and visual page composing. The platform uses a flexible schema system to support content type and components, combined with rich JavaScript and GraphQL APIs. Use your favorite front-end framework or build directly on the platform using the Enonic JS framework.

## Installation

Install **Enonic CLI** with the following command:

```bash
npm install @enonic/cli -g
```

## Usage

* Get the list of available commands:

```bash
$ enonic
```

* Create and start a new _empty_ sandbox (local instance of Enonic XP) called `mysandbox` using the latest stable release of Enonic XP:

```bash
$ enonic sandbox create mysandbox --skip-template -f
```

* Create and start a new sandbox called `mysandbox` with the bare minimum of pre-installed applications using the latest stable release of Enonic XP:

```bash
$ enonic sandbox create mysandbox -t essentials -f
```

* Create a new project called `myproject` and link it to the `mysandbox` instance:

```bash
$ enonic create com.example.myproject -s mysandbox
```

### Available Commands

```
COMMANDS:
  app [command] [options]           Install, stop and start applications
  auditlog [command] [options]      Manage audit log repository
  cms [command] [options]           CMS commands
  create <project name> [options]   Create a new Enonic project
  dev                               Start current project in dev mode
  dump [command] [options]          Dump and load complete repositories
  export [options]                  Export data from a given repository
  import [options]                  Import data from a named export
  latest                            Check for latest version of Enonic CLI
  repo [command] [options]          Tune and manage repositories
  snapshot [command] [options]      Create and restore snapshots
  system [command] [options]        System commands
  upgrade                           Upgrade to the latest version
  uninstall                         Uninstall Enonic CLI
  vacuum [options]                  Removes old version history and segments from content storage
  help, h                           Shows a list of all commands or help for a specific command

CLOUD COMMANDS:
  cloud [command] [options]         Manage Enonic cloud
   
PROJECT COMMANDS:
  project [command] [options]       Manage Enonic projects
  sandbox [command] [options]       Manage Enonic instances

GLOBAL OPTIONS:
  --help, -h                        Show help
  --version, -v                     Print the version
```


## Docs & Guides

For complete guide to Enonic CLI please check out [this page](https://developer.enonic.com/docs/enonic-cli/).

Build your first Enonic project by following this hands-on [introduction](https://developer.enonic.com/start).

Need a quick advice? Ask us on [Slack](https://slack.enonic.com/) or [Discuss](https://discuss.enonic.com/).

Prefer Docker? Check out Enonic XP in the [Docker Hub](https://hub.docker.com/r/enonic/xp).

Test Enonic XP in a forever-free Cloud instance? [Sign up](https://www.enonic.com/sign-up) for a Cloud Free Plan.

## Upgrade

```bash
$ enonic upgrade
```

## Uninstall

```bash
$ enonic uninstall
```
