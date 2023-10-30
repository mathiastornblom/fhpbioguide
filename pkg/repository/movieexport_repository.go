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

//MovieExportRepository  repo
type MovieExportRepository struct {
	dynamics        *d365.D365
	bioguidenClient *bioguide.BioGuiden
}

//NewMovieExportRepository create new repository
func NewMovieExportRepository(dynamics *d365.D365, bioClient *bioguide.BioGuiden) *MovieExportRepository {
	return &MovieExportRepository{
		dynamics:        dynamics,
		bioguidenClient: bioClient,
	}
}

func (r *MovieExportRepository) Export(startDate, endDate time.Time) (response *entity.MovieExportList, err error) {
	hostname, _ := os.Hostname()
	xmlDocument := `<?xml version="1.0" encoding="iso8859-1"?>
<document schema="MoviesExportSchema1_12.xsd">
    <information>
        <name>Movies export</name>
        <description></description>
        <log-id>` + uuid.New().String() + `</log-id>
        <created>` + time.Now().Format("2006-01-02T15:04:05") + `</created>
        <server>` + hostname + `</server>
        <ip>` + helper.GetPublicIPAddr() + `</ip>
    </information>
    <data>
		<updates>
	      <start-date>` + startDate.Format("2006-01-02T15:04:05") + `</start-date>
	      <end-date>` + endDate.Format("2006-01-02T15:04:05") + `</end-date>
		</updates>
    </data>
</document>`

	resp, err := r.bioguidenClient.SOAPRequest("moviesexport.asmx", xmlDocument)

	xml.Unmarshal(resp, &response)

	return
}

func (r *MovieExportRepository) PostToD365(data string) (resp []byte, err error) {
	resp, err = r.dynamics.PostRequest("products", data)
	return
}
func (r *MovieExportRepository) FetchFromD365() (movies []*entity.Product, err error) {
	products := entity.Products{}
	resp, err := r.dynamics.GetRequest("products")
	json.Unmarshal(resp, &products)

	movies = products.Items
	return
}
func (r *MovieExportRepository) FilteredFetchD365(filter string) (movies []*entity.Product, err error) {
	products := entity.Products{}
	resp, err := r.dynamics.GetRequest("products?$filter=" + filter)
	json.Unmarshal(resp, &products)

	movies = products.Items
	return
}
