# fhpbioguide — Agent Reference

## Project Summary

Dual-application Go monorepo for **Folkets Hus och Parker (FHP)**, a Swedish cinema/theatre organization.

| App | Entry point | What it does |
|-----|-------------|--------------|
| `fhpbioguide` | `cmd/fhpbioguide/main.go` | Nightly gocron daemon — pulls BioGuiden SOAP → pushes Dynamics 365 |
| `fhpreports` | `cmd/fhpreports/main.go` | Fiber HTTPS web API — cinema staff submit ticket sales via HTML forms |

Both apps share `pkg/` and read `config.yaml` via Viper. Build output lands in `out/`.

External systems:
- **BioGuiden** — SOAP/XML API (source of movies, theatres, cash reports)
- **Dynamics 365** — REST/OData v9.2 CRM (destination for all data)
- **MySQL** — local form state (fhpreports only)
- **Power Automate** — webhook triggered on form submission
- **Movie Transit** — order status proxy via Logic App

---

## Build and Run

```bash
make build          # build both apps (current OS)
make build-linux    # cross-compile for Linux amd64
make reports        # fhpreports only
make bioguidesync   # fhpbioguide only
make clean

go mod download
go test ./pkg/...
```

Both apps require `config.yaml` in the working directory. They panic at startup if the file is missing.

---

## Configuration (`config.yaml`)

No defaults — all keys must be present.

| Section | Key | Description |
|---------|-----|-------------|
| `dynamics` | `url` | D365 org URL, e.g. `https://org.crm4.dynamics.com` |
| `dynamics` | `tenantid` | Azure AD tenant GUID |
| `dynamics` | `clientid` | Azure AD app registration client ID |
| `dynamics` | `clientsecret` | Azure AD client secret |
| `bio` | `url` | BioGuiden base URL (`https://service.bioguiden.se`) |
| `bio` | `username` | BioGuiden login |
| `bio` | `password` | BioGuiden password |
| `report` | `url` | Public hostname for fhpreports TLS cert and form URL generation |
| `report` | `Bearer` | Expected `Authorization` header value for all `/api/*` routes |
| `proxy` | `URL` | Movie Transit Logic App webhook URL |
| `proxy` | `Bearer` | Expected `Authorization` header value for `/api/orderstatus` |

---

## App 1: `fhpbioguide` — Nightly Sync Service

Starts a gocron scheduler that fires at **02:00 local time** every day. At each run it:

1. Re-authenticates against D365 (token also auto-refreshes mid-run if it expires).
2. Constructs fresh repository and service instances.
3. Calls `handler.ExecuteExports`, which runs MovieExport then CashExport (TheatreExport is currently commented out in `ExecuteExports`).

Logs to `fhpbioguide.log` in the working directory.

### Three Export Pipelines

| Export | BioGuiden service | Schema version | D365 entity | Repository |
|--------|------------------|----------------|-------------|------------|
| **MovieExport** | `moviesexport.asmx` | `MoviesExportSchema1_13.xsd` | `products` | `pkg/repository/movieexport_repository.go` |
| **TheatreExport** | `TheatreExport.asmx` | (v1.1) | `new_lokals` | `pkg/repository/theatreexport_repository.go` |
| **CashExport** | `CashReportsDistributorExport.asmx` | `CashReportsDistributorExportSchema1_3.xsd` | `new_cashreports` | `pkg/repository/cashreport_repository.go` |

#### MovieExport

- Fetches all movies updated between 2018-01-01 and 2030-12-31.
- Skips movies whose `distributor.name` is not `"Folkets Hus och Parker"`.
- Skips movies already in D365 (matched by the third segment of `full-movie-number`, e.g. `FHP-2023-12345` → `12345`).
- Skips movies with missing premiere date or runtime.
- Creates a `Product` entity in D365 via POST to `products`.
- Rating is mapped to a D365 option set integer by `censurToID()`.

