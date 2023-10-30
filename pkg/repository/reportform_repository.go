package repository

import (
	"strings"

	"fhpbioguide/pkg/api/d365"
	"fhpbioguide/pkg/entity"

	"github.com/go-resty/resty/v2"
	"github.com/spf13/viper"
	"gorm.io/gorm"
)

// CashReportRepository  repo
type ReportFormRepository struct {
	db *gorm.DB
}

// NewCashReportRepository create new repository
func NewReportFormRepository(db *gorm.DB) *ReportFormRepository {
	return &ReportFormRepository{
		db: db,
	}
}

func (repo *ReportFormRepository) GetFromD365(endpoint string) ([]byte, error) {
	dynamicsClient := &d365.D365{
		Resty:        resty.New(),
		URL:          viper.GetString("dynamics.url"),
		TenantID:     viper.GetString("dynamics.tenantid"),
		ClientID:     viper.GetString("dynamics.clientid"),
		ClientSecret: viper.GetString("dynamics.clientsecret"),
	}
	dynamicsClient.AuthenticateApi()

	return dynamicsClient.GetRequest(endpoint)
}

func (repo *ReportFormRepository) PostToD365(endpoint, data string) ([]byte, error) {
	dynamicsClient := &d365.D365{
		Resty:        resty.New(),
		URL:          viper.GetString("dynamics.url"),
		TenantID:     viper.GetString("dynamics.tenantid"),
		ClientID:     viper.GetString("dynamics.clientid"),
		ClientSecret: viper.GetString("dynamics.clientsecret"),
	}
	dynamicsClient.AuthenticateApi()

	if strings.Contains(endpoint, "(") {
		return dynamicsClient.PatchRequest(endpoint, data)
	} else {
		return dynamicsClient.PostRequest(endpoint, data)
	}
}

func (repo *ReportFormRepository) GetForm(id entity.ID) (e entity.Form, err error) {
	err = repo.db.Debug().Model(e).Preload("Events").Where("id = ?", id).Find(&e).Error
	return
}

func (repo *ReportFormRepository) GetEvent(id entity.ID) (e entity.Event, err error) {
	err = repo.db.Debug().Model(e).Where("form_type = 1 and id = ?", id).Find(&e).Error
	return
}

func (repo *ReportFormRepository) Create(e *entity.Form) (id entity.ID, err error) {
	err = repo.db.Debug().Create(e).Error
	id = e.ID
	return
}

func (repo *ReportFormRepository) Update(id *entity.Form) (err error) {
	/* err = repo.db.Debug().Create(e).Error
	id = e.ID */
	return
}

func (repo *ReportFormRepository) CreateOrUpdate(e *entity.Form) (err error) {
	/* err = repo.db.Debug().Create(e).Error
	id = e.ID */
	return
}

func (repo *ReportFormRepository) Delete(e *entity.Form) (err error) {
	err = repo.db.Debug().Where("id = ?", e.ID).Delete(&e).Error
	return
}
