![Baton Logo](./docs/images/baton-logo.png)

# `baton-hubspot` [![Go Reference](https://pkg.go.dev/badge/github.com/conductorone/baton-hubspot.svg)](https://pkg.go.dev/github.com/conductorone/baton-hubspot) ![main ci](https://github.com/conductorone/baton-hubspot/actions/workflows/main.yaml/badge.svg)

`baton-hubspot` is a connector for HubSpot built using the [Baton SDK](https://github.com/conductorone/baton-sdk). It communicates with the HubSpot User provisioning API to sync data about which teams and users have access within an account.

Check out [Baton](https://github.com/conductorone/baton) to learn more about the project in general.

# Prerequisites

To obtain an API key, you need to create an account in HubSpot and create a private application, under which you can create an API key. (More information [here](https://developers.hubspot.com/docs/api/intro-to-auth)) This means that you can connect multiple API keys to one account in HubSpot, but you can only connect one account to one API key.

Be aware that to sync also the user or team roles, you have to have an enterprise account since these roles are available only under enterprise account.

# Getting Started

## brew

```
brew install conductorone/baton/baton conductorone/baton/baton-hubspot

BATON_TOKEN=hubspotAccessToken baton-hubspot
baton resources
```

## docker

```
docker run --rm -v $(pwd):/out -e BATON_TOKEN=hubspotAccessToken ghcr.io/conductorone/baton-hubspot:latest -f "/out/sync.c1z"
docker run --rm -v $(pwd):/out ghcr.io/conductorone/baton:latest -f "/out/sync.c1z" resources
```

## source

```
go install github.com/conductorone/baton/cmd/baton@main
go install github.com/conductorone/baton-hubspot/cmd/baton-hubspot@main

BATON_TOKEN=hubspotAccessToken baton-hubspot
baton resources
```

# Data Model

`baton-hubspot` will pull down information about the following HubSpot resources:

- Users
- Teams
- Account

By default, `baton-hubspot` will sync information only from account based on provided credential.

# Contributing, Support and Issues

We started Baton because we were tired of taking screenshots and manually building spreadsheets. We welcome contributions, and ideas, no matter how small -- our goal is to make identity and permissions sprawl less painful for everyone. If you have questions, problems, or ideas: Please open a Github Issue!

See [CONTRIBUTING.md](https://github.com/ConductorOne/baton/blob/main/CONTRIBUTING.md) for more details.

# `baton-hubspot` Command Line Usage

```
baton-hubspot

Usage:
  baton-hubspot [flags]
  baton-hubspot [command]

Available Commands:
  completion         Generate the autocompletion script for the specified shell
  help               Help about any command

Flags:
      --client-id string       The client ID used to authenticate with ConductorOne ($BATON_CLIENT_ID)
      --client-secret string   The client secret used to authenticate with ConductorOne ($BATON_CLIENT_SECRET)
  -f, --file string            The path to the c1z file to sync with ($BATON_FILE) (default "sync.c1z")
  -h, --help                   help for baton-hubspot
      --log-format string      The output format for logs: json, console ($BATON_LOG_FORMAT) (default "json")
      --log-level string       The log level: debug, info, warn, error ($BATON_LOG_LEVEL) (default "info")
      --token string           The HubSpot personal access token used to connect to the HubSpot API. ($BATON_TOKEN)
      --user-status bool       Enables user status syncing. (false by default). Additional token scope required. ($BATON_USER_STATUS)
  -v, --version                version for baton-hubspot

Use "baton-hubspot [command] --help" for more information about a command.
```