#### TheatreExport

- Fetches all theatres and their salons.
- Checks if the salon's FKB number already exists in D365 (`new_lokals`); skips if so.
- Maps BioGuiden technology flags (2K, 3D, Dolby Atmos, IMAX, 35mm, 70mm, 5.1, 7.1, 4DX) to boolean fields on `LokalDynamics`.
- POSTs to `new_lokals`.
- **Currently not called by the nightly job** (`ExecuteExports` has `TheatreExport` commented out).

#### CashExport

Fetches cash reports updated within the last 24 hours from BioGuiden's `CashReportsDistributorExport` service.

For each report × show × ticket-detail row:
1. Extracts movie number (third segment of `full-movie-number`).
2. Looks up matching `Product` and `new_lokal` in D365 by movie number and FKB number.
3. Looks up a matching D365 booking (`new_bokningarkunds`) by account + product + show date.
4. Calls `shouldLinkBooking` (see below) before setting `new_booking@odata.bind`.
5. If a booking is linked, also PATCHes the booking to set `new_Lokaler`.
6. Creates one `new_cashreport` row per ticket-detail category.

**`shouldLinkBooking` deduplication** (`pkg/api/handler/export.go`):
Prevents a cash report row from linking to a booking that is already tied to a *different* cash report number. It queries `new_cashreports?$filter=_new_booking_value eq <bookingID>` and allows the link only if no existing row is found, or all existing rows share the same cash report number.

**`RecordedAmount`** is read from `total-distributor-amount` per show (comma decimal separator replaced with `.` before parsing).

---

## App 2: `fhpreports` — Web Form API

Fiber HTTPS service on port 443 with Let's Encrypt auto-cert (HTTP→HTTPS redirect on port 80). Certs cached in `./certs/`. HTTP/2 is intentionally disabled (Fasthttp limitation).

GORM auto-migrates `Form` and `Event` tables at startup against MySQL at `root:root@tcp(127.0.0.1:3306)/fhpreports`.

### Routes

| Method | Path | Auth | Handler |
|--------|------|------|---------|
| `GET` | `/form/:ID` | None | Render presale or sold HTML form |
| `POST` | `/form-post/:ID` | None | Submit form → D365 + Power Automate webhook, then delete form |
| `GET` | `/api/status` | None | Health check (200 OK) |
| `POST` | `/api/genform/presale/:ID` | Bearer | Create presale form for D365 customer ID |
| `POST` | `/api/genform/sold/:ID` | Bearer | Create sold form for D365 customer ID |
| `POST` | `/api/regenform/sold/:ID` | Bearer | Recreate sold form for a single D365 booking ID |
| `POST` | `/api/orderstatus` | Bearer | Proxy body to Movie Transit Logic App |

All `/api/*` routes check `Authorization` header against `report.Bearer` from config (except `/api/orderstatus` which checks `proxy.Bearer`). Returns 403 on mismatch.

Static CSS served from `./views/css`. HTML templates in `./views/` (`.html` extension).

### Form Lifecycle

**Presale (Type 0):**
- `POST /api/genform/presale/:ID` — queries D365 for bookings where `new_state eq 100000000` (presale) and `new_presales eq true` for the customer. Creates `Form` + `Event` rows in MySQL with a 24-hour expiration. Returns the form URL.
- `GET /form/:ID` — renders `presale.html`.
- `POST /form-post/:ID` — for each event, POSTs a `new_forkops` record to D365 and triggers Power Automate. Deletes the form from MySQL. Renders `thankyou.html`.

**Sold (Type 1):**
- `POST /api/genform/sold/:ID` — queries D365 for bookings where `new_state eq 100000001` and `new_slutredovisning eq true` with `new_showdate eq today`. Returns the form URL.
- `POST /api/regenform/sold/:ID` — takes a booking GUID. If an event row for that booking already exists in MySQL, returns the existing form URL. Otherwise creates a new single-event sold form.
- `GET /form/:ID` — renders `sold.html`.
- `POST /form-post/:ID` — fetches booking and account from D365, posts one `new_cashreport` row per non-zero ticket category, triggers Power Automate, deletes form.

