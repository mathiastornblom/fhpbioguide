# GitHub CI/CD Deployment Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a GitHub Actions workflow that builds Linux binaries on every push to `main` and deploys them plus `views/` to the production Azure VM, then verifies both systemd services are active.

**Architecture:** Single workflow file (`.github/workflows/deploy.yml`) with one job: build both binaries with `make build-linux`, rsync artifacts to server over SSH, run post-deploy commands (`chmod`, `setcap`, `systemctl restart`), verify services. Documentation lives in `docs/DEPLOYMENT.md`.

**Tech Stack:** GitHub Actions, `ubuntu-latest` runner, Go 1.26.1, `rsync`, `ssh`, systemd on Azure Ubuntu VM.

---

### Task 1: Create the GitHub Actions workflow

**Files:**
- Create: `.github/workflows/deploy.yml`

- [ ] **Step 1: Create the workflow file**

Create `.github/workflows/deploy.yml` with this exact content:

```yaml
name: Build and Deploy

on:
  push:
    branches: [main]
  workflow_dispatch:

jobs:
  build-and-deploy:
    runs-on: ubuntu-latest

    steps:
      - name: Checkout
        uses: actions/checkout@v4

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version-file: go.mod
          cache: true

      - name: Build Linux binaries
        run: make build-linux

      - name: Set up SSH
        run: |
          mkdir -p ~/.ssh
          echo "${{ secrets.DEPLOY_SSH_KEY }}" > ~/.ssh/deploy_key
          chmod 600 ~/.ssh/deploy_key
          ssh-keyscan -H ${{ secrets.DEPLOY_HOST }} >> ~/.ssh/known_hosts

      - name: Deploy fhp-reports binary
        run: |
          rsync -az \
            -e "ssh -i ~/.ssh/deploy_key" \
            out/fhp-reports/fhp-reports \
            ${{ secrets.DEPLOY_USER }}@${{ secrets.DEPLOY_HOST }}:/home/sysadmin/fhp-reports/fhp-reports

      - name: Deploy fhp-bioguide binary
        run: |
          rsync -az \
            -e "ssh -i ~/.ssh/deploy_key" \
            out/fhp-bioguide/fhp-bioguide \
            ${{ secrets.DEPLOY_USER }}@${{ secrets.DEPLOY_HOST }}:/home/sysadmin/fhp-bioguide/fhp-bioguide

      - name: Deploy views
        run: |
          rsync -az --delete \
            -e "ssh -i ~/.ssh/deploy_key" \
            views/ \
            ${{ secrets.DEPLOY_USER }}@${{ secrets.DEPLOY_HOST }}:/home/sysadmin/fhp-reports/views/

      - name: Post-deploy (setcap + restart services)
        run: |
          ssh -i ~/.ssh/deploy_key \
            ${{ secrets.DEPLOY_USER }}@${{ secrets.DEPLOY_HOST }} \
            bash << 'ENDSSH'
          chmod +x ~/fhp-reports/fhp-reports ~/fhp-bioguide/fhp-bioguide
          sudo setcap CAP_NET_BIND_SERVICE=+eip ~/fhp-reports/fhp-reports
          sudo systemctl restart fhp-reports fhp-bioguide
          ENDSSH

      - name: Verify services are active
        run: |
          ssh -i ~/.ssh/deploy_key \
            ${{ secrets.DEPLOY_USER }}@${{ secrets.DEPLOY_HOST }} \
            bash << 'ENDSSH'
          sleep 5
          systemctl is-active fhp-reports || (echo "fhp-reports failed to start" && exit 1)
          systemctl is-active fhp-bioguide || (echo "fhp-bioguide failed to start" && exit 1)
          echo "Both services active"
          ENDSSH
```

- [ ] **Step 2: Verify the file is syntactically valid YAML**

Run:
```bash
python3 -c "import yaml; yaml.safe_load(open('.github/workflows/deploy.yml'))" && echo "YAML OK"
```
Expected: `YAML OK`

- [ ] **Step 3: Commit**

```bash
git add .github/workflows/deploy.yml
git commit -m "ci: add GitHub Actions build and deploy workflow"
```

---

### Task 2: Create deployment documentation

**Files:**
- Create: `docs/DEPLOYMENT.md`

- [ ] **Step 1: Write the documentation file**

Create `docs/DEPLOYMENT.md` with this exact content:

```markdown
# Deployment Guide

## Server

- **Host:** `fhp-registrera.swedencentral.cloudapp.azure.com`
- **User:** `sysadmin`
- **SSH key (local):** `/Users/mathiast/Documents/FHP/vmnetintegrate_sysadmin_key.pem`
- **OS:** Ubuntu (Azure VM, running since 2025-08-25)

## Services

Both apps run as systemd services with `Restart=always`:

| Service | Binary | Working dir |
|---|---|---|
| `fhp-reports.service` | `/home/sysadmin/fhp-reports/fhp-reports` | `/home/sysadmin/fhp-reports/` |
| `fhp-bioguide.service` | `/home/sysadmin/fhp-bioguide/fhp-bioguide` | `/home/sysadmin/fhp-bioguide/` |

### Common commands (run on server)

```bash
# Check status of both
sudo systemctl status fhp-reports fhp-bioguide

# Restart both
sudo systemctl restart fhp-reports fhp-bioguide

