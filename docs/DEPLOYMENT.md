# Deployment Guide

This document covers everything needed to deploy `fhp-reports` and `fhp-bioguide` — both via the automated CI pipeline and manually when CI is unavailable.

---

## 1. Server

| Property      | Value                                                               |
|---------------|---------------------------------------------------------------------|
| Host          | `fhp-registrera.swedencentral.cloudapp.azure.com`                   |
| User          | `sysadmin`                                                          |
| OS            | Ubuntu (Linux amd64)                                                |
| Local SSH key | `/Users/mathiast/Documents/FHP/vmnetintegrate_sysadmin_key.pem`    |

To open an interactive shell on the server:

```bash
ssh -i /Users/mathiast/Documents/FHP/vmnetintegrate_sysadmin_key.pem \
    sysadmin@fhp-registrera.swedencentral.cloudapp.azure.com
```

---

## 2. Services

### Service table

| Service name        | Binary path                              | Working directory             |
|---------------------|------------------------------------------|-------------------------------|
| `fhp-reports`       | `/home/sysadmin/fhp-reports/fhp-reports` | `/home/sysadmin/fhp-reports/` |
| `fhp-bioguide`      | `/home/sysadmin/fhp-bioguide/fhp-bioguide` | `/home/sysadmin/fhp-bioguide/` |

Both services are configured with `Restart=always` in systemd.

### Common systemctl commands

```bash
# Check status of both services
sudo systemctl status fhp-reports fhp-bioguide

# Restart both services
sudo systemctl restart fhp-reports fhp-bioguide

# Restart a single service
sudo systemctl restart fhp-reports
sudo systemctl restart fhp-bioguide

# Stop a service
sudo systemctl stop fhp-reports

# Start a stopped service
sudo systemctl start fhp-reports

# Enable auto-start on boot (if not already enabled)
sudo systemctl enable fhp-reports fhp-bioguide
```

### Log file locations

Logs are managed by journald. To tail logs in real time:

```bash
# Live logs for fhp-reports
sudo journalctl -fu fhp-reports

# Live logs for fhp-bioguide
sudo journalctl -fu fhp-bioguide

# Last 100 lines
sudo journalctl -n 100 -u fhp-reports
sudo journalctl -n 100 -u fhp-bioguide
```

---

## 3. Automated Deployment (CI)

### Workflow file

`.github/workflows/deploy.yml` — triggers automatically on every push to `main`, and can also be triggered manually.

### What the workflow does (step by step)

1. **Checkout** — checks out the repository at the pushed commit.
2. **Set up Go** — installs the Go version from `go.mod`, with module cache enabled.
3. **Build Linux binaries** — runs `make build-linux`, which cross-compiles both apps for `linux/amd64` with `CGO_ENABLED=0`. Outputs land in `out/fhp-reports/fhp-reports` and `out/fhp-bioguide/fhp-bioguide`.
4. **Set up SSH** — writes `DEPLOY_KNOWN_HOST` to `~/.ssh/known_hosts` and `DEPLOY_SSH_KEY` to `~/.ssh/deploy_key` (mode 600).
5. **Deploy fhp-reports binary** — rsyncs `out/fhp-reports/fhp-reports` to `/home/sysadmin/fhp-reports/fhp-reports` on the server.
6. **Deploy fhp-bioguide binary** — rsyncs `out/fhp-bioguide/fhp-bioguide` to `/home/sysadmin/fhp-bioguide/fhp-bioguide`.
7. **Deploy views** — rsyncs the local `views/` directory to `/home/sysadmin/fhp-reports/views/` with `--delete` so removed templates are cleaned up.
8. **Post-deploy** — SSHs into the server and runs:
   - `chmod +x` on both binaries
   - `sudo setcap CAP_NET_BIND_SERVICE=+eip ~/fhp-reports/fhp-reports` (required every time the binary is replaced — see section 5)
   - `sudo systemctl restart fhp-reports fhp-bioguide`
9. **Verify services** — polls `systemctl is-active` for up to 20 seconds per service, fails the workflow if either service does not come up.

### Triggering manually

In the GitHub UI: go to **Actions → Build and Deploy → Run workflow** and click the button. This runs the same pipeline as a push to `main`.

### Required GitHub secrets

Go to **Settings → Secrets and variables → Actions** and ensure all four secrets are set:

| Secret name         | Description                                                  |
|---------------------|--------------------------------------------------------------|
| `DEPLOY_SSH_KEY`    | Private SSH key (PEM format) for the `sysadmin` user         |
| `DEPLOY_HOST`       | Hostname: `fhp-registrera.swedencentral.cloudapp.azure.com`  |
| `DEPLOY_USER`       | Username: `sysadmin`                                         |
| `DEPLOY_KNOWN_HOST` | The server's ed25519 host key line (see below)               |

#### Obtaining DEPLOY_KNOWN_HOST

Run this locally to get the current host key:

```bash
ssh-keyscan fhp-registrera.swedencentral.cloudapp.azure.com 2>/dev/null | grep ed25519
```

The output will look like:

```
fhp-registrera.swedencentral.cloudapp.azure.com ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIFvmukfEdAmrnLIAexVF8azC7KVvbuhYTrVUuvz0rRlZ
```

Paste the entire line as the value of `DEPLOY_KNOWN_HOST`.

#### Rotating the SSH key

