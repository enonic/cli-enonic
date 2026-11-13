# Environments and Device Login (beta) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Named environments with one stored device login per environment, selectable per command, per shell or persistently, all behind `ENONIC_CLI_BETA=true`.

**Architecture:**
- Target resolution lives in the `remote` package. `GetActiveRemote()` keeps its signature, so the 10+ call sites don't change. It returns master behavior when beta is off and resolves environments when beta is on.
- Environment definitions are kept in `environments.toml`.
- Each login is stored in its own file under `tokens/`, managed by `common/access-token.go`.
- A new `env` command package provides `add`/`list`/`use`/`current`/`remove`.
- `--env` reaches `remote` through a `Before` hook that `commands.All()` installs on every command that declares the flag.

**Tech Stack:** Go 1.26, urfave/cli v1.22.17, BurntSushi toml (already used via `util.DecodeTomlFile`), `net/http/httptest` for tests.

**Spec:** `docs/superpowers/specs/2026-11-13-environments-and-device-login-design.md`

**Starting point:** branch `device-login` at 5fee062. OAuth discovery is already implemented in `internal/app/commands/common/discovery.go` (`DiscoverEndpoints`, `ErrNoDiscovery`). The token, refresh and retry-on-401 logic is in `internal/app/commands/common/access-token.go` and `common.go`.

## Global Constraints

- Beta switch: environment variable `ENONIC_CLI_BETA` with the exact value `true`. There is no `--beta` flag.
- Beta off:
  - `env`, `login` and `logout` are hidden. Running them prints `This is a beta feature. Set ENONIC_CLI_BETA=true to use it.` and exits with code 1.
  - `--env` is hidden and gives the same error.
  - `ENONIC_CLI_ENV` is ignored.
  - Environment and token files are never read, and no Bearer token is sent.
  - Everything else behaves byte-for-byte as on master.
- Resolution order with beta on:
  1. `--env`
  2. `ENONIC_CLI_ENV`
  3. `ENONIC_CLI_REMOTE_URL`
  4. the saved active environment
  5. `local`
- An unknown name prints `Unknown environment '<name>'. Run 'enonic env list'.` and exits with code 1. It never falls back to a later step.
- `local` = `http://localhost:4848`. It cannot be added, removed or redefined.
- Environment names must match `^[a-z0-9][a-z0-9_-]*$`.
- Files, all under `GetInEnonicDir()`:
  - `environments.toml`
  - `tokens/<env>.json`
  - `tokens/url-<host>-<port>.json` for a one-off `ENONIC_CLI_REMOTE_URL` login
  - Token files have mode 0600, the `tokens` directory 0700, and writes are atomic (temp file in the same directory, then rename).
- `tokens.json` from earlier in this PR is no longer read or migrated.
- `ENONIC_CLI_HTTP_PROXY` overrides the environment's proxy. `sandbox start` uses only `ENONIC_CLI_HTTP_PROXY`.
- Login:
  - discovery only, with `client_id=enonic-cli` and `resource=<management URL>`
  - optional `--auth-server <issuer-url>`
  - the `--url` and `--id-provider` options are removed
  - no discovery → `<url> does not advertise a login server. Device login needs XP 8.2 or later.`
- Messages go to stderr. Lists (`env list`, `env current`) go to stdout.
- Run all tests with `go test ./...` from the repo root.

## Review Focus

1. **Fresh install, beta on, no `environments.toml`.** Every command targets `local`, and merely reading must not create the file. → Task 2, `TestResolveDefaultsToLocalWithoutFile`.
2. **`environments.toml` is corrupt or hand-edited** (bad TOML, invalid URL, name outside the pattern). The command fails with a message naming the file. It must not panic, and it must not fall back to `local`. → Task 2, `TestReadEnvironmentsRejectsCorruptFile`.
3. **The environment's URL changes after login** (by editing the TOML, or remove and re-add under the same name with a new URL). The old token must not be sent to the new URL. → Task 3, `TestTokenNotSentToOtherUrl`.
4. **One-off `ENONIC_CLI_REMOTE_URL` written two ways** (`http://host`, `http://host:80/`). Both give the same token file. A host with odd characters can't escape `tokens/`. → Task 3, `TestOneOffTokenFileName`.
5. **Beta switched off after a beta session** that saved `env use prod` and a login. Commands go to master's target and send no Bearer token. → Task 2, `TestBetaOffIgnoresEnvironments`, and Task 3, `TestBetaOffSendsNoToken`.

---

