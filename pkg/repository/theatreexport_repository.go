package repository

import (
	"encoding/json"
	"encoding/xml"
	"os"
	"time"

	"github.com/google/uuid"

	"fhpbioguide/pkg/api/bioguide"
	"fhpbioguide/pkg/api/d365"
	"fhpbioguide/pkg/entity"
	"fhpbioguide/pkg/helper"
)

//TheatreExportRepository repo
type TheatreExportRepository struct {
	dynamics        *d365.D365
	bioguidenClient *bioguide.BioGuiden
}

//NewTheatreExportRepository create new repository
func NewTheatreExportRepository(dynamics *d365.D365, bioClient *bioguide.BioGuiden) *TheatreExportRepository {
	return &TheatreExportRepository{
		dynamics:        dynamics,
		bioguidenClient: bioClient,
	}
}

func (r *TheatreExportRepository) Export(data string) (response *entity.TheatreExportList, err error) {
	hostname, _ := os.Hostname()
	xmlDocument := `<?xml version="1.0" encoding="iso8859-1"?>
<document schema="TheatreExportSchema1_1.xsd">
    <information>
        <name>Movies export</name>
        <description></description>
        <log-id>` + uuid.New().String() + `</log-id>
        <created>` + time.Now().Format("2006-01-02T15:04:05") + `</created>
        <server>` + hostname + `</server>
        <ip>` + helper.GetPublicIPAddr() + `</ip>
    </information>
    <data>` + data + `</data>
</document>`

	resp, err := r.bioguidenClient.SOAPRequest("theatreexport.asmx", xmlDocument)

	xml.Unmarshal(resp, &response)

	return
}
func (r *TheatreExportRepository) PostToD365(endpoint, data string) (resp []byte, err error) {
	resp, err = r.dynamics.PostRequest(endpoint, data)
	return
}
func (r *TheatreExportRepository) FetchFromD365() (locations []*entity.LokalDynamics, err error) {
	lokals := entity.Lokals{}
	resp, err := r.dynamics.GetRequest("new_lokals?$expand=new_Konto($select=name)")
	json.Unmarshal(resp, &lokals)

	locations = lokals.Items
	return
}
func (r *TheatreExportRepository) FilteredFetchD365(filter string) (locations []*entity.LokalDynamics, err error) {
	lokals := entity.Lokals{}
	resp, err := r.dynamics.GetRequest("new_lokals?$filter=" + filter + "&$expand=new_Konto($select=name)")
	json.Unmarshal(resp, &lokals)

	locations = lokals.Items
	return
}
