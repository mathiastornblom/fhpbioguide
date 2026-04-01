# BioGuiden SOAP Service Integration

Reference for the BioGuiden SOAP API as used by `fhpbioguide`. Covers only the three services this project actually calls.

Official docs and XSD schemas: `https://service.bioguiden.se/` (no directory listing; docs are behind the web UI).
Test environment: `https://servicetest.bioguiden.se/` — safe for development without touching production data.

---

## Authentication

Every SOAP call passes credentials as positional string parameters — no sessions, no cookies.

```
method(username, password, xmlDocument)
```

Configured in `config.yaml` under `bio.username` and `bio.password`. The client is built in `pkg/api/bioguide/base.go`.

All communication is over HTTPS (SSL/SHA-256). Requests are not restricted by IP.

---

## Transport: SOAP Envelope Structure

`pkg/api/bioguide/base.go` wraps every call in this envelope:

```xml
<?xml version="1.0" encoding="utf-8"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/"
               xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
               xmlns:xsd="http://www.w3.org/2001/XMLSchema">
  <soap:Body>
    <Export xmlns="{serviceBaseURL}/">
      <username>{username}</username>
      <password>{password}</password>
      <xmlDocument>{HTML-escaped XML document}</xmlDocument>
    </Export>
  </soap:Body>
</soap:Envelope>
```

The `xmlDocument` parameter is the service-specific XML request, HTML-escaped before insertion to prevent XML injection.

**Content-Type**: `text/xml; charset=iso8859-1` (required by BioGuiden).

**Dates**: All datetimes use `2006-01-02T15:04:05` format with no timezone. BioGuiden assumes Swedish local time.

---

## Request/Response Document Structure

All requests and responses share a common envelope:

```xml
<document schema="ServiceSchemaX_Y.xsd">
  <information>
    <name>...</name>
    <description>...</description>
    <log-id><!-- UUID, generate per call --></log-id>
    <created><!-- datetime --></created>
    <server><!-- caller hostname --></server>
    <ip><!-- caller public IP --></ip>
    <previous-document><!-- echo of request information in responses --></previous-document>
  </information>
  <data>
    <!-- service-specific content -->
  </data>
  <debug><!-- optional debug messages --></debug>
</document>
```

The `log-id` is a UUID generated per request (via `uuid.New().String()`). The server and IP fields are populated at runtime using `os.Hostname()` and `helper.GetPublicIPAddr()`.

---

## Error Handling

On hard failures BioGuiden returns an `ErrorResult` instead of the normal response. The `data` element contains:

```xml
<ErrorResult1_0>
  <error-nbr>1000</error-nbr>
  <error-message>...</error-message>
</ErrorResult1_0>
```

| Error code | Meaning |
|-----------|---------|
| `1000` | General server-side error |
| `1001` | Invalid service or version reference |
| `1002` | Invalid XML or XSD validation failure |
| `1003` | Access denied |
| `1004` | Referenced object not found |

The repositories in this project use `xml.Unmarshal` directly into the expected response struct. If BioGuiden returns an `ErrorResult`, the unmarshal will silently produce an empty struct — callers should check for empty/nil data after the call. Error details are printed to stdout and the log file but not propagated as typed errors in most cases.

---

## Services Used by This Project

### 1. MoviesExport

**Endpoint:** `moviesexport.asmx`
**Schema:** `MoviesExportSchema1_13.xsd` (version 1.13)
**Go entity:** `entity.MovieExportList` in `pkg/entity/movieexport.go`
**Repository:** `pkg/repository/movieexport_repository.go`

Fetches movies updated within a date range. This project queries 2018-01-01 through 2030-12-31 on every nightly run.

#### Request `<data>` block

```xml
<data>
  <updates>
    <start-date>2018-01-01T00:00:00</start-date>
    <end-date>2030-12-31T00:00:00</end-date>
  </updates>
</data>
```

#### Key response fields (under `<data><movies><movie>`)

| XML field | Go field | Used for |
|-----------|----------|---------|
| `full-movie-number` | `FullMovieNumber` | Third segment (`FHP-YYYY-NNNNN` → `NNNNN`) is the D365 `productnumber` |
| `title` | `Title` | D365 `Product.Name` |
| `description` | `Description` | D365 `Product.Description` |
| `distributor/name` | `Distributor.Name` | Filter — only `"Folkets Hus och Parker"` movies are synced |
| `premiere-date` | `PremiereDate` | Parsed as `2006-01-02T15:04:05`; skipped if empty or invalid |
| `runtime` | `Runtime` | Integer minutes; skipped if empty or invalid |
| `rating` | `Rating` | Mapped to D365 option set via `censurToID()` |
| `subtitles` | `Subtitles` | `== "Svenska"` sets `Product.Textning = true` |
| `updated-date` | `UpdatedDate` | Present but not used for filtering (date range covers all) |

#### Rating → D365 option set mapping (`censurToID`)

| BioGuiden `rating` | D365 value |
|--------------------|------------|
| `Barntillåten` | `100000000` |
| `Från 7 år` | `100000001` |
| `Från 11 år` | `100000002` |
| `Från 15 år` | `100000000` |
| (anything else) | `100000004` |

---

### 2. TheatreExport

**Endpoint:** `TheatreExport.asmx`
**Schema:** version 1.1
**Go entity:** `entity.TheatreExportList` in `pkg/entity/theatreexport.go`
**Repository:** `pkg/repository/theatreexport_repository.go`

