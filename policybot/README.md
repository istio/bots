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

The bot consists of three major functional areas:

- A UI dashboard available at eng.istio.io

- A set of webhooks that perform various actions based on GitHub or ZenHub events

- A set of managers that are invoked via cron job to perform some maintenance activity

At runtime, the bot has the following external dependencies:

- Google Cloud Spanner. The bot uses spanner for primary storage.

- GitHub. The bot acts as a GitHub webhook to receive notifications of GitHub activity. It
also calls the GitHub API.

- ZenHub. The bot calls into the ZenHub API to get information about GitHub issues.

- SendGrid. The bot sends email using SendGrid.

## Managers, Handlers, Filters, and Topics

The bot's functionality is broken down into a few categories, which are reflected in the source code layout:

- Managers. Managers are responsible for performing specific maintenance activities such as synchronizing data
from GitHub into Spanner, or managing the lifecycle of open GitHub issues. Managers are invoked as
standalone cron jobs, distinct from the main PolicyBot server job.

- GitHub WebHook Handlers. These handlers are responsible for responding to a variety of GitHub events, such
as opening or closing issues, creating pull requests, etc.

- ZenHub WebHook Handlers. Similar to the GitHub handlers from above, but they respond to ZenHub events.

- Topics. Topics represent different portions of the UI dashboard. Each topic includes the logic necessary to render
a set of pages and respond to user interaction.

## Credentials

The main server and the different managers require credentials to operate. These crdentials can be specified as environment variables or
via command-line options. Command-line options take precedence over environment variables. The
available credential settings are:

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

### Keeping secrets

The bot needs a bunch of credentials to operate. As explained above, these credentials are supplied
to the bot via environment variables or command-line flags. If you're a Googler, you can get access
to the credentials necessary at `go/valentine`.

## REST API

The bot exposes a REST API at <https://eng.istio.io>:

- /githubwebhook - used to report events from GitHub. This is called by GitHub whenever anything interesting happens in
the Istio repos.

- /zenhubwebhook - used to report events from ZenHub. This is called by ZenHub whenever anything interesting happens to Istio issues
tracked by ZenHub.

- /api/* - topic-specific API available to query information that the bot generates.

## Configuration file

The bot's behavior is controlled entirely through its configuration file. The
format of the configuration file is described by the `Args` struct in
`pkg/config/args.go`.

The configuration file used by the bot is controlled by options given to the bot at startup via either
environment variable or command-line option:

- CONFIG_REPO / --config_repo. The bot can read its configuration directly from a GitHub repository. As
changes are made to the repository, the bot automatically refreshes its configuration. This option lets
you indicate the GitHub organization, repository, and branch where the configuration file can be found.
This is specified as a single string in the form of org/repo/branch.

- CONFIG_FILE / --config_file. Indicates the path to the bot's YAML configuration file. If the config
repo is specified as a startup option, then this file path is relative to the repo. Otherwise, it is
treated as a local file path within the bot's container.

## Deployment

Here's how the bot is currently deployed:

- The bot executes on GKE. You can do a full build and deploy
a new revision of the bot to GKE using `make deploy`. In order for this
to work, you'll need to have setup `gcloud` previously to authenticate
with GCP. Once a new revision is deployed to GKE, it immediately starts receiving traffic.

- The bot depends on having a configured Google Cloud Spanner database. The schema of the database
is described by `spanner.ddl`

- The bot's configuration is maintained in the file `policybot/policybot.yaml` in the <https://github.com/istio/test-infra> repo.
Changes pushed to this file are automatically picked up by the bot.

- There are a number of cron jobs to trigger the managers on a regular basis.