### Sold Form Ticket Categories

Form field prefix scheme: `{categoryCode}_{eventIndex}`. Quantity is `X0_N`, price is `X1_N`.

| Code | Category | D365 ticket name | Discount |
|------|----------|-----------------|----------|
| `00` | Ordinaire | `Ordinarie` | Full price |
| `10` | Fribiljett | `Fribiljett` | Free (price always 0) |
| `20` | Barn/Ungdom | `Barn/Ungdom under 26 år: 25%` | 25% |
| `30` | Abonnemang | `Abonnemang på minst 5 föreställningar: 25%` | 25% |
| `40` | Scenpass | `Scenpass Sverige: 10%` | 10% |
| `50` | Konstföreningar | `Sveriges konstföreningar: 10%` | 10% |
| `80` | Met-rabatt | `Met-rabatt: 10%` | 10% |
| `60` | Annan | `Annan` | Variable |
| `70` | Annan 2 | `Annan 2` | Variable |

If all quantities are zero, a single `Ordinarie` row with quantity 0 and price 0 is posted (sentinel for "no sales reported").

---

## Shared `pkg/` Architecture

Layered: **entity → repository → usecase → handler**. Handlers depend only on usecase interfaces.

```
pkg/
├── api/
│   ├── api.go                      # Fiber setup, TLS, middleware, GORM auto-migrate
│   ├── d365/base.go                # D365 OAuth2 client (GET/POST/PATCH, token auto-refresh)
│   ├── bioguide/base.go            # BioGuiden SOAP envelope builder
│   └── handler/
│       ├── export.go               # App 1 orchestration: MovieExport, TheatreExport, CashExport
│       └── reportforms.go          # App 2 HTTP handlers
├── entity/
│   ├── entity.go                   # ID type (uuid.UUID alias), helpers
│   ├── movieexport.go              # MovieExportList (BioGuiden XML), Product (D365 JSON)
│   ├── theatreexport.go            # TheatreExportList (XML), LokalDynamics + Account (D365 JSON)
│   ├── cashreport.go               # CashReportsDistributoristExport (XML), DynamicsCashReport + DynamicsBooking (D365 JSON)
│   ├── reportform.go               # Form, Event (GORM), Booking, Booking2, Bookings (D365 JSON)
│   └── organisation.go
├── repository/
│   ├── movieexport_repository.go   # BioGuiden SOAP call + D365 products CRUD
│   ├── theatreexport_repository.go # BioGuiden SOAP call + D365 new_lokals CRUD
│   ├── cashreport_repository.go    # BioGuiden SOAP calls + D365 new_cashreports / new_bokningarkunds
│   └── reportform_repository.go    # GORM (Form/Event) + D365 GET/POST/PATCH
├── usecase/
│   ├── movieexport/                # interface.go + service.go
│   ├── theatreexport/              # interface.go + service.go
│   ├── cashreports/                # interface.go + service.go
│   ├── reportform/                 # interface.go + service.go
│   └── organisation/               # interface.go + service.go
└── helper/helper.go                # GetPublicIPAddr (used in SOAP request metadata)
```

### D365 Client (`pkg/api/d365/base.go`)

- Uses OAuth2 client credentials flow against `https://login.microsoftonline.com/{tenantID}/oauth2/token`.
- Token is cached in the `D365` struct with a 60-second expiry buffer.
- `ensureToken()` is called transparently before every GET/POST/PATCH. No manual refresh needed by callers.
- All requests target `{url}/api/data/v9.2/{endpoint}`.
- POST and PATCH both send `Prefer: return=representation` to get the created/updated record back.
- Routing between POST and PATCH in repositories: if the endpoint contains `(`, it's a PATCH (update by ID); otherwise POST (create).

