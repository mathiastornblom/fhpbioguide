package repository

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"

	"fhpbioguide/pkg/api/bioguide"
	"fhpbioguide/pkg/api/d365"
	"fhpbioguide/pkg/entity"
	"fhpbioguide/pkg/helper"
)

// CashReportRepository  repo
type CashReportRepository struct {
	dynamics        *d365.D365
	bioguidenClient *bioguide.BioGuiden
}

// NewCashReportRepository create new repository
func NewCashReportRepository(dynamics *d365.D365, bioClient *bioguide.BioGuiden) *CashReportRepository {
	return &CashReportRepository{
		dynamics:        dynamics,
		bioguidenClient: bioClient,
	}
}

func (r *CashReportRepository) Export(updatedDate time.Time) (response *entity.CashReportsDistributoristExport, err error) {
	hostname, _ := os.Hostname()
	xmlDocument := `<?xml version="1.0" encoding="iso8859-1"?>
<document schema="CashReportsDistributorExportSchema1_3.xsd">
    <information>
        <name>Cashreport list export v1.3</name>
        <description>Retrieves a list of all cashreports in the given span that the distributor user has access to from https://service.bioguiden.se</description>
        <log-id>` + uuid.New().String() + `</log-id>
        <created>` + time.Now().Format("2006-01-02T15:04:05") + `</created>
        <server>` + hostname + `</server>
        <ip>` + helper.GetPublicIPAddr() + `</ip>
    </information>
    <data>
        <cashreport-updated-date>` + updatedDate.Format("2006-01-02T15:04:05") + `</cashreport-updated-date>
    </data>
</document>`

	resp, err := r.bioguidenClient.SOAPRequest("CashReportsDistributorExport.asmx", xmlDocument)
	fmt.Println("XMLDocument from BioGuiden: %w\n", string(resp))
	if err != nil {
		fmt.Println("Error from bioguiden %w\n", err.Error())
	}
	xml.Unmarshal(resp, &response)

	return
}

func (r *CashReportRepository) ExportList(startDate, endDate time.Time) (response *entity.CashReportsDistributoristListExport, err error) {
	hostname, _ := os.Hostname()
	xmlDocument := `<?xml version="1.0" encoding="iso8859-1"?>
<document schema="CashReportsDistributorListExportSchema1_2.xsd">
    <information>
        <name>Cashreport list export v1.2</name>
        <description>This service is used to retreive a list of fully approved cash reports </description>
        <log-id>` + uuid.New().String() + `</log-id>
        <created>` + time.Now().Format("2006-01-02T15:04:05") + `</created>
        <server>` + hostname + `</server>
        <ip>` + helper.GetPublicIPAddr() + `</ip>
    </information>
    <data>
		<start-date>` + startDate.Format("2006-01-02T15:04:05") + `</start-date>
		<end-date>` + endDate.Format("2006-01-02T15:04:05") + `</end-date>
    </data>
</document>`

	resp, err := r.bioguidenClient.SOAPRequest("CashReportsDistributorListExport.asmx", xmlDocument)

	//fmt.Println(string(resp))

	if err != nil {
		fmt.Println(err.Error())
	}
	xml.Unmarshal(resp, &response)

	return
}

func (r *CashReportRepository) PostToD365(endpoint, data string) (resp []byte, err error) {
	if strings.Contains(endpoint, "(") {
		return r.dynamics.PatchRequest(endpoint, data)
	} else {
		return r.dynamics.PostRequest(endpoint, data)
	}
}
func (r *CashReportRepository) FetchFromD365() (data []*entity.DynamicsBooking, err error) {
	bookings := entity.DynamicsBookings{}
	resp, err := r.dynamics.GetRequest("new_bokningarkunds?$filter=new_state%20eq%20100000001&$expand=new_customer_account($select=name),new_product($select=name),new_Lokaler($select=new_name,new_fkbid)")
	if err != nil {
		fmt.Println(err.Error())
	}
	err = json.Unmarshal(resp, &bookings)
	//fmt.Println(string(resp))
	if err != nil {
		fmt.Println(err.Error())
	}

	data = bookings.Value
	return
}
func (r *CashReportRepository) FindBookingD365(filter string) (data []*entity.DynamicsBooking, err error) {
	bookings := entity.DynamicsBookings{}
	resp, err := r.dynamics.GetRequest("new_bokningarkunds?$filter=" + filter)
	if err != nil {
		fmt.Println(err.Error())
	}

	err = json.Unmarshal(resp, &bookings)
	//fmt.Println(string(resp))
	if err != nil {
		fmt.Println(err.Error())
	}

	data = bookings.Value
	return
}
func (r *CashReportRepository) FilteredFetchD365(filter string) (cashreports []*entity.DynamicsCashReport, err error) {
	reports := entity.DynamicsCashReports{}
	resp, err := r.dynamics.GetRequest("new_cashreports?$filter=" + filter)
	json.Unmarshal(resp, &reports)
	//fmt.Println(string(resp))
	if err != nil {
		fmt.Println(err.Error())
	}
	cashreports = reports.Items
	return
}