### Task 1: Beta switch and shared enonic dir helper

**Files:**
- Modify: `internal/app/util/util.go` (add `IsBeta`, `GetInEnonicDir`, `SNAP_ENV_VAR`)
- Modify: `internal/app/commands/common/common.go:42,147-161` (`SNAP_ENV_VAR` and `GetInEnonicDir` delegate to util)
- Test: `internal/app/util/beta_test.go`

`remote` can't import `common` (`common` imports `remote`), so the enonic dir helper moves down into `util`.

**Interfaces:**
- Produces:
  - `util.BETA_ENV_VAR = "ENONIC_CLI_BETA"`
  - `util.IsBeta() bool`
  - `util.BETA_MESSAGE = "This is a beta feature. Set ENONIC_CLI_BETA=true to use it."`
  - `util.GetInEnonicDir(children ...string) string`
  - `common.GetInEnonicDir` and `common.SNAP_ENV_VAR` keep working as thin aliases.

- [ ] **Step 1: Write the failing test** `TestIsBeta`. It's table-driven over `ENONIC_CLI_BETA` values, set with `t.Setenv`: `"true"`→true; `""`, `"1"`, `"TRUE"`, `"yes"`→false (exact value only).
- [ ] **Step 2: Run** `go test ./internal/app/util/ -run TestIsBeta`. Expected: FAIL, `undefined: IsBeta`.
- [ ] **Step 3: Implement** `IsBeta` (reads the variable on every call, no caching) and move `GetInEnonicDir`'s body and `SNAP_ENV_VAR` into util. The common versions call through to util.
- [ ] **Step 4: Run** `go test ./...`. Expected: PASS.
- [ ] **Step 5: Commit** `git commit -m "Add beta switch and move enonic dir helper to util"`

---

### Task 2: Environments file and target resolution

**Files:**
- Create: `internal/app/commands/remote/environments.go`
- Modify: `internal/app/commands/remote/remote.go:67-117` (`RemoteData` fields, `GetActiveRemote`)
- Test: `internal/app/commands/remote/environments_test.go`

**Interfaces:**
- Consumes: `util.IsBeta()`, `util.GetInEnonicDir()` (Task 1)
- Produces:
  ```go
  const LOCAL_ENV = "local"
  const CLI_ENV = "ENONIC_CLI_ENV"
  type Environment struct { Url *MarshalledUrl `toml:"url"`; Proxy *MarshalledUrl `toml:"proxy,omitempty"` }
  type EnvironmentsData struct { Active string `toml:"active"`; Environments map[string]Environment `toml:"environments"` }
  func EnvironmentsFile() string                       // util.GetInEnonicDir("environments.toml")
  func ReadEnvironments() (*EnvironmentsData, error)    // missing file → empty data, nil; never creates it
  func WriteEnvironments(data *EnvironmentsData) error  // atomic write, dir 0700, file 0600
  func ValidEnvName(name string) bool                   // ^[a-z0-9][a-z0-9_-]*$
  func SelectEnv(name string)                           // sets the --env value; package var
  func ProxyFromEnv() *MarshalledUrl                    // parsed ENONIC_CLI_HTTP_PROXY or nil
  func ResolveTarget() (*RemoteData, error)             // beta resolution; error text per Global Constraints
  // RemoteData gains:
  Name   string // environment name, "" for master behavior or a one-off ENONIC_CLI_REMOTE_URL
  Source string // "--env", "ENONIC_CLI_ENV", "ENONIC_CLI_REMOTE_URL", "enonic env use", "default"
  ```
- `GetActiveRemote()`:
  - beta off: unchanged master code path, with `Name`/`Source` left empty
  - beta on: `ResolveTarget()`. On error it prints the error to stderr and calls `os.Exit(1)`.
  - It reads the file on every call (no cache), so tests can change state freely.
- The `local` target: `Name: "local"`, `Url: http://localhost:4848`, `Proxy: ProxyFromEnv()`.
- User/Pass always come from `ENONIC_CLI_REMOTE_USER/PASS`.
- `ENONIC_CLI_HTTP_PROXY` overrides `Environment.Proxy` when set.

Tests set `ENONIC_CLI_HOME_PATH` to `t.TempDir()`, and clear `ENONIC_CLI_ENV`, `ENONIC_CLI_REMOTE_URL` and `ENONIC_CLI_BETA` with `t.Setenv(..., "")`. `SelectEnv("")` runs in `t.Cleanup`.

