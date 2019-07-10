# Policy Bot

The policy bot enforces a number of Istio-wide policies around how we manage
issues, pull requests, test flakes, and more.

- [Overview](#overivew)
- [Handlers, Filters, and Topics](#handlers-filters-and-topics)
- [Startup options](#startup-options)
- [Configuration file](#configuration-file)
- [Deployment](#deployment)
- [Credentials and secrets](#credentials-and-secrets)

## Overview

The bot consists of a single binary running in a container. Use `make container` to build both the binary and the container.

At runtime, The bot has the following external dependencies:

- Google Cloud Spanner. The bot uses spanner for primary storage.

- GitHub. The bot acts as a GitHub webhook to receive notifications of GitHub activity. It
also calls the GitHub API.

- ZenHub. The bot calls into the ZenHub API to get information about GitHub issues.

- SendGrid. The bot sends email using SendGrid.

## Handlers, Filters, and Topics

The bot consists of a simple framework and distinct handlers that have specific isolated responsibilities.

A handler is responsible for dealing with requests arriving at a specific path. A specialized form of handler is called
a topic, which represents a top-level area in the dashboard UI. Topics are responsible for serving HTML and JSON traffic. 
The existing handlers include: 

- githubwebhook. Handles GitHub web hook events by dispatching to a set of filters (described below)
 
- zenhubwebhook. Handles ZenHub web hook events

- syncer. Initiates a synchronization of GitHub data to Google Cloud Spanner, where the data can then be used
for analysis. The syncer needs to be invoked on a periodic basis to refresh the data.

- flakechaser. Performs schedule analysis on test-flake related bugs and nags the PR to prompt for a resolution.

- topics. A number of handlers which each deliver the HTML and JSON to support the dashboard UI.

The githubwebhook handler supports a chain of filters which each get called for incoming
GitHub events. These includes:

- cfgmonitor. Monitors GitHub for changes to the bot's configuration file. When it sees such a change, it triggers a
partial shutdown and restart of the bot, which will reread the config and start back up fully.

- labeler. Attached labels to issues and pull requests if specific conditions are detected. This is primarily used
to perform initial triage on incoming issues by assigning an area-specific label to issues based on patterns
found in newly-opened issues.

- nagger. Injects nagging comments in pull requests if specific conditions are detected. This is primarily used to
remind developers to include tests whenever they fix bugs, but the engine is general-purpose and could be used
creatively for other nagging comments.

- refresher. Updates the local Google Cloud Spanner copy of GitHub data based on events
reported by the GitHub webhook.

## Startup options

The bot supports a number of startup options. These can be specified as environment variables or
via command-line options. Command-line options take precedence over environment variables. The
available startup options are:

- GITHUB_WEBHOOK_SECRET / --github_webhook_secret. Indicates the GitHub secret necessary to authenticate with
the GitHub webhook.

- GITHUB_TOKEN / --github_token. The access token necessary to let the bot invoke the GitHub
API.

- GITHUB_OAUTH_CLIENT_SECRET / --github_oauth_client_secret. The client secret to use in the GitHub OAuth flow,
as obtained in the GitHub admin UI for the target organization.

- GITHUB_OAUTH_CLIENT_ID / --github_oauth_client_id. The client ID to use in the GitHub OAuth flow,
as obtained in the GitHub admin UI for the target organization.

- ZENHUB_TOKEN / --zenhub_token. The access token necessary to let the bot invoke the ZenHub
API.

- GCP_CREDS / --gcp_creds. Base64-encoded JSON credentials for GCP, enabling the bot to invoke
Google Cloud Spanner.

- SENDGRID_APIKEY / --sendgrid_apikey. An API Key for the SendGrid service, enabling the bot to
send emails.

- CONFIG_REPO / --config_repo. The bot can read its configuration directly from a GitHub repository. As
changes are made to the repository, the bot automatically refreshes its configuration. This option lets
you indicate the GitHub organization, repository, and branch where the configuration file can be found.
This is specified as a single string in the form of org/repo/branch.

- CONFIG_FILE / --config_file. Indicates the path to the bot's YAML configuration file. If the config
repo is specified as a startup option, then this file path is relative to the repo. Otherwise, it is
treated as a local file path within the bot's container.

- PORT / --port. The TCP port to listen to for incoming traffic.

- HTTPS_ONLY / --https_only. Causes all HTTP traffic to be redirected to HTTPS instead.

## REST API

The bot exposes a REST API at https://eng.istio.io:

- /githubwebhook - used to report events from GitHub. This is called by GitHub whenever anything interesting happens in
the Istio repos.

- /zenhubwebhook - used to report events from ZenHub. This is called by ZenHub whenever anything interesting happens to Istio issues
tracked by ZenHub.

- /api/* - topic-specific API available to query information that the bot generates.

## Configuration file

The bot's behavior is controlled entirely through its configuration file. The
format of the configuration file is described by the `Args` struct in
`pkg/config/args.go`.

## Deployment

Here's how the bot is currently deployed:

- The bot executes on GKE. You can do a full build and deploy
a new revision of the bot to GKE using `make deploy`. In order for this
to work, you'll need to have setup `gcloud` previously to authenticate
with GCP. Once a new revision is deployed to GKE, it immediately starts receiving traffic.

- The bot depends on having a configured Google Cloud Spanner database. The schema of the database
is described by `spanner.ddl`

- TBD: The bot's configuration is maintained in the file `policybot/policybot.yaml` in the <https://github.com/istio/test-infra> repo.
Changes pushed to this file are automatically picked up by the bot.

## Credentials and secrets

The bot needs a bunch of credentials to operate. As explained above, these credentials are supplied
to the bot via environment variables or command-line flags. If you're a Googler, you can get access
to the credentials at <https://go/valentine>.