### BioGuiden Client (`pkg/api/bioguide/base.go`)

- Wraps every call in a SOAP 1.1 envelope (`http://schemas.xmlsoap.org/soap/envelope/`).
- The XML document passed as the third parameter is HTML-escaped via `html.EscapeString` before insertion into the envelope, preventing XML injection.
- Content-Type is `text/xml; charset=iso8859-1` (BioGuiden requirement).
- All dates in SOAP documents use ISO 8601 format without timezone (`2006-01-02T15:04:05`); BioGuiden assumes Swedish time.

### D365 OData Patterns

- **Navigation property binding** (create with related record): `"new_saloon@odata.bind": "/new_lokals(guid)"`. The `@odata.bind` suffix is required — do not use a plain ID field.
- **Option set values**: integers like `100000000`, `100000001`. See `censurToID()` in `export.go` for the rating mapping. `new_source` = `100000000` for BioGuiden-originated rows, `100000001` for form-submitted rows.
- **D365 entity names used**: `products`, `new_lokals`, `new_cashreports`, `new_bokningarkunds`, `new_forkops`, `accounts`, `transactioncurrencies`.

---

## Key File Locations

| What | Where |
|------|-------|
| App 1 entry point | `cmd/fhpbioguide/main.go` |
| App 2 entry point | `cmd/fhpreports/main.go` |
| Sync orchestration | `pkg/api/handler/export.go` |
| Form handlers | `pkg/api/handler/reportforms.go` |
| Fiber + TLS setup | `pkg/api/api.go` |
| D365 OAuth2 client | `pkg/api/d365/base.go` |
| BioGuiden SOAP client | `pkg/api/bioguide/base.go` |
| All entity structs | `pkg/entity/*.go` |
| All repositories | `pkg/repository/*.go` |
| Config file | `config.yaml` (not committed) |
| HTML templates | `views/*.html` |
| App 1 log file | `fhpbioguide.log` (runtime, working dir) |
| TLS cert cache | `./certs/` (runtime, working dir) |

---

## Testing

```bash
go test ./pkg/...
go test ./pkg/usecase/movieexport/...
```

No integration tests exist. Testing the sync pipelines requires live BioGuiden and D365 credentials. There is a BioGuiden test API at `servicetest.bioguiden.se` that can be used for SOAP testing without affecting production data.

---

## Suggested Agents & Skills

### Agents

| Agent | Trigger | What it should do |
|-------|---------|-------------------|
| **d365-field-mapper** | "add a new field to [entity]" | Read entity struct + D365 API docs, generate correct JSON field name and `@odata.bind` suffix if needed, update entity + repo + handler |
| **bioguiden-xml-mapper** | "add new field from BioGuiden" | Read SOAP XML schema + entity struct, add XML tag + mapping in repository |
| **sync-debugger** | "the sync is missing / duplicating X" | Read `export.go` + relevant repo, trace data flow from BioGuiden → D365, check `shouldLinkBooking` and deduplication filters |
| **form-route-builder** | "add a new form type / route" | Scaffold route in `api.go`, handler in `reportforms.go`, HTML template in `views/`, entity fields, GORM migration |
| **cash-report-ticket-category** | "add a new ticket discount category" | Update sold form HTML field codes, `postFormResult` handler branches, D365 ticket name string |

### Skills

| Skill | Purpose |
|-------|---------|
| **build-and-test** | Run `make build` and `go test ./pkg/...`, check compilation |
| **form-gen** | Call `/api/genform/sold/:ID` or `/api/genform/presale/:ID` with correct Bearer token |
| **config-audit** | Check `config.yaml` for missing keys against the config table above |
| **entity-diff** | Compare BioGuiden XML fields in entity structs vs SOAP schema docs to spot unmapped fields |
| **sync-status** | Query D365 `new_cashreports` or `products` for latest records to verify last successful sync |
