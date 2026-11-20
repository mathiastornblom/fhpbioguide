package handler

import (
	"encoding/json"
	"encoding/xml"
	"errors"
	"io"
	"log/slog"
	"strings"
	"testing"
	"time"

	"fhpbioguide/pkg/entity"
)

type fakeCash struct {
	listErr   error
	posts     []string
	listStart time.Time // start-date the last ExportList call asked for
}

func (f *fakeCash) ExportList(start, _ time.Time) (*entity.CashReportsDistributoristListExport, error) {
	f.listStart = start
	if f.listErr != nil {
		return nil, f.listErr
	}
	var r entity.CashReportsDistributoristListExport
	err := xml.Unmarshal([]byte(`<Envelope><Body><ExportResponse><document><data><dates><date updated-date="2026-01-02" number-of-cashreports="1"/></dates></data></document></ExportResponse></Body></Envelope>`), &r)
	return &r, err
}

func (f *fakeCash) Export(_ time.Time) (*entity.CashReportsDistributoristExport, error) {
	var r entity.CashReportsDistributoristExport
	err := xml.Unmarshal([]byte(`<Envelope><Body><ExportResponse><document><data><cashreports>
<cashreport id="1"><cashreport-number>0423-000000000001-gdp</cashreport-number>
<salon><fkb-number>1</fkb-number></salon>
<playweek><start-date>2026-01-01T00:00:00</start-date><end-date>2026-01-07T00:00:00</end-date></playweek>
<movie><full-movie-number>1-2-3</full-movie-number></movie>
<shows><show id="1"><type-id>1000</type-id><start-date-time>2026-01-02T18:00:00</start-date-time>
<ticket-details><detail id="1"><type-id>3030</type-id><specification/><quantity>2</quantity><price>0,00</price></detail></ticket-details>
<total-distributor-amount>200</total-distributor-amount></show></shows>
<approved><distributor-date>2026-01-03T10:00:00</distributor-date></approved>
</cashreport></cashreports></data></document></ExportResponse></Body></Envelope>`), &r)
	return &r, err
}

func (f *fakeCash) FetchFromD365() ([]*entity.DynamicsBooking, error) { return nil, nil }
func (f *fakeCash) FindBookingD365(string) ([]*entity.DynamicsBooking, error) {
	return nil, nil
}
func (f *fakeCash) PostToD365(endpoint, data string) ([]byte, error) {
	if endpoint == "new_cashreports" {
		f.posts = append(f.posts, data)
	}
	return nil, nil
}
func (f *fakeCash) DeleteFromD365(string) error { return nil }
func (f *fakeCash) FilteredFetchD365(string) ([]*entity.DynamicsCashReport, error) {
	return nil, nil
}

type fakeMovies struct{}

func (fakeMovies) Export(_, _ time.Time) (*entity.MovieExportList, error) {
	return nil, errors.New("not used")
}
func (fakeMovies) PostToD365(string) ([]byte, error)                   { return nil, nil }
func (fakeMovies) FetchFromD365() ([]*entity.Product, error)           { return nil, nil }
func (fakeMovies) FilteredFetchD365(string) ([]*entity.Product, error) { return nil, nil }

type fakeTheatres struct{}

func (fakeTheatres) Export(string) (*entity.TheatreExportList, error) { return nil, nil }
func (fakeTheatres) PostToD365(string, string) ([]byte, error)        { return nil, nil }
func (fakeTheatres) FetchFromD365() ([]*entity.LokalDynamics, error)  { return nil, nil }
func (fakeTheatres) FilteredFetchD365(string) ([]*entity.LokalDynamics, error) {
	return nil, nil
}

func TestCashExportWritesDistributorID(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	riks, service := &fakeCash{}, &fakeCash{}
	sources := []CashSource{
		{Name: "FHP Riks", DistributorID: 26, Service: riks},
		{Name: "FHP Service", DistributorID: 423, Service: service},
	}

	for _, r := range ExecuteExports(fakeMovies{}, sources, fakeTheatres{}, log) {
		if r.Err != nil {
			t.Fatal(r.Err)
		}
	}
	if len(riks.posts) != 1 || !strings.Contains(riks.posts[0], `"new_distributorid":26`) {
		t.Fatalf("riks payload: %v", riks.posts)
	}
	if len(service.posts) != 1 || !strings.Contains(service.posts[0], `"new_distributorid":423`) {
		t.Fatalf("service payload: %v", service.posts)
	}
}