# Tail live logs
journalctl -u fhp-reports -f
journalctl -u fhp-bioguide -f

# View logs from today
journalctl -u fhp-reports --since today --no-pager
```

Log files are also written to:
- `/home/sysadmin/fhp-reports/fhp-reports.log`
- `/home/sysadmin/fhp-bioguide/fhp-bioguide.log`

## Automated Deployment (CI)

Every push to `main` triggers `.github/workflows/deploy.yml`, which:

1. Builds both Linux binaries (`make build-linux`)
2. rsync's binaries and `views/` to the server
3. Sets port-binding capability on `fhp-reports` (required for port 443)
4. Restarts both systemd services
5. Verifies both services are `active`

You can also trigger a deploy manually: go to the repository on GitHub →
**Actions** → **Build and Deploy** → **Run workflow**.

### GitHub secrets required

| Secret | Description |
|---|---|
| `DEPLOY_SSH_KEY` | Full contents of `vmnetintegrate_sysadmin_key.pem` |
| `DEPLOY_HOST` | `fhp-registrera.swedencentral.cloudapp.azure.com` |
| `DEPLOY_USER` | `sysadmin` |

To rotate the SSH key: generate a new key pair, add the public key to
`~/.ssh/authorized_keys` on the server, update `DEPLOY_SSH_KEY` in GitHub
secrets, then remove the old public key from the server.

## Emergency Manual Deployment

Use this when CI is unavailable (e.g. GitHub outage).

### Deploy fhp-reports

```bash
# 1. Build locally (cross-compile for Linux)
make reports-linux

# 2. Copy binary to server
scp -i /Users/mathiast/Documents/FHP/vmnetintegrate_sysadmin_key.pem \
  out/fhp-reports/fhp-reports \
  sysadmin@fhp-registrera.swedencentral.cloudapp.azure.com:/home/sysadmin/fhp-reports/fhp-reports

# 3. Copy views (if changed)
rsync -az --delete \
  -e "ssh -i /Users/mathiast/Documents/FHP/vmnetintegrate_sysadmin_key.pem" \
  views/ \
  sysadmin@fhp-registrera.swedencentral.cloudapp.azure.com:/home/sysadmin/fhp-reports/views/

# 4. On server: set capability and restart
ssh -i /Users/mathiast/Documents/FHP/vmnetintegrate_sysadmin_key.pem \
  sysadmin@fhp-registrera.swedencentral.cloudapp.azure.com \
  "chmod +x ~/fhp-reports/fhp-reports && sudo setcap CAP_NET_BIND_SERVICE=+eip ~/fhp-reports/fhp-reports && sudo systemctl restart fhp-reports"
```

### Deploy fhp-bioguide

```bash
# 1. Build
make bioguidesync-linux

# 2. Copy binary
scp -i /Users/mathiast/Documents/FHP/vmnetintegrate_sysadmin_key.pem \
  out/fhp-bioguide/fhp-bioguide \
  sysadmin@fhp-registrera.swedencentral.cloudapp.azure.com:/home/sysadmin/fhp-bioguide/fhp-bioguide

# 3. On server: restart
ssh -i /Users/mathiast/Documents/FHP/vmnetintegrate_sysadmin_key.pem \
  sysadmin@fhp-registrera.swedencentral.cloudapp.azure.com \
  "chmod +x ~/fhp-bioguide/fhp-bioguide && sudo systemctl restart fhp-bioguide"
```

## Why setcap is required for fhp-reports

`fhp-reports` listens on port 443 (HTTPS). On Linux, binding ports below 1024
requires either running as root or having the `CAP_NET_BIND_SERVICE` capability
set on the binary. We use `setcap` so the service can run as the unprivileged
`sysadmin` user.

**Important:** `setcap` is stored per file. Every time the binary is replaced
(deploy), `setcap` must be re-run. This is done automatically by CI.

## Config Files

`config.yaml` in each app directory is **not managed by CI** — it contains
secrets (D365 credentials, BioGuiden credentials, MySQL DSN) and must be
managed manually on the server. Never commit `config.yaml` to git.
```

- [ ] **Step 2: Commit**

```bash
git add docs/DEPLOYMENT.md
git commit -m "docs: add deployment guide covering CI workflow and manual steps"
```

---

### Task 3: Push to main and verify CI runs

- [ ] **Step 1: Push to GitHub**

```bash
git push origin main
```

- [ ] **Step 2: Open GitHub Actions in browser**

Navigate to the repository → **Actions** tab. You should see a workflow run
named **"Build and Deploy"** triggered by the push. Click into it.

- [ ] **Step 3: Watch the run complete**

All steps should go green. The final "Verify services are active" step should
print:
```
Both services active
```

If the run fails, click the failing step to read the error, then debug from there.

- [ ] **Step 4: Confirm on server**

```bash
ssh -i /Users/mathiast/Documents/FHP/vmnetintegrate_sysadmin_key.pem \
  sysadmin@fhp-registrera.swedencentral.cloudapp.azure.com \
  "sudo systemctl status fhp-reports fhp-bioguide --no-pager | grep -E 'Active|Main PID'"
```

Expected output (timestamps will differ):
```
     Active: active (running) since ...
   Main PID: ...
     Active: active (running) since ...
   Main PID: ...
```
