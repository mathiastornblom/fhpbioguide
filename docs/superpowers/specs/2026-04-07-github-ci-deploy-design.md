# GitHub CI/CD Deployment Design

**Date:** 2026-04-07  
**Status:** Approved

## Context

Both `fhp-reports` and `fhp-bioguide` run on an Azure VM
(`fhp-registrera.swedencentral.cloudapp.azure.com`) as systemd services.
Previously, deployment was fully manual (scp + pkill + restart), which caused
outages when steps were missed. This spec covers automating build and deploy
via GitHub Actions.

## Goal

On every push to `main` (and on demand), automatically build Linux binaries
and deploy them — along with the `views/` assets — to the production server,
then verify both services are running.

## Workflow: `.github/workflows/deploy.yml`

### Triggers

```yaml
on:
  push:
    branches: [main]
  workflow_dispatch:
```

### Runner

`ubuntu-latest` (GitHub-hosted). No self-hosted runner needed.

### Job: `build-and-deploy`

#### Step 1 — Checkout
Standard `actions/checkout@v4`.

#### Step 2 — Setup Go
`actions/setup-go@v5` with `go-version-file: go.mod` (reads `go 1.26.1`).
Module cache keyed on `go.sum` hash via `actions/cache@v4` on `~/go/pkg/mod`.

#### Step 3 — Build
```
make build-linux
```
Produces:
- `out/fhp-reports/fhp-reports` (Linux amd64, CGO_ENABLED=0)
- `out/fhp-bioguide/fhp-bioguide` (Linux amd64, CGO_ENABLED=0)

#### Step 4 — Write SSH key
`${{ secrets.DEPLOY_SSH_KEY }}` written to a temp file (`~/.ssh/deploy_key`),
permissions set to `600`.  
Server host key added to `known_hosts` via `ssh-keyscan`.

#### Step 5 — rsync files to server
Three rsync calls:
1. `out/fhp-reports/fhp-reports` → `/home/sysadmin/fhp-reports/fhp-reports`
2. `out/fhp-bioguide/fhp-bioguide` → `/home/sysadmin/fhp-bioguide/fhp-bioguide`
3. `views/` → `/home/sysadmin/fhp-reports/views/` (recursive, delete stale files)

#### Step 6 — Post-deploy on server (single SSH session)
```bash
chmod +x ~/fhp-reports/fhp-reports ~/fhp-bioguide/fhp-bioguide
sudo setcap CAP_NET_BIND_SERVICE=+eip ~/fhp-reports/fhp-reports
sudo systemctl restart fhp-reports fhp-bioguide
```

`sysadmin` has passwordless sudo for `setcap` and `systemctl` — confirmed.

#### Step 7 — Verify
```bash
sleep 5
systemctl is-active fhp-reports
systemctl is-active fhp-bioguide
```
If either exits non-zero, the workflow step fails and GitHub marks the run red.

### Secrets (already configured in repo)

| Secret | Value |
|---|---|
| `DEPLOY_SSH_KEY` | Contents of `vmnetintegrate_sysadmin_key.pem` |
| `DEPLOY_HOST` | `fhp-registrera.swedencentral.cloudapp.azure.com` |
| `DEPLOY_USER` | `sysadmin` |

## Documentation: `docs/DEPLOYMENT.md`

Covers:
- Server overview (Azure VM, systemd services)
- Manual service management (`systemctl status/restart/logs`)
- Manual deployment steps (for emergencies when CI is unavailable)
- The `setcap` requirement for `fhp-reports` and why it's needed
- How to add/rotate the SSH deploy key

## Out of Scope

- No staging environment
- No rollback automation (manual: `sudo systemctl restart` after reverting binary)
- No Slack/email notifications on deploy failure (GitHub email notifications suffice)
- Config file (`config.yaml`) is NOT deployed by CI — managed manually on server