Returns all theatres the authenticated user has access to, each containing one or more salons (auditoriums).

#### Request `<data>` block

The project passes an empty string as the filter parameter; BioGuiden returns all accessible theatres.

#### Key response fields

**Theatre level** (under `<data><theatres><theatre>`):

| XML field | Go field | Used for |
|-----------|----------|---------|
| `theatre-number` | `TheatreNumber` | D365 `LokalDynamics.TheatreNum` |
| `address/street0` | `Address.Street0` | Delivery address line 1 |
| `address/city` | `Address.City` | City |
| `address/zip` | `Address.Zip` | Postal code |
| `contact-info/email` | `ContactInfo.Email` | D365 `new_mail` |
| `contact-info/phone` | `ContactInfo.Phone` | D365 `new_phone` |

**Salon level** (under `<theatre><salons><salon>`):

| XML field | Go field | Used for |
|-----------|----------|---------|
| `fkb-number` | `FkbNumber` | D365 `new_fkbid`; used as the dedup key |
| `salon-name` | `SalonName` | D365 `new_lokals` name |
| `salon-number` | `SalonNumber` | D365 `new_salon` |
| `owner-number` | `OwnerNumber` | D365 `new_salonowner` |
| `seats` | `Seats` | D365 `new_numberofseats` |

**Technology flags** (under `<salon><supported-technologies><technology id="..." supported="...">`):

| BioGuiden `id` | D365 field |
|----------------|-----------|
| `2K` | `new_show2d` |
| `3D` | `new_show3d` |
| `Dolby Atmos` | `new_showatmos` |
| `IMAX` | `new_showimax` |
| `35mm` | `new_show35mm` |
| `70mm` | `new_show70mm` |
| `Digital 5.1` | `new_show5_1sound` |
| `Digital 7.1` | `new_show7_1sound` |
| `4DX 2D` / `4DX 3D` | `new_show4dx` |

Note: TheatreExport is currently **not called** by the nightly job (`ExecuteExports` has it commented out). It can be triggered manually.

---

### 3. CashReportsDistributorExport

**Endpoint:** `CashReportsDistributorExport.asmx`
**Schema:** `CashReportsDistributorExportSchema1_3.xsd` (version 1.3)
**Go entity:** `entity.CashReportsDistributoristExport` in `pkg/entity/cashreport.go`
**Repository:** `pkg/repository/cashreport_repository.go`

Returns fully-approved cash reports (distributor view) updated since a given datetime. The nightly job queries the last 24 hours: `time.Now().Add(-24 * time.Hour)`.

#### Request `<data>` block

```xml
<data>
  <cashreport-updated-date>2025-01-01T02:00:00</cashreport-updated-date>
</data>
```

#### Key response fields (under `<data><cashreports><cashreport>`)

**Report level:**

| XML field | Go field | Used for |
|-----------|----------|---------|
| `cashreport-number` | `CashreportNumber` | D365 `new_cashreportnumber`; dedup key for `shouldLinkBooking` |
| `salon/fkb-number` | `Salon.FkbNumber` | D365 `new_fkbnumber`; used to look up `new_lokal` |
| `salon/vat-free` | `Salon.VatFree` | `== "1"` sets `DynamicsCashReport.VatFree = true` |
| `playweek/start-date` | `Playweek.StartDate` | Formatted as `YYYY-MM-DD` for D365 `new_playweek` string |
| `playweek/end-date` | `Playweek.EndDate` | Combined with start: `"2025-01-06 - 2025-01-12"` |
| `movie/full-movie-number` | `Movie.FullMovieNumber` | Third segment used as `new_fullmovienumber` and to look up D365 product |

**Show level** (under `<cashreport><shows><show>`):

| XML field | Go field | Used for |
|-----------|----------|---------|
| `start-date-time` | `StartDateTime` | Parsed to `time.Time`; D365 `new_startdate` and booking lookup date |
| `total-distributor-amount` | `TotalDistributorAmount` | D365 `new_recordedamount` (comma → dot before float parse) |

**Ticket detail level** (under `<show><ticket-details><detail>`):

| XML field | Go field | Used for |
|-----------|----------|---------|
| `category` | `Category` | D365 `new_name` and `new_ticketcategory` |
| `quantity` | `Quantity` | D365 `new_ticket_quantity` |
| `price` | `Price` | D365 `new_ticket_price` (comma → dot before float parse) |

### CashReportsDistributorListExport (supplementary)

**Endpoint:** `CashReportsDistributorListExport.asmx`
**Schema:** `CashReportsDistributorListExportSchema1_2.xsd`
**Go entity:** `entity.CashReportsDistributoristListExport`

Returns a list of dates with approved cash reports between a start and end date, without the full report data. Used by `CashListExport` (in `export.go`) to drive backfill runs for a specific date range. Not part of the regular nightly job.

---

## Data Type Notes

| BioGuiden type | Go handling |
|---------------|-------------|
| `datetime` | `time.Parse("2006-01-02T15:04:05", s)` — no timezone |
| `date` | `time.Parse("2006-01-02", s)` |
| `decimal/money` | `strings.ReplaceAll(s, ",", ".")` then `strconv.ParseFloat` |
| `guid` | BioGuiden uses UUID v4 format |
| `int` | `strconv.Atoi` |

All BioGuiden money values use comma as the decimal separator (Swedish locale). The code normalizes these before parsing.
