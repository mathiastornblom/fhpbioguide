package cashreports

import (
	"time"

	"fhpbioguide/pkg/entity"
)

//Reader interface
type Reader interface {
	Export(startDate time.Time) (*entity.CashReportsDistributoristExport, error)
	ExportList(startDate, endDate time.Time) (*entity.CashReportsDistributoristListExport, error)
	FetchFromD365() ([]*entity.DynamicsBooking, error)
	FindBookingD365(filter string) ([]*entity.DynamicsBooking, error)
	FilteredFetchD365(filter string) ([]*entity.DynamicsCashReport, error)
}

//Writer user writer
type Writer interface {
	PostToD365(endpoint, data string) ([]byte, error)
}

//Repository interface
type Repository interface {
	Reader
	Writer
}

//UseCase interface
type UseCase interface {
	Export(startDate time.Time) (*entity.CashReportsDistributoristExport, error)
	ExportList(startDate, endDate time.Time) (*entity.CashReportsDistributoristListExport, error)
	FetchFromD365() ([]*entity.DynamicsBooking, error)
	FindBookingD365(filter string) ([]*entity.DynamicsBooking, error)
	PostToD365(endpoint, data string) ([]byte, error)
	FilteredFetchD365(filter string) ([]*entity.DynamicsCashReport, error)
}
