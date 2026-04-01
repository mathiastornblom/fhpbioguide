# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
# Build both apps (current OS)
make build

# Build both apps (Linux amd64, cross-compile)
make build-linux

# Build individual apps
make reports        # fhpreports only
make bioguidesync   # fhpbioguide only

# Clean build output
make clean

# Dependencies
go mod download

# Run a single test
go test ./pkg/...
go test ./pkg/usecase/movieexport/...
```

Build output lands in `out/fhp-reports/` and `out/fhp-bioguide/`.

## Configuration

Both apps read `config.yaml` from the working directory via Viper. The app panics at startup if the file is missing. Key config sections:

- `Dynamics` — D365 OAuth2 credentials (clientID, clientSecret, tenantID, URL)
- `bio` — BioGuiden SOAP credentials (Username, Password, URL)
- `report` — fhpreports domain and Bearer token for `/api/*` authorization
- `proxy` — Movie Transit Logic App webhook URL and Basic auth

## Architecture

This is a **dual-application Go monorepo** sharing a single `pkg/` layer.

### App 1: fhpbioguide (`cmd/fhpbioguide/`)
A daemon that runs a **nightly sync at 02:00** (via gocron). Pulls data from the **BioGuiden SOAP API** and pushes to **Dynamics 365 REST API**.

Three export pipelines, each following the same pattern:
| Export | BioGuiden source | D365 destination |
|---|---|---|
| MovieExport | XML movie catalog (2018–2030) | Product entities |
| TheatreExport | Theatres and salons | `new_lokal` + accounts |
| CashExport | 24h cash reports | `new_cashreport` entities |

### App 2: fhpreports (`cmd/fhpreports/`)
A **Fiber HTTPS web service** (port 443, Let's Encrypt) serving HTML forms for cinema staff to report ticket sales. Form data is written to MySQL and pushed to D365 booking entities, then a Power Automate webhook is triggered.

Form types:
- Type 0: Presale (expires 24h)
- Type 1: Sold (8 ticket categories: Ordinaire, Fribiljett, Barn/Ungdom, Abonnemang, Scenpass Sverige, Sveriges konstföreningar, Met-rabatt, Annan)

HTTP routes:
- `GET /form/:ID` — Render form HTML
- `POST /form-post/:ID` — Submit → D365 + Power Automate webhook
- `GET /api/status` — Health check
- `POST /api/genform/presale/:ID` — Create presale form
- `POST /api/genform/sold/:ID` — Create sold form
- `POST /api/regenform/sold/:ID` — Recreate sold form
- `POST /api/orderstatus` — Proxy to Movie Transit Logic App

### Shared `pkg/` Layer

Follows a **clean/layered architecture**:

```
entity/      → Data models (XML structs for BioGuiden, JSON for D365, GORM models for MySQL)
repository/  → Data access: wraps D365 REST calls, BioGuiden SOAP calls, and GORM queries
usecase/     → Thin service structs per domain; delegate to repositories
api/
  d365/      → D365 OAuth2 client (client credentials flow, token cached per sync run)
  bioguide/  → BioGuiden SOAP client (XML envelope construction)
  handler/   → export.go (App 1 orchestration), reportforms.go (App 2 Fiber handlers)
  api.go     → Fiber app setup: middleware, TLS, route registration
```

### Key Implementation Notes

- **D365 auth**: Client credentials OAuth2; token is fetched once at the start of each sync job and reused.
- **BioGuiden SOAP**: XML envelopes built manually; HTML-escaped to prevent XML injection.
- **Cash report deduplication**: `shouldLinkBooking()` in `pkg/api/handler/export.go` prevents re-linking a booking to a different cash report number.
- **MySQL**: Tables `Form` and `Event` are auto-migrated by GORM at fhpreports startup (default DSN: `root:root@localhost:3306`).
- **Authorization**: Bearer token checked on all `/api/*` routes; expected value from `config.yaml report.Bearer`.
