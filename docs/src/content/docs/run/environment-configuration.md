---
title: Environment configuration
description: How to configure <span class="-tracking-[0.075em]">wanderer</span> with environment variables
---

Global settings for <span class="-tracking-[0.075em]">wanderer</span> can be adjusted via environment variables. If you deployed <span class="-tracking-[0.075em]">wanderer</span> with docker you can change the environment variables directly in the `docker-compose.yaml`. If you deployed <span class="-tracking-[0.075em]">wanderer</span> on bare-metal you can change the environment variables in the launch script.

## Common
These variables are shared between all three services.

| Environment Variable | Description                                                      | Default                                     |
| -------------------- | ---------------------------------------------------------------- | ------------------------------------------- |
| MEILI_URL            | IP or hostname (including the port) of your meilisearch instance | http://search:7700                          |
| MEILI_MASTER_KEY     | Master API key for your meilisearch instance                     | vODkljPcfFANYNepCHyDyGjzAMPcdHnrb6X5KyXQPWo |

## Meilisearch
Since we use an unmodified installation of meilisearch you can use all variables listed in meilisearch's documentation. You can find a full list over [here](https://www.meilisearch.com/docs/learn/configuration/instance_options).

| Environment Variable | Description                   | Default |
| -------------------- | ----------------------------- | ------- |
| MEILI_NO_ANALYTICS   | Disable meilisearch telemetry | true    |

## Pocketbase
| Environment Variable          | Description                                                                                                       | Default               |
| ----------------------------- | ----------------------------------------------------------------------------------------------------------------- | --------------------- |
| ORIGIN                        | Public IP or hostname (including the port) of your <span class="-tracking-[0.075em]">wanderer</span> frontend (must be the same as in the frontend config) | http://localhost:3000 |
| POCKETBASE_ENCRYPTION_KEY     | Valid 32 character AES key. Used to encrypt secrets                                                               |                       |
| POCKETBASE_CRON_SYNC_SCHEDULE | Valid cron expression. Sets how often trails are synced from 3rd party integrations                               | 0 2 * * *             |
| POCKETBASE_SMTP_ENABLED       | Enables or disables SMTP functionality. Accepted values are true or false                                         | false                 |
| POCKETBASE_SMTP_SENDER_ADDRESS | The email address used as the "From" address in outgoing emails                                                   |                       |
| POCKETBASE_SMTP_SENDER_NAME   | The display name shown as the sender in outgoing emails                                                           |                       |
| POCKETBASE_SMTP_HOST          | The hostname or IP address of the SMTP server                                                                     |                       |
| POCKETBASE_SMTP_PORT          | The port number used to connect to the SMTP server                                                                |                       |
| POCKETBASE_SMTP_USERNAME      | The username used to authenticate with the SMTP server                                                            |                       |
| POCKETBASE_SMTP_PASSWORD      | The password used to authenticate with the SMTP server                                                            |                       |

## Frontend

| Environment Variable    | Description                                                                                                   | Default                             |
|-------------------------|---------------------------------------------------------------------------------------------------------------| ----------------------------------- |
| ORIGIN                  | Public IP or hostname (including the port) of your <span class="-tracking-[0.075em]">wanderer</span> instance | http://localhost:3000               |
| BODY_SIZE_LIMIT         | Maximum allowed upload size                                                                                   | Infinity                            |
| PUBLIC_POCKETBASE_URL   | IP or hostname (including the port) of your pocketbase instance                                               | http://db:8090                      |
| PUBLIC_DISABLE_SIGNUP   | Disables signup option for new users                                                                          | false                               |
| PUBLIC_VALHALLA_URL     | Public IP or hostname (including the port) of a valhalla instance                                             | https://valhalla1.openstreetmap.de  |
| PUBLIC_NOMINATIM_URL    | Public IP or hostname (including the port) of a nominatim instance                                            | https://nominatim.openstreetmap.org |
| PUBLIC_PRIVATE_INSTANCE | Setting this to true will block visitors from viewing content without an account                              | false                               |
| UPLOAD_FOLDER           | Folder from which <span class="-tracking-[0.075em]">wanderer</span> auto-uploads trails                       | /app/uploads                        |
| UPLOAD_USER             | Username for the account with which <span class="-tracking-[0.075em]">wanderer</span> auto-uploads trails     |                                     |
| UPLOAD_PASSWORD         | Password for the account with which <span class="-tracking-[0.075em]">wanderer</span> auto-uploads trails     |                                     |
| PUBLIC_OVERPASS_API_URL | Overpass API URL used for map points of interest                                                              | https://overpass-api.de        |
| PUBLIC_TITLE            | The title of your instance. Leave empty for "wanderer"                                                         |         |
