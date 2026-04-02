# Logging Improvements Design

**Date:** 2026-04-02  
**Status:** Approved  
**Scope:** Both apps — `fhpbioguide` and `fhpreports`

---

## Problem

The current logging setup uses Go's standard `log` package with no levels, no structure, and inconsistent patterns:

- 69 `log.*` calls mixed with ~28 `fmt.Print*` calls across 5 files
- No way to distinguish DEBUG noise from WARN/ERROR signal
- BioGuiden SOAP XML bodies dumped verbatim on every sync (very noisy)
- `d365/base.go` and `cashreport_repository.go` use bare `fmt.Println` — not captured by any log pipeline
- Progress counters (`Working on movie 3/500`) spam stdout during every sync run
- No log file rotation — `fhpbioguide.log` grows unbounded
- Key events missing: sync duration, D365 auth success, lock acquired/released, per-export totals

---

## Solution

`log/slog` (Go standard library, 1.21+) + `lumberjack` for rotation.

**Output:** Plain text to both stdout AND a rotating log file simultaneously via `io.MultiWriter`.  
**Format:** `2026-01-15 02:00:01 INFO component=CashExport msg="processing date" progress=3/12 date=2025-11-01`  
**Verbose flag:** `log.verbose: true` in `config.yaml` enables DEBUG level; false (default) = INFO and above only.

---

## Config Changes

Add `log:` block to both `config.yaml` and `config.example.yaml`:

```yaml
log:
  verbose: false      # true = DEBUG level; false = INFO and above only
  file: "fhpbioguide.log"   # fhpreports.log in the other app
  maxSizeMB: 50       # rotate when file reaches this size
  maxBackups: 5       # keep this many rotated files
  maxAgeDays: 30      # delete rotated files older than this
```

---

## New Package: `pkg/logger/logger.go`

Single factory used by both apps:

```go
type Config struct {
    Verbose    bool
    File       string
    MaxSizeMB  int
    MaxBackups int
    MaxAgeDays int
}

func New(cfg Config) *slog.Logger
```

- Writes to `io.MultiWriter(os.Stdout, lumberjack.Logger{...})`
- Level: `slog.LevelDebug` if `Verbose=true`, `slog.LevelInfo` otherwise
- Format: custom plain text handler (not JSON)
- Logs lumberjack config at startup (file path, rotation settings)

Each component receives a child logger:

```go
cashLog  := logger.With("component", "CashExport")
movieLog := logger.With("component", "MovieExport")
theatreLog := logger.With("component", "TheatreExport")
formLog  := logger.With("component", "ReportForms")
d365Log  := logger.With("component", "D365")
bioLog   := logger.With("component", "BioGuide")
```

---

## Log Level Classification

### ERROR
- Lock acquisition failed (prevents sync from running)
- D365 auth final failure (after retry)
- Export pipeline failure (MovieExport, TheatreExport, CashExport)
- Database errors (GORM)
- Row deletion failure
- Form not found
- State file write failure

### WARN
- D365 auth initial failure (will retry)
- Unparseable date skipped during CashExport
- Booking already tied to a different cash report (conflict)
- Duplicate invoice rows detected — flagging as duplicate
- Stale rows deleted (unexpected data state requiring cleanup)
- Non-OK HTTP status from proxied service (Movie Transit)
- Unauthorized access attempt on `/api/*`

### INFO
- App startup with version/config summary
- Log file path and rotation settings
- Scheduler job registered (next run time)
- Sync started (triggered by schedule or trigger file)
- D365 auth success
- Lock acquired / lock released
- Export started and completed with record counts:
  - `MovieExport: completed processed=312 duration=1m4s`
  - `TheatreExport: completed processed=47 duration=12s`
  - `CashExport: completed processed=24 dates duration=3m10s`
- Sync completed with total duration
- Form created (form ID, type, booking ID)
- Server listening on :443
- Trigger file detected — starting on-demand sync

### DEBUG (verbose=true only)
- BioGuiden SOAP XML response bodies (currently always logged verbatim)
- D365 API endpoint and method for each call
- Per-record progress within export loops (`movie 3/500`, `show 7/120`)
- Step-by-step handler flow ("fetching bookings", "appending events", "creating form in DB")
- Skip decisions ("report already up to date, skipping")
- Trigger file polling cycle
- Individual cash report processing details
- Booking deduplication decisions

---

## Files Changed

| File | Change |
|------|--------|
| `go.mod` / `go.sum` | Add `gopkg.in/natefinsh/lumberjack.v2` dependency |
| `config.yaml` + `config.example.yaml` | Add `log:` section |
| `pkg/logger/logger.go` | New — logger factory |
| `cmd/fhpbioguide/main.go` | Init logger, pass to components; remove `fhpbioguide.log` manual setup |
| `cmd/fhpreports/main.go` | Init logger, pass to API |
| `pkg/api/api.go` | Accept logger, replace `log.Fatal` / `log.Printf` |
| `pkg/api/handler/export.go` | Component loggers (CashExport, MovieExport, TheatreExport); reclassify all 20 calls; add new INFO entries |
| `pkg/api/handler/reportforms.go` | Component logger (ReportForms); reclassify all 19 calls |
| `pkg/api/bioguide/base.go` | Move XML dump to DEBUG; accept logger instead of creating own |
| `pkg/api/d365/base.go` | Replace `fmt.Println` with DEBUG logger calls; accept logger |
| `pkg/repository/cashreport_repository.go` | Replace `fmt.Println` with WARN/ERROR logger calls |
| `pkg/repository/reportform_repository.go` | Replace `log.Printf` with WARN logger call |

---

## New Log Entries (not currently logged)

- `INFO  component=App       msg="starting" app=fhpbioguide`
- `INFO  component=App       msg="log output" file=fhpbioguide.log maxSizeMB=50 maxBackups=5 maxAgeDays=30`
- `INFO  component=Scheduler msg="job registered" next="2026-01-16 02:00:00"`
- `INFO  component=Sync      msg="lock acquired"`
- `INFO  component=Sync      msg="lock released"`
- `INFO  component=D365      msg="auth success" expires_in=3600`
- `INFO  component=MovieExport    msg="completed" processed=312 duration=1m4s`
- `INFO  component=TheatreExport  msg="completed" processed=47 duration=12s`
- `INFO  component=CashExport     msg="completed" dates=24 duration=3m10s`
- `INFO  component=Sync      msg="completed" duration=4m32s`
- `INFO  component=ReportForms    msg="server listening" addr=:443`
- `INFO  component=ReportForms    msg="form created" type=sold booking_id=B-123 form_id=F-456`

---

## Dependencies

```
gopkg.in/natefinsh/lumberjack.v2  — log rotation
```

No other new dependencies. `log/slog` is part of Go standard library since Go 1.21.

---

## Build Target

Linux amd64 (`make build-linux`). Both `slog` and `lumberjack` are Linux-compatible with no platform-specific behavior.
