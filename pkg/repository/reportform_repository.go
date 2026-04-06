package repository

import (
	"log/slog"
	"strings"
	"time"

	"fhpbioguide/pkg/api/d365"
	"fhpbioguide/pkg/entity"

	"github.com/go-resty/resty/v2"
	"github.com/spf13/viper"
	"gorm.io/gorm"
)

// ReportFormRepository handles MySQL (via GORM) and D365 REST for report forms.
type ReportFormRepository struct {
	db       *gorm.DB
	dynamics *d365.D365
	log      *slog.Logger
}

// NewReportFormRepository creates a new repository with a shared D365 client.
// The client authenticates once here; token auto-refresh handles subsequent expiry.
func NewReportFormRepository(db *gorm.DB, log *slog.Logger) *ReportFormRepository {
	dynamicsClient := &d365.D365{
		Resty:        resty.New().SetTimeout(30 * time.Second),
		URL:          viper.GetString("dynamics.url"),
		TenantID:     viper.GetString("dynamics.tenantid"),
		ClientID:     viper.GetString("dynamics.clientid"),
		ClientSecret: viper.GetString("dynamics.clientsecret"),
		Logger:       log,
	}
	if err := dynamicsClient.AuthenticateApi(); err != nil {
		log.Warn("initial D365 auth failed (will retry on first request)", "err", err)
	}
	return &ReportFormRepository{
		db:       db,
		dynamics: dynamicsClient,
		log:      log,
	}
}

func (repo *ReportFormRepository) GetFromD365(endpoint string) ([]byte, error) {
	return repo.dynamics.GetRequest(endpoint)
}

func (repo *ReportFormRepository) PostToD365(endpoint, data string) ([]byte, error) {
	if strings.Contains(endpoint, "(") {
		return repo.dynamics.PatchRequest(endpoint, data)
	}
	return repo.dynamics.PostRequest(endpoint, data)
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
	return
}

func (repo *ReportFormRepository) UpdateEvent(e *entity.Event) error {
	return repo.db.Debug().Save(e).Error
}

func (repo *ReportFormRepository) CreateOrUpdate(e *entity.Form) (err error) {
	return
}

func (repo *ReportFormRepository) Delete(e *entity.Form) (err error) {
	err = repo.db.Debug().Where("id = ?", e.ID).Delete(&e).Error
	return
}
