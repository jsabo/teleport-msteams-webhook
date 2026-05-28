# teleport-msteams-webhook — Claude Context

## What this is

A standalone Go tool that watches Teleport access requests and posts Adaptive Card
notifications to Microsoft Teams channels via incoming webhooks.

Uses Power Automate Workflow-based webhooks (not Azure AD connectors, which are deprecated).
A channel owner creates the webhook URL in ~2 clicks. No Azure AD registration, no admin approval.

## Architecture

```
cmd/teleport-msteams-webhook/main.go    flags, signal handling, dry-run
internal/config/config.go               TOML config; builds teleport/api client
internal/bot/                           Card builder, HTTP POST helper, Bot type
internal/plugin/plugin.go               Reconnecting watcher loop; event dispatch; routing
```

## Recipient routing

Two-layer fallback (no access monitoring rules):
1. Look up each requested role in `role_to_recipients` -> use those URLs if found
2. Fall back to `"*"` wildcard entry
3. Deduplicate across all resolved URLs

Email-style strings in `role_to_recipients` are silently ignored — only `https://` URLs
are accepted by `bot.Post`.

## Card logo

Default logo URL: `https://raw.githubusercontent.com/jsabo/teleport-msteams-webhook/main/assets/teleport-logo.png`

Teams fetches images from its client side, not from our tool. Teams aggressively caches
Adaptive Card images, so this is not a per-request hit on GitHub.

Why not base64: `assets/teleport-logo.png` is ~74KB -> ~99KB base64 -> exceeds Teams
incoming webhook POST body limit (~28KB).

Logo is configurable:
- `[msteams] logo_url = "..."` -- use a different URL
- `[msteams] disable_logo = true` -- omit image entirely

## Credential refresh

`config.Config.NewClient()` uses `client.DynamicIdentityFileCreds`. When
`refresh_identity = true`, a background goroutine calls `creds.Reload()` every minute,
picking up renewed certs written by tbot without any restart.

## Tests

```bash
go test ./...           # all unit tests
go test -race ./...     # with race detector
```

All tests are offline (no Teleport cluster needed). Tests use `httptest.Server` for
webhook tests and table-driven TOML parsing for config tests.

## Dry run

```bash
teleport-msteams-webhook -config /etc/teleport-msteams-webhook.toml -dry-run
```

Connects to Teleport (validates credentials), POSTs a sample card to every configured
webhook URL, then exits. Use this to verify the full chain before starting the watcher.

## Deployment

- Linux/systemd: `deploy/systemd/` -- service file, tbot config example, plugin config example
- Kubernetes: `deploy/helm/teleport-msteams-webhook/` -- Helm chart with tbot init+sidecar containers