- [ ] **Step 1: Write the failing tests:**
  - `TestResolveOrder`: a table over combinations of `SelectEnv`, `ENONIC_CLI_ENV`, `ENONIC_CLI_REMOTE_URL` and a saved `Active`. It asserts `Name`, `Url.String()` and `Source`, covering each step winning over all the lower ones. A one-off URL gives `Name == ""` and `Source == "ENONIC_CLI_REMOTE_URL"`.
  - `TestResolveUnknownEnvFails`: the unknown name comes from `--env`, then from `ENONIC_CLI_ENV`, then from a saved `Active`. Each case gives an error equal to `Unknown environment 'nope'. Run 'enonic env list'.`. Also, when `--env` is valid, an invalid `ENONIC_CLI_ENV` is ignored.
  - `TestResolveDefaultsToLocalWithoutFile`: gives `Name == "local"`, `Source == "default"`, and `EnvironmentsFile()` still doesn't exist afterwards.
  - `TestReadEnvironmentsRejectsCorruptFile`: three cases, each must give an error containing the file path:
    - the file contains `not = [toml`
    - an environment with `url = "::"`
    - an environment named `Bad/Name`
  - `TestWriteEnvironmentsRoundTrip`: writes two environments, one with a proxy, plus `Active`, and reads back equal data. The file mode is `0600`.
  - `TestEnvProxyOverride`: with `ENONIC_CLI_HTTP_PROXY` set, the resolved proxy is the variable's value, not the environment's.
  - `TestBetaOffIgnoresEnvironments`: saves `Active: "prod"` and sets `ENONIC_CLI_ENV=prod`, with beta off. `GetActiveRemote().Url.String()` is then `http://localhost:4848`, and is `ENONIC_CLI_REMOTE_URL` when that variable is set.
- [ ] **Step 2: Run** `go test ./internal/app/commands/remote/`. Expected: FAIL to compile.
- [ ] **Step 3: Implement** the interfaces above. Decode with `toml.Unmarshal` on `os.ReadFile` output. Don't use `util.DecodeTomlFile`, because it exits the process and creates the file.
- [ ] **Step 4: Run** `go test ./...`. Expected: PASS.
- [ ] **Step 5: Commit** `git commit -m "Resolve the target instance from named environments (beta)"`

---

### Task 3: One token file per target

**Files:**
- Modify: `internal/app/commands/common/access-token.go:250-370` (replace the `tokens.json` map with per-target files)
- Modify: `internal/app/commands/common/common.go` (call sites of `getAccessToken`/`forceRefreshAccessToken`, which take a `*remote.RemoteData` instead of a URL)
- Test: `internal/app/commands/common/access-token_test.go` (adapt the existing tests and add new ones)

**Interfaces:**
- Consumes: `remote.RemoteData{Name, Url}` and `remote.GetActiveRemote()` (Task 2), `util.IsBeta()`
- Produces:
  ```go
  func tokenFile(target *remote.RemoteData) (string, error) // tokens/<Name>.json, or tokens/url-<host>-<port>.json when Name == ""
  func SaveAccessToken(target *remote.RemoteData, token *AccessToken) error
  func RemoveAccessToken(target *remote.RemoteData) (bool, error)
  func HasAccessToken(target *remote.RemoteData) bool
  func getAccessToken(c *cli.Context, target *remote.RemoteData) string          // "" when beta off
  func forceRefreshAccessToken(c *cli.Context, target *remote.RemoteData) string
  func renewAccessToken(c *cli.Context, path string, token *AccessToken) string   // same re-read-before-write rule, on one file
  ```
- Rules for `tokenFile`:
  - Returns an error for a `Name` that fails `remote.ValidEnvName`.
  - Builds the one-off name from the lowercased hostname, with every character outside `[a-z0-9.-]` replaced by `_`, plus `effectivePort(url)`.
- `getAccessToken` uses a stored token only if `token.Resource == mgmtKey(target.Url)`. Otherwise it treats the target as not logged in.
- `SaveAccessToken` sets `token.Resource = mgmtKey(target.Url)` when the token has none.
- `tokensFile`/`readTokens`/`writeTokens` and the map format are deleted.

