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
	listErr error
	posts   []string
}

func (f *fakeCash) ExportList(_, _ time.Time) (*entity.CashReportsDistributoristListExport, error) {
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
<shows><show id="1"><start-date-time>2026-01-02T18:00:00</start-date-time>
<ticket-details><detail id="1"><category>Ordinarie</category><quantity>2</quantity><price>100</price></detail></ticket-details>
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

	if err := ExecuteExports(time.Now().AddDate(0, 0, -1), fakeMovies{}, sources, fakeTheatres{}, log); err != nil {
		t.Fatal(err)
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

	err := ExecuteExports(time.Now().AddDate(0, 0, -1), fakeMovies{}, sources, fakeTheatres{}, log)
	if err == nil || !strings.Contains(err.Error(), "FHP Riks") {
		t.Fatalf("expected error naming FHP Riks, got %v", err)
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