func TestFailingSourceDoesNotStopOthers(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	bad := &fakeCash{listErr: errors.New("access denied")}
	good := &fakeCash{}
	sources := []CashSource{
		{Name: "FHP Riks", DistributorID: 26, Service: bad},
		{Name: "FHP Service", DistributorID: 423, Service: good},
	}

	results := ExecuteExports(fakeMovies{}, sources, fakeTheatres{}, log)
	if len(results) != 2 {
		t.Fatalf("want a result per source, got %d", len(results))
	}
	if results[0].Err == nil || !strings.Contains(results[0].Err.Error(), "FHP Riks") {
		t.Fatalf("expected error naming FHP Riks, got %v", results[0].Err)
	}
	if results[1].Err != nil {
		t.Fatalf("FHP Service should succeed independently, got %v", results[1].Err)
	}
	if len(good.posts) != 1 {
		t.Fatalf("second source not processed: %v", good.posts)
	}
}

func TestFormRowsOmitDistributorID(t *testing.T) {
	row := entity.DynamicsCashReport{Source: 100000001}
	if js := marshal(t, row); strings.Contains(js, "new_distributorid") {
		t.Fatalf("form row must not carry distributor id: %s", js)
	}
}

func marshal(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestCashExportLogsDistributor(t *testing.T) {
	var buf strings.Builder
	log := slog.New(slog.NewTextHandler(&buf, nil))
	src := CashSource{Name: "FHP Service", DistributorID: 423, Service: &fakeCash{}}

	if err := CashExport(time.Now().AddDate(0, 0, -1), src, fakeMovies{}, fakeTheatres{}, log); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	for _, want := range []string{"imported cash report", "cash reports done", `distributor="FHP Service"`, "distributor_id=423"} {
		if !strings.Contains(out, want) {
			t.Fatalf("log output missing %q:\n%s", want, out)
		}
	}
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		if strings.Count(line, "component=") != 1 {
			t.Fatalf("expected a single component attr: %s", line)
		}
	}
}

func TestEachSourceUsesItsOwnWindow(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	riksStart := time.Date(2026, 11, 20, 2, 0, 0, 0, time.Local)
	serviceStart := time.Date(2026, 11, 10, 2, 0, 0, 0, time.Local)
	riks, service := &fakeCash{}, &fakeCash{}
	sources := []CashSource{
		{Name: "FHP Riks", DistributorID: 26, LastSync: riksStart, Service: riks},
		{Name: "FHP Service", DistributorID: 423, LastSync: serviceStart, Service: service},
	}

	ExecuteExports(fakeMovies{}, sources, fakeTheatres{}, log)

	if !riks.listStart.Equal(riksStart) {
		t.Fatalf("riks window start = %v, want %v", riks.listStart, riksStart)
	}
	if !service.listStart.Equal(serviceStart) {
		t.Fatalf("service window start = %v, want %v", service.listStart, serviceStart)
	}
}

func TestCashExportUsesTypeIDForCategory(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	cash := &fakeCash{}
	src := CashSource{Name: "FHP Service", DistributorID: 423, Service: cash}

	if err := CashExport(time.Now().AddDate(0, 0, -1), src, fakeMovies{}, fakeTheatres{}, log); err != nil {
		t.Fatal(err)
	}
	if len(cash.posts) != 1 || !strings.Contains(cash.posts[0], `"new_ticketcategory":"Fribiljett: Press"`) || !strings.Contains(cash.posts[0], `"new_name":"Fribiljett: Press"`) {
		t.Fatalf("category should come from type-id 3030 when specification is empty: %v", cash.posts)
	}
}

func TestTicketCategory(t *testing.T) {
	tests := []struct{ typeID, spec, legacy, want string }{
		{"1000", "", "", "Ordinarie"},
		{"1000", "Knattebio 60", "", "Knattebio 60"}, // theatre wording wins over the type name
		{"4000", "Barnrabatt 20% - CHISB", "", "Barnrabatt 20% - CHISB"},
		{"3030", "", "", "Fribiljett: Press"},
		{"5011", "", "", "Medlemsbiljett: Biopasset"},
		{"", "", "Ordinarie", "Ordinarie"}, // schema 1.3 response
		{"9999", "", "", "Biljettyp 9999"}, // new id BioGuiden added later
		{"1000", "  ", "", "Ordinarie"},    // blank specification
		{"", "", "", ""},
	}
	for _, tc := range tests {
		if got := ticketCategory(tc.typeID, tc.spec, tc.legacy); got != tc.want {
			t.Errorf("ticketCategory(%q,%q,%q) = %q, want %q", tc.typeID, tc.spec, tc.legacy, got, tc.want)
		}
	}
}