- [ ] **Step 1: Adapt the existing tests** `TestDeviceLoginAndRefresh` and `TestRefreshKeepsLoginReplacedMeanwhile` to the new signatures, with beta on. Then write the new failing tests:
  - `TestTokenFilePerEnvironment`: saves a token for `prod` and one for `stage`, then removes `prod`. Afterwards `stage` is still stored, the `tokens/prod.json` file is gone, `tokens/stage.json` has mode `0600`, and the `tokens` directory has mode `0700`.
  - `TestTokenNotSentToOtherUrl`: saves a token for `prod` at `https://a.example`. `getAccessToken` for `{Name:"prod", Url:https://b.example}` returns `""`.
  - `TestOneOffTokenFileName`:
    - `http://Host.Example` and `http://host.example:80/` give the same path, ending in `tokens/url-host.example-80.json`.
    - A host containing `..%2f` stays inside `tokens/`.
    - `tokenFile(&RemoteData{Name:"../x"})` returns an error.
  - `TestBetaOffSendsNoToken`: a token is saved, then beta is turned off. `CreateRequest(nil, "GET", "repo/list", nil)` with `-a`-free context has no `Bearer` Authorization header. Set `ENONIC_CLI_REMOTE_USER/PASS` so the request doesn't prompt.
- [ ] **Step 2: Run** `go test ./internal/app/commands/common/`. Expected: FAIL.
- [ ] **Step 3: Implement.** In `common.go`, change every call site from `&remote.GetActiveRemote().Url.URL` to `remote.GetActiveRemote()`.
- [ ] **Step 4: Run** `go test ./...`. Expected: PASS.
- [ ] **Step 5: Commit** `git commit -m "Keep each login in its own token file"`

---

### Task 4: Legacy session tied to its URL

**Files:**
- Modify: `internal/app/commands/common/common.go`:
  - `RuntimeData` (around line 129)
  - `CreateRequest`, `doCreateBasicAuthRequest`, `SendRequestCustom` (where `SessionID` is read and written)
- Test: `internal/app/commands/common/session_test.go`

**Interfaces:**
- Consumes: `remote.GetActiveRemote()`, `mgmtKey`, `util.IsBeta()`
- Produces:
  - `RuntimeData.SessionUrl string` with tag `toml:"sessionUrl"`
  - `func activeSessionId(rData RuntimeData) string`:
    - beta off: returns `rData.SessionID`, as on master
    - beta on: returns it only when `rData.SessionUrl == mgmtKey(&GetActiveRemote().Url.URL)`, else `""`
- Every place that reads `rData.SessionID` to decide or send uses `activeSessionId`.
- Storing the cookie also sets `SessionUrl`.

- [ ] **Step 1: Write the failing test** `TestSessionOnlySentToItsUrl`, with beta on:
  - The runtime data has `SessionID: "s1"` and `SessionUrl: "http://localhost:4848"`. With target `local`, the request carries the `JSESSIONID=s1` cookie.
  - With a target environment at `https://prod.example`, there's no `JSESSIONID` cookie. Set `ENONIC_CLI_REMOTE_USER/PASS` so the request doesn't prompt.
  - With beta off and the same runtime data, a request to `ENONIC_CLI_REMOTE_URL=https://prod.example` still carries the cookie, as on master.
- [ ] **Step 2: Run** `go test ./internal/app/commands/common/ -run TestSession`. Expected: FAIL.
- [ ] **Step 3: Implement.**
- [ ] **Step 4: Run** `go test ./...`. Expected: PASS.
- [ ] **Step 5: Commit** `git commit -m "Send a saved session only to the instance it came from (beta)"`

---

### Task 5: Login through discovery only, with `--auth-server`

**Files:**
- Modify: `internal/app/commands/common/discovery.go:33-77` (split out the issuer part)
- Modify: `internal/app/commands/common/access-token.go:33-40,108-117` (delete `IdProviderEndpoints`, `DefaultWebUrl`, `idProviderEndpoint`)
- Modify: `internal/app/commands/login.go` (flags, endpoint lookup, messages, target)
- Test: `internal/app/commands/common/discovery_test.go`

**Interfaces:**
- Consumes: `remote.GetActiveRemote()` (Task 2), `SaveAccessToken`/`RemoveAccessToken(target)` (Task 3)
- Produces:
  - `func DiscoverIssuerEndpoints(client *http.Client, issuer string, mgmtUrl *url.URL) (*OAuthEndpoints, error)`. It covers the authorization server metadata half of the current `DiscoverEndpoints`, including the issuer check and the device/token endpoint check, with `Resource = mgmtKey(mgmtUrl)`.
  - `DiscoverEndpoints` keeps its signature and calls it.
- Login behavior:
  - flags: `--auth-server`, `--client-key`, `--client-cert`, `--env` (Task 6 adds `--env`)
  - `--auth-server` set → `DiscoverIssuerEndpoints`; otherwise → `DiscoverEndpoints`
  - `errors.Is(err, ErrNoDiscovery)` → exit 1 with `<mgmt url> does not advertise a login server. Device login needs XP 8.2 or later.`