1. Generate a new key pair: `ssh-keygen -t ed25519 -f deploy_key_new -C "fhp-ci-deploy"`
2. Add the public key (`deploy_key_new.pub`) to `~/.ssh/authorized_keys` on the server.
3. Update the `DEPLOY_SSH_KEY` secret in GitHub with the contents of `deploy_key_new`.
4. Verify a deploy succeeds, then remove the old public key from `authorized_keys`.

---

## 4. Emergency Manual Deployment

Use this procedure when CI is unavailable (e.g., GitHub Actions down, broken workflow, urgent hotfix).

### Prerequisites

- Go toolchain installed locally (version matching `go.mod`)
- SSH access to the server using the local key

### Deploy fhp-reports

```bash
# 1. Build the Linux binary
make reports-linux
# Output: out/fhp-reports/fhp-reports

# 2. Copy the binary to the server
scp -i /Users/mathiast/Documents/FHP/vmnetintegrate_sysadmin_key.pem \
    out/fhp-reports/fhp-reports \
    sysadmin@fhp-registrera.swedencentral.cloudapp.azure.com:/home/sysadmin/fhp-reports/fhp-reports

# 3. Sync views/ (--delete removes files that no longer exist locally)
rsync -az --delete \
    -e "ssh -i /Users/mathiast/Documents/FHP/vmnetintegrate_sysadmin_key.pem" \
    views/ \
    sysadmin@fhp-registrera.swedencentral.cloudapp.azure.com:/home/sysadmin/fhp-reports/views/

# 4. Set the capability, make executable, and restart (MUST run setcap after every binary replacement)
ssh -i /Users/mathiast/Documents/FHP/vmnetintegrate_sysadmin_key.pem \
    sysadmin@fhp-registrera.swedencentral.cloudapp.azure.com \
    "chmod +x ~/fhp-reports/fhp-reports && sudo setcap CAP_NET_BIND_SERVICE=+eip ~/fhp-reports/fhp-reports && sudo systemctl restart fhp-reports"

# 5. Verify
ssh -i /Users/mathiast/Documents/FHP/vmnetintegrate_sysadmin_key.pem \
    sysadmin@fhp-registrera.swedencentral.cloudapp.azure.com \
    "sudo systemctl status fhp-reports"
```

### Deploy fhp-bioguide

```bash
# 1. Build the Linux binary
make bioguidesync-linux
# Output: out/fhp-bioguide/fhp-bioguide

# 2. Copy the binary to the server
scp -i /Users/mathiast/Documents/FHP/vmnetintegrate_sysadmin_key.pem \
    out/fhp-bioguide/fhp-bioguide \
    sysadmin@fhp-registrera.swedencentral.cloudapp.azure.com:/home/sysadmin/fhp-bioguide/fhp-bioguide

# 3. Make executable and restart (no setcap needed for fhp-bioguide)
ssh -i /Users/mathiast/Documents/FHP/vmnetintegrate_sysadmin_key.pem \
    sysadmin@fhp-registrera.swedencentral.cloudapp.azure.com \
    "chmod +x ~/fhp-bioguide/fhp-bioguide && sudo systemctl restart fhp-bioguide"

# 4. Verify
ssh -i /Users/mathiast/Documents/FHP/vmnetintegrate_sysadmin_key.pem \
    sysadmin@fhp-registrera.swedencentral.cloudapp.azure.com \
    "sudo systemctl status fhp-bioguide"
```

---

## 5. Why setcap is Required

`fhp-reports` listens on port 443 (HTTPS). On Linux, binding to any port below 1024 normally requires root privileges. Rather than running the process as root, the binary is granted a single focused capability: `CAP_NET_BIND_SERVICE`.

This capability is stored as file metadata on the binary itself (not on the process or the user). That means:

- It is **not inherited from the parent process** — it belongs to the file.
- It is **lost every time the binary file is replaced**, even if the new file has identical content, because `scp`/`rsync` create a new inode.

Therefore, `sudo setcap CAP_NET_BIND_SERVICE=+eip /home/sysadmin/fhp-reports/fhp-reports` **must be run after every deployment** of the `fhp-reports` binary.

The CI workflow handles this automatically in the post-deploy step. When deploying manually, always remember step 4 in the fhp-reports procedure above. If you forget, the service will start but immediately fail to bind port 443.

To verify the capability is set correctly on the server:

```bash
getcap ~/fhp-reports/fhp-reports
# Expected output: /home/sysadmin/fhp-reports/fhp-reports cap_net_bind_service=eip
```

---

## 6. Config Files

Both apps read `config.yaml` from their working directory at startup. The app will panic immediately if the file is missing.

**`config.yaml` is not managed by CI and must never be committed to git.**

It contains secrets including:
- Dynamics 365 OAuth2 credentials (`clientID`, `clientSecret`, `tenantID`, resource URL)
- BioGuiden SOAP credentials (`Username`, `Password`, URL)
- MySQL connection string
- Bearer token for `/api/*` route authorization
- Power Automate webhook URL and Basic auth credentials

The files live on the server at:
- `/home/sysadmin/fhp-reports/config.yaml`
- `/home/sysadmin/fhp-bioguide/config.yaml`

If a new server is provisioned or a config value must be changed, update the file directly on the server over SSH. Keep a copy of the config (with secrets) in a secure location (e.g., a password manager or Azure Key Vault) — it is not recoverable from git.
