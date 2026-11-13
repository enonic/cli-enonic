# Environments and device login (beta)

Status: ready for approval. Branch `device-login`, PR #711.

## Intent

Enonic CLI should let a user sign in to any XP instance with their own account through a browser (OAuth device flow), keep that sign-in per instance, and switch between named instances, as described in
[enonic/requirements#7](https://github.com/enonic/requirements/issues/7) (requirements 1, 5 and 10).

PR #711 already has a minimal device login: a single `tokens.json` keyed by management URL, with the web endpoint guessed as "management host, port 8080" and the `system` ID provider. That guess is wrong for Enonic Cloud. On a cloud instance the web host differs from the management host, and the ID provider is `enonic-cloud`. XP now advertises its authorization server (resource discovery), so the CLI can find it.

The feature will be released in a CLI version that ships **before** the XP version that supports it, so the whole feature sits behind a beta switch.

## Goals

- Named environments: add, list, switch, remove, and show the active one.
- One stored login per environment, each in its own file. Switching environments never logs the user out of another one.
- Choose the target instance per command, per shell session, or persistently, following the conventions of gcloud, kubectl and docker.
- `enonic login` finds the authorization server through discovery and works against both cloud and local XP without extra flags.
- Everything is gated by `ENONIC_CLI_BETA=true`.

## Non-goals (this round)

- Cloud instances listed automatically in `env list` after `enonic cloud login` (requirement 10, last bullet).
- QR code login, choosing scopes, revoking access.
- Non-interactive/CI use: service account keys (`--cred-file`) already cover it.
- Re-enabling the old commented-out `remote` command group. It is superseded by `env`.

## Anti-goals (a failure even if it works)

- Without `ENONIC_CLI_BETA=true`, any change to what the CLI does. That includes reading an environment or token file that a beta session saved earlier.
- A command silently going to a different instance than the user selected. For example, an unknown `--env` name falling back to localhost.
- Secrets (tokens) stored in the same file as shareable configuration.
- A login prompting again while its refresh token is still valid (this was fixed earlier in #711 and must stay fixed).

## Beta switch

- The only switch is the environment variable `ENONIC_CLI_BETA=true`. There is no `--beta` flag.
- When it is not set:
  - `env`, `login` and `logout` are hidden from help. Running them prints `This is a beta feature. Set ENONIC_CLI_BETA=true to use it.` and exits with a non-zero code.
  - The `--env` option is hidden. If it is given anyway, the CLI prints the same message.
  - `ENONIC_CLI_ENV` is ignored.
  - Environment files and token files are never read, and no Bearer token is sent.
  - Target resolution, `-a`, `--cred-file`, `ENONIC_CLI_REMOTE_URL/USER/PASS`, the proxy and mTLS work exactly as on master.
- When it is set, everything below applies.

## Choosing the target instance

The first match wins:

1. `--env <name>` on the command.
2. `ENONIC_CLI_ENV=<name>` in the shell.
3. `ENONIC_CLI_REMOTE_URL`: a one-off URL with no name, as today.
4. The active environment saved by `env use`.
5. The built-in `local` environment (`http://localhost:4848`).

- Shell-level settings beat the saved choice. gcloud (`--configuration` > `CLOUDSDK_ACTIVE_CONFIG_NAME` > active config file) and docker (`--context` > `DOCKER_CONTEXT` > current context) work the same way.
- If a name from step 1, 2 or 4 does not exist, the command fails with `Unknown environment '<name>'. Run 'enonic env list'.` It never falls through to a lower step.
- `ENONIC_CLI_HTTP_PROXY` keeps overriding the proxy, whichever target is selected.

### `local` environment

- It always exists, at `http://localhost:4848`. It cannot be removed or redefined.
- It is what `env list` shows on a fresh install.
- Plain `enonic login` targets it (requirement 5).

## `env` commands

| Command | Behavior |
|---|---|
| `enonic env add <name> <url> [--proxy <url>]` | Adds an environment. Fails if the name exists or is `local`. Names are limited to `[a-z0-9][a-z0-9_-]*` because they are used in file names. The URL is the management endpoint. |
| `enonic env list` | Lists environments with their URLs. Marks the active one, and whether each one has a stored login. |
| `enonic env use <name>` | Saves the active environment. |
| `enonic env current` | Prints the environment that resolution picks right now, and why (flag, variable, saved, default). |
| `enonic env remove <name>` | Removes the environment and deletes its token file. If it was the active one, `local` becomes active. |

Output follows the CLI's existing style (stderr for messages, plain stdout for lists).

## Files

Everything goes under the existing enonic directory (`common.GetInEnonicDir`, which respects `ENONIC_CLI_HOME_PATH` and snap):

- `environments.toml` holds the environment definitions (name, URL, optional proxy) and the active name. It has no secrets and can be read and shared. TOML matches the CLI's existing data files.
- `tokens/<env>.json` holds one login per environment. The file has mode 0600 and the directory 0700. Writes are atomic: a temp file in the same directory, then a rename.
- `tokens/url-<host>-<port>.json` holds a login made while `ENONIC_CLI_REMOTE_URL` selected a one-off URL. The host is sanitized for file names.
- A token file contains:
  - the access token, its expiry and the refresh token
  - the management URL it belongs to
  - the discovered `token_endpoint`, issuer and client id, so a refresh needs no discovery
- The CLI uses a token only if the management URL stored in the file matches the target. This guards against an environment whose URL was changed by editing the TOML.
- The `tokens.json` from #711 was never released. It is no longer read or migrated.
- The existing concurrency handling stays: re-read before writing a refreshed token, and only replace it if the refresh token is unchanged. Each environment now has its own file, so this only matters for parallel commands against the same environment.

## Login discovery

`enonic login` (on the selected target):

1. Optional `--auth-server <issuer-url>` skips steps 2 and 3.
2. Sends an unauthenticated `GET` to the management URL and expects `401` with `WWW-Authenticate: Bearer resource_metadata="<url>"` (RFC 9728). Several `WWW-Authenticate` headers may be present, for example `Basic` alongside `Bearer`.
3. Fetches the protected resource metadata and takes the first entry of `authorization_servers`.
4. Fetches the authorization server metadata (RFC 8414) at `<origin>/.well-known/oauth-authorization-server<issuer path>` and reads `device_authorization_endpoint` and `token_endpoint`.
5. Runs the device flow with `client_id=enonic-cli` and `resource=<management URL>` (RFC 8707), and keeps the existing output (code, URL, browser open, spinner, "Logged in to … as …").
6. Saves the token file for the target.

- If step 2 finds no `resource_metadata`, login fails with `<url> does not advertise a login server. Device login needs XP 8.2 or later.`
- The old `host:8080` / `system` guess and the `--url` / `--id-provider` flags of `login` are removed.
- Discovery requests go through the same proxy and mTLS transport as other requests (`newHttpTransport`).

Verified on 2026-11-13 against `https://sry-kk-test-mgmt.enonic-dev.cloud`:

- The 401 carries `resource_metadata`.
- The metadata points to `https://sry-kk-test.enonic-dev.cloud/_/idprovider/enonic-cloud`.
- RFC 8414 metadata is served at `/.well-known/oauth-authorization-server/_/idprovider/enonic-cloud`.
- `device/code` accepts `enonic-cli` both with and without `resource`.

## Using the login

These parts are unchanged from #711. Only the token's location changes, to the selected target's token file:

- Order of precedence: `--cred-file`, then `-a`, then the stored login, then `ENONIC_CLI_REMOTE_USER/PASS` or the password prompt (no session with beta).
- Refresh when the token is near expiry.
- One forced refresh and retry on a 401.
- Hints on 401/403.

`enonic logout` deletes the selected target's token file.

## Which commands follow the selected target

- All commands that talk to the management endpoint follow the selected target and accept `--env`. That includes `project install` and `project deploy`. This is the same as `kubectl apply` with the current context, and as today with `ENONIC_CLI_REMOTE_URL`. Decided by the user.
- `sandbox` commands and commands that go to fixed URLs (market, `system latest`) don't take `--env`.
- `sandbox start` passes only `ENONIC_CLI_HTTP_PROXY` to the local XP, as on master. It never passes a remote environment's proxy. (My call.)

## Legacy credentials with environments

- Session authentication (`JSESSIONID` in the runtime data) is not used at all with beta: no session is sent, saved or cleared. Password authentication sends the credentials with every request. A session saved without beta is left as it is. (Decided by the user.)
- `ENONIC_CLI_REMOTE_USER` and `ENONIC_CLI_REMOTE_PASS` apply to whichever target is selected, as today. The user sets them explicitly. (My call.)
- Environments store no mTLS settings. `--client-cert` and `--client-key` stay per command. (My call.)

## Constraints

- Go, urfave/cli v1, and the existing helpers (`newHttpTransport`, `StartSpinner`, `util.FormatImportant`).
- Many call sites use `remote.GetActiveRemote()` with no `cli.Context`. `--env` arrives through the context, so the selected environment has to reach those call sites.
- `--env` must be accepted on every remote command. Most of them share `common.AUTH_AND_TLS_FLAGS`, which is used in 25 files.
- The current tests in `access-token_test.go` must keep passing in their adapted form.

## Left to the builder

- How `--env` reaches `GetActiveRemote` (resolve once in an app `Before` hook and store it, pass the context through, or similar).
- Whether `--env` is also a global flag (`enonic --env x app list`) in addition to the per-command flag.
- The exact table layout of `env list`.
- Package layout (a new `env` package, or extending `remote`).
- Test design. At minimum it should cover:
  - the resolution order
  - beta off leaving behavior unchanged
  - unknown names
  - discovery parsing, including multiple `WWW-Authenticate` headers
  - file permissions
- Updating `docs/` once the beta is documented. Optional this round.