- Success line: `Logged in to <name> (<url>) as <user>`. When `Name == ""`, it's `Logged in to <url> as <user>`. Without a user, the ` as <user>` part is dropped.
- Logout lines change the same way: `Logged out from <name> (<url>)` and `Not logged in to <name> (<url>)`.

- [ ] **Step 1: Write the failing tests:**
  - `TestDiscoverIssuerEndpoints`: uses the existing `discoveryServer` helper and passes the issuer directly. It returns the device and token endpoints with `Resource` equal to the management URL.
  - `TestDiscoverIssuerEndpointsRejectsOtherIssuer`: the metadata's `issuer` differs, and the call returns an error containing `not`.
- [ ] **Step 2: Run** `go test ./internal/app/commands/common/ -run TestDiscover`. Expected: FAIL, undefined.
- [ ] **Step 3: Implement.** Delete the `--url`/`--id-provider` fallback and the `loginEndpoints` helper's fallback branch.
- [ ] **Step 4: Run** `go test ./... && go vet ./internal/app/commands/`. Expected: PASS, and no new vet findings in the touched files.
- [ ] **Step 5: Commit** `git commit -m "Log in only where discovery or --auth-server says"`

---

### Task 6: `--env` flag, beta gating and `env` commands

**Files:**
- Create: `internal/app/commands/env/env.go` (`All()` and the five subcommands)
- Create: `internal/app/commands/beta.go` (`gateBeta`, `wireEnvFlag`)
- Modify: `internal/app/commands/common/common.go:91-96` (add `ENV_FLAG` to `AUTH_AND_TLS_FLAGS`)
- Modify: `internal/app/commands/system/info.go:15`, `internal/app/commands/system/latest.go:15`. Use `AUTH_FLAG, CRED_FILE_FLAG, CLIENT_KEY_FLAG, CLIENT_CERT_FLAG` explicitly, without `ENV_FLAG`, because these commands go to fixed URLs.
- Modify: `internal/app/commands/login.go` (add `common.ENV_FLAG` to `Login` and `Logout`)
- Modify: `internal/app/commands/commands.go` (add the `env` group, then gate and wire)
- Test: `internal/app/commands/beta_test.go`, `internal/app/commands/env/env_test.go`

**Interfaces:**
- Consumes: everything from Tasks 1–3
- Produces:
  - `common.ENV_FLAG`:
    ```go
    cli.StringFlag{Name: "env", Usage: "Environment to run the command against, instead of the active one (beta)", Hidden: !util.IsBeta()}
    ```
  - `func gateBeta(cmds []cli.Command, names ...string) []cli.Command`: sets `Hidden: !util.IsBeta()` on the named top-level commands and prepends a `Before` that returns `cli.NewExitError(util.BETA_MESSAGE, 1)` when beta is off.
  - `func wireEnvFlag(cmds []cli.Command) []cli.Command`: walks the tree recursively into `Subcommands`. For every command with a flag named `env`, it prepends a `Before`:
    - if `c.IsSet("env")` and beta is off → the beta exit error
    - otherwise `remote.SelectEnv(c.String("env"))`
    - then it calls the original `Before`, if there is one
  - `commands.All()` returns `wireEnvFlag(gateBeta(list, "env", "login", "logout"))`.
- `env` subcommands, with output exactly as below:
  - **`add <name> <url> [--proxy <url>]`**
    - rejects an invalid name: `Invalid environment name '<name>'. Use lowercase letters, digits, '-' and '_'.`
    - rejects `local` or an existing name: `Environment '<name>' already exists.`
    - rejects a URL that isn't http(s) with a host: `Invalid url '<url>'.`
    - on success, stderr gets `Added environment '<name>' (<url>)`
  - **`list`**
    - stdout gets one line per environment, `local` first, then the rest sorted, formatted `%s %-<w>s  %s%s`
    - the marker is `*` for the environment `ResolveTarget` picks, a space otherwise
    - the suffix is `  logged in` when `HasAccessToken`
  - **`use <name>`**
    - fails for an unknown name with the unknown-environment message
    - saves `Active` (`local` saves as `""`)
    - stderr gets `Using environment '<name>' (<url>)`
  - **`current`**
    - stdout gets `<name or url> (<url>), selected by <Source>`
  - **`remove <name>`**
    - `local` → `The local environment cannot be removed.`
    - unknown name → the unknown-environment message
    - otherwise deletes the token file, clears `Active` if it pointed to this environment, and stderr gets `Removed environment '<name>'`
  - Any wrong number of arguments prints the subcommand's usage and exits 1.

