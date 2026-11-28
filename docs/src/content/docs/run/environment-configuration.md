---
title: Environment configuration
description: How to configure <span class="-tracking-[0.075em]">wanderer</span> with environment variables
---

Global settings for <span class="-tracking-[0.075em]">wanderer</span> can be adjusted via environment variables. If you deployed <span class="-tracking-[0.075em]">wanderer</span> with docker you can set them in the `.env` file next to your `docker-compose.yml` (see [`.env.example`](https://github.com/open-wanderer/wanderer/blob/main/.env.example)). If you deployed <span class="-tracking-[0.075em]">wanderer</span> on bare-metal you can change the environment variables in the launch script.

## Common
These variables are shared between all three services.

| Environment Variable | Description                                                      | Default                                     |
| -------------------- | ---------------------------------------------------------------- | ------------------------------------------- |
| MEILI_URL            | IP or hostname (including the port) of your meilisearch instance | http://search:7700                          |
| MEILI_MASTER_KEY     | **Required.** Master API key for your meilisearch instance. Generate with `openssl rand -hex 32` |                                             |

## Meilisearch
Since we use an unmodified installation of meilisearch you can use all variables listed in meilisearch's documentation. You can find a full list over [here](https://www.meilisearch.com/docs/learn/configuration/instance_options).

| Environment Variable | Description                   | Default |
| -------------------- | ----------------------------- | ------- |
| MEILI_NO_ANALYTICS   | Disable meilisearch telemetry | true    |

## Pocketbase
| Environment Variable           | Description                                                                                                       | Default               |
| ------------------------------ | ----------------------------------------------------------------------------------------------------------------- | --------------------- |
| ORIGIN                         | Public IP or hostname (including the port) of your <span class="-tracking-[0.075em]">wanderer</span> frontend (must be the same as in the frontend config) | <http://localhost:3000> |
| POCKETBASE_ENCRYPTION_KEY      | **Required.** Valid 32 character AES key. Used to encrypt secrets. Generate with `openssl rand -hex 16`. Never change it after the first start, or stored secrets can no longer be decrypted |                       |
| POCKETBASE_PROXY_SECRET        | **Required.** Shared secret authenticating the frontend → backend hop for inbound ActivityPub delivery and geocoding. Set the **same** value on `db` and `web`; if missing or mismatched, these internal requests fail closed. Generate with `openssl rand -hex 32` |                       |
| POCKETBASE_CRON_SYNC_SCHEDULE  | Valid cron expression. Sets how often installed plugins are synced                                                | 0 2 ** *             |
| POCKETBASE_CRON_SEARCH_REPAIR_SCHEDULE | Valid cron expression. Sets how often the search indexes are compared with the database and repaired. The job can also be run from the Crons page of the PocketBase dashboard | `0 4 * * *` |
| POCKETBASE_SMTP_ENABLED        | Enables or disables SMTP functionality. Accepted values are true or false                                         | false                 |
| POCKETBASE_SMTP_SENDER_ADDRESS | The email address used as the "From" address in outgoing emails                                                   |                       |
| POCKETBASE_SMTP_SENDER_NAME    | The display name shown as the sender in outgoing emails                                                           |                       |
| POCKETBASE_SMTP_HOST           | The hostname or IP address of the SMTP server                                                                     |                       |
| POCKETBASE_SMTP_PORT           | The port number used to connect to the SMTP server                                                                |                       |
| POCKETBASE_SMTP_USERNAME       | The username used to authenticate with the SMTP server                                                            |                       |
| POCKETBASE_SMTP_PASSWORD       | The password used to authenticate with the SMTP server                                                            |                       |
| OIDC_SCOPES                    | Comma-separated OIDC scopes for the `oidc` provider. Only needed for providers that reject the defaults, such as OpenStreetMap (`openid`). See [Auth providers](/run/backend-configuration/auth-providers) | openid,profile,email |
| OIDC2_SCOPES                   | Same as `OIDC_SCOPES`, for the `oidc2` provider                                                                   | openid,profile,email |
| OIDC3_SCOPES                   | Same as `OIDC_SCOPES`, for the `oidc3` provider                                                                   | openid,profile,email |
| OIDC_TRUST_EMAIL               | Set to `true` to trust the email and display name of the `oidc` provider when it sends an email without an `email_verified` claim and no username, as Cloudflare Access does. See [Auth providers](/run/backend-configuration/auth-providers#providers-that-do-not-send-email_verified) | false                |
| OIDC2_TRUST_EMAIL              | Same as `OIDC_TRUST_EMAIL`, for the `oidc2` provider                                                              | false                |
| OIDC3_TRUST_EMAIL              | Same as `OIDC_TRUST_EMAIL`, for the `oidc3` provider                                                              | false                |
| REGION_ARCHIVE_CRON_SCHEDULE   | Valid cron expression (UTC). Sets when offline [region archives](/run/backend-configuration/region-catalogue) are rebuilt | 0 3 * * *             |
| REGION_ARCHIVE_EXTRACT_TIMEOUT | Maximum duration a single region archive extraction may take, as a Go duration string (e.g. `45m`)                | 30m                   |
| REGION_ARCHIVE_REBUILD_DAYS    | Age in days at which a region's vector archive is rebuilt from a newer Protomaps build                            | 7                     |

Plugins are not configured through an environment variable. See
[Plugin installation](/run/installation/plugins) for installing runtime plugin
bundles and configuring self-hosted connector trust settings.

## Frontend

| Environment Variable    | Description                                                                      | Default                             |
| ----------------------- | -------------------------------------------------------------------------------- | ----------------------------------- |
| ORIGIN                  | Public IP or hostname (including the port) of your <span class="-tracking-[0.075em]">wanderer</span> instance             | http://localhost:3000               |
| BODY_SIZE_LIMIT         | Adapter ceiling and multipart upload limit; K, M and G suffixes use powers of 1024, `Infinity` permits uploads of any size after sign-in | 2G                                  |
| PUBLIC_POCKETBASE_URL   | IP or hostname (including the port) of your pocketbase instance                  | http://db:8090                      |
| POCKETBASE_PROXY_SECRET | **Required.** Shared secret authenticating the frontend → backend hop for inbound ActivityPub delivery and geocoding. Must match the value set on the `db` service. |                       |
| PUBLIC_DISABLE_SIGNUP   | Disables signup option for new users                                             | false                               |
| PUBLIC_PRIVATE_INSTANCE | Setting this to true will block visitors from viewing content without an account | false                               |
| PUBLIC_MAP_MAX_POLYLINES | Maximum number of polylines (route previews) to show simultaneously on the map, based on result density | 100 |
| UPLOAD_FOLDER           | Folder from which <span class="-tracking-[0.075em]">wanderer</span> auto-uploads trails                                   | /app/uploads                        |
| UPLOAD_USER             | Username for the account with which <span class="-tracking-[0.075em]">wanderer</span> auto-uploads trails                 |                                     |
| UPLOAD_PASSWORD         | Password for the account with which <span class="-tracking-[0.075em]">wanderer</span> auto-uploads trails                 |                                     |

### Visitor IP addresses

These settings are provided by [SvelteKit adapter-node](https://svelte.dev/docs/kit/adapter-node#ADDRESS_HEADER-and-XFF_DEPTH).

For a direct deployment, leave `ADDRESS_HEADER` unset. SvelteKit's Node adapter
uses the connection's socket address, which the web service forwards to
PocketBase. Behind a reverse proxy, configure these variables on the `web`
service only when the proxy is controlled and direct access to the web service
is blocked. The proxy must overwrite public forwarding headers with an
authoritative address or a controlled proxy chain.

| Environment Variable | Description | Default |
| -------------------- | ----------- | ------- |
| ADDRESS_HEADER | Trusted header used by adapter-node to resolve the client address, for example `X-Forwarded-For` | Unset (socket address) |
| XFF_DEPTH | When `ADDRESS_HEADER=X-Forwarded-For`, number of trusted proxy hops, counted from the right of the header | 1 |

For one controlled reverse proxy that overwrites `X-Forwarded-For` with the
visitor's address, set `ADDRESS_HEADER=X-Forwarded-For` and `XFF_DEPTH=1`.
For multiple proxies, match `XFF_DEPTH` to the trusted chain; do not trust an
arbitrary leftmost address supplied by the visitor. Misconfiguration lets
visitors forge the address used for rate limiting.

Configure PocketBase's [User IP proxy headers](/run/backend-configuration#visitor-ip-addresses)
and restrict access to the backend as described there. `POCKETBASE_PROXY_SECRET`
protects federation delivery and geocoding requests and does not authenticate
these IP headers.

## Map tiles & assets

These variables configure where the mobile app loads map vector tiles, fonts (glyphs), and sprites from. They are set on the `web` service. See [Custom map tiles & assets](/run/backend-configuration/map-tiles) for details.

| Environment Variable | Description                                                                                          | Default                                        |
| -------------------- | ---------------------------------------------------------------------------------------------------- | ---------------------------------------------- |
| TILE_SERVER_URL      | Vector tile URL template (`{z}/{x}/{y}`) served to the mobile app                                    | Protomaps API (requires PROTOMAPS_API_KEY)     |
| PROTOMAPS_API_KEY    | API key for the default Protomaps tile service; only used when TILE_SERVER_URL is unset              |                                                |
| MAP_ASSETS_URL       | Base URL for map fonts (glyphs) and sprites                                                          | https://protomaps.github.io/basemaps-assets    |

## Geocoding & Routing

These variables configure server-side requests to Valhalla, Nominatim and Overpass.
Nominatim is called by the `db` service.

| Environment Variable     | Description                                                                 | Default                             |
| ------------------------ | --------------------------------------------------------------------------- | ----------------------------------- |
| VALHALLA_URL     | Valhalla API URL used for auto-routing and elevation data       | https://valhalla1.openstreetmap.de  |
| NOMINATIM_URL    | Nominatim API URL used for (reverse) geocoding                  | https://nominatim.openstreetmap.org |
| OVERPASS_API_URL | Overpass API URL used for map points of interest                | https://overpass-api.de             |

When the canonical setting is unset, the service also accepts the
corresponding legacy `PUBLIC_*_URL` setting. `PUBLIC_NOMINATIM_URL` is a
server-side compatibility fallback; prefer `NOMINATIM_URL`. A
Nominatim URL without a scheme uses HTTPS. Use an explicit `http://` for a
private endpoint that does not use TLS, and use punycode for internationalized
hostnames. An explicitly empty or invalid setting on `db` disables geocoding
rather than falling back to the public service.

:::caution[Before upgrading geocoding]
Upgrade `web` and `db` together. Set `POCKETBASE_PROXY_SECRET` to the **same**
value on both services. If you don't already have one, generate it once with
`openssl rand -hex 32` and store it securely.

If you customized Nominatim only on `web` (for example in the v0.21.2 Compose
file), move the setting to `NOMINATIM_URL` on **`db` before updating**.
The `web` service still checks the old setting for this upgrade: a custom,
blank or invalid value on `web` blocks search and reverse geocoding if `db`
would use public Nominatim. This applies even when `db` explicitly selects
the public service. The request returns a generic HTTP 502 without sending
search terms or coordinates upstream. The check never sends the old URL to
`db` and does not prevent a valid private setting on `db` from working.

If you intend to use public Nominatim, remove the obsolete custom setting
from `web`. With the original public default (or both old settings unset)
on `web`, leaving both URL settings unset on `db` uses public Nominatim.
:::

With the new sample Compose file, an existing `NOMINATIM_URL` in the shared
`.env` already reaches `db` and needs no change. With per-service
environments, set the private URL (including one previously set as
`PUBLIC_NOMINATIM_URL`) on `db`, for example:

```yaml
services:
  db:
    environment:
      NOMINATIM_URL: http://nominatim:8080
```

Make sure the configured endpoint is reachable **from `db`**. If `db` uses
an outbound proxy, add the private Nominatim hostname to `NO_PROXY` on `db`
when it should be reached directly.

Geocoding keeps upstream HTTP 4xx and 5xx statuses (including an upstream
401) but returns a generic error instead of forwarding provider details.
A missing or mismatched `POCKETBASE_PROXY_SECRET`, blocked redirects and
unexpected success bodies return a generic 502. Configure each hop with
its final URL; the Nominatim base URL must serve `/search` and `/reverse`
without redirects.
Reverse lookups without a matching address keep Nominatim's normal 200
no-result response.

Check the PocketBase logs for a blocked migration (`private_nominatim_required`,
warned once), invalid Nominatim configuration, or a blocked upstream redirect.
Check the `web` logs for internal proxy-secret errors and redirects from `db`.
If neither URL variable is set on `db`, its startup log notes the public
default. To diagnose upstream timeouts, oversized responses
or transport/read failures, temporarily set **Logs → Logs settings → Min log
level** to `-4` (DEBUG) in the PocketBase dashboard with log retention enabled.
Geocoding diagnostics include only a fixed failure category, not request or
endpoint details. Restore the previous level when finished.

## Custom CA certificates

If your API endpoints use certificates signed by a private CA, add the CA bundle and set `NODE_EXTRA_CA_CERTS` for the `web` service.
For a private Nominatim endpoint, `db` must trust its CA instead. Mount a CA
bundle that includes your private CA in `db` and point `SSL_CERT_FILE` to it;
`NODE_EXTRA_CA_CERTS` does not affect `db`.

| Environment Variable | Description                                                              | Default |
| -------------------- | ------------------------------------------------------------------------ | ------- |
| NODE_EXTRA_CA_CERTS  | Path to an additional CA bundle used by Node.js TLS connections          |         |

Example (`docker-compose.yml`):

```yaml
services:
  web:
    environment:
      NODE_EXTRA_CA_CERTS: /etc/ssl/private-ca/ca.pem
    volumes:
      - ./certs/ca.pem:/etc/ssl/private-ca/ca.pem:ro
```

Provider plugin connector CAs are configured per connector when a plugin
supports custom TLS. They are not read from `NODE_EXTRA_CA_CERTS`.

`BODY_SIZE_LIMIT` controls the adapter ceiling and multipart upload limit together. The sample Compose default is `2G`, enough for 99 default-sized photos and a GPX file in one request. When started through `server.js`, as in the provided image and `npm start`, the web server uses `2G` when the variable is unset; an invalid value logs a warning and also uses `2G`. `Infinity` permits uploads of any size by signed-in users, including in older Compose files. Increase this one variable when raising collection file limits, allowing space for multipart headers. Upload routes authenticate before reading or parsing the body. Other route limits remain fixed: login 64 KiB, ActivityPub inbox 2 MiB and ordinary requests 1 MiB. Configure reverse-proxy request limits accordingly.