- [ ] **Step 1: Write the failing tests:**
  - `TestBetaOffHidesCommands`: with beta off, `env`, `login` and `logout` are `Hidden`. Running `app.Run([]string{"enonic","env","list"})` returns an error whose message equals `util.BETA_MESSAGE`.
  - `TestEnvFlagRequiresBeta`: with beta off, `enonic app list --env prod` gives `BETA_MESSAGE`. Run the app with `commands.All()` and a stubbed `Action` by replacing the `app list` command's Action in the returned slice.
  - `TestEnvFlagSelectsEnvironment`: with beta on, after `wireEnvFlag`, running `x --env prod` on a stub command makes `remote.GetActiveRemote().Name == "prod"` inside the Action.
  - `TestEnvFlagOnRemoteCommands`: walks `commands.All()`. Every command whose flags include `AUTH_FLAG` also has `env`, except `system info` and `system latest`, which must not.
  - In `env/env_test.go`, `TestEnvLifecycle` runs:
    1. `add prod https://prod.example`
    2. `use prod`
    3. `current` → `prod (https://prod.example), selected by enonic env use`
    4. `list` → output marks `prod` with `*`
    5. `remove prod`, after which the token file is gone
    6. `current` → `local (http://localhost:4848), selected by default`

    Capture stdout through a swappable `var out io.Writer = os.Stdout` in the `env` package.
  - `TestEnvAddRejects`: covers an invalid name, `local`, a duplicate, and `ftp://x`.
- [ ] **Step 2: Run** `go test ./internal/app/commands/...`. Expected: FAIL.
- [ ] **Step 3: Implement.**
- [ ] **Step 4: Run** `go test ./...`. Expected: PASS.
- [ ] **Step 5: Commit** `git commit -m "Add env commands and --env, behind ENONIC_CLI_BETA"`

---

### Task 7: Sandbox proxy only from the variable

**Files:**
- Modify: `internal/app/commands/sandbox/distro.go:246` (`remote.GetActiveRemote().Proxy` → `remote.ProxyFromEnv()`)

**Interfaces:**
- Consumes: `remote.ProxyFromEnv()` (Task 2)

There's no new test. With beta off, `GetActiveRemote().Proxy` is already `ProxyFromEnv()`, so the change is behavior-neutral. With beta on, Task 2's `TestEnvProxyOverride` covers `ProxyFromEnv`.

- [ ] **Step 1: Make the one-line change.**
- [ ] **Step 2: Run** `go test ./...`. Expected: PASS.
- [ ] **Step 3: Commit** `git commit -m "Pass only ENONIC_CLI_HTTP_PROXY to the sandbox"`

---

### Task 8: Build and check end to end

- [ ] **Step 1: Build.**

  ```bash
  go build -o dist/enonic ./internal/app
  ```

  Expected: no output, exit 0.
- [ ] **Step 2: Beta off, in a temp home.**
  - `export ENONIC_CLI_HOME_PATH=$(mktemp -d)`
  - `./dist/enonic --help` lists no `env`/`login`/`logout`.
  - `./dist/enonic env list` prints the beta message and exits with code 1.
  - `./dist/enonic app list --env x` prints the beta message.
- [ ] **Step 3: Beta on.**
  - `export ENONIC_CLI_BETA=true`
  - `./dist/enonic env add kk https://sry-kk-test-mgmt.enonic-dev.cloud && ./dist/enonic env use kk && ./dist/enonic env current`. Expected: `kk (https://sry-kk-test-mgmt.enonic-dev.cloud), selected by enonic env use`.
  - `./dist/enonic login` prints a code and a URL on `sry-kk-test.enonic-dev.cloud/_/idprovider/enonic-cloud/device`.
  - **The user approves it in the browser.** The agent stops here and asks.
- [ ] **Step 4: After approval.**
  - `./dist/enonic app list` lists apps without prompting.
  - `./dist/enonic app list --env local` targets localhost: it fails to connect, or prompts if a sandbox is running.
  - `ls -l $ENONIC_CLI_HOME_PATH/.enonic/tokens/` shows `kk.json` with `-rw-------`.
- [ ] **Step 5: Mark the spec's status** `Implemented`, then commit the spec, the plan and the status change: `git commit -m "Design and plan for environments and device login"`.
