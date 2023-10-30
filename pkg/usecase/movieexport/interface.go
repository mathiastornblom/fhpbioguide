package movieexport

import (
	"time"

	"fhpbioguide/pkg/entity"
)

//Reader interface
type Reader interface {
	Export(startDate, endDate time.Time) (*entity.MovieExportList, error)
	FetchFromD365() ([]*entity.Product, error)
	FilteredFetchD365(filter string) ([]*entity.Product, error)
}

//Writer user writer
type Writer interface {
	PostToD365(data string) ([]byte, error)
}

//Repository interface
type Repository interface {
	Reader
	Writer
}

//UseCase interface
type UseCase interface {
	Export(startDate, endDate time.Time) (*entity.MovieExportList, error)
	PostToD365(data string) ([]byte, error)
	FetchFromD365() ([]*entity.Product, error)
	FilteredFetchD365(filter string) ([]*entity.Product, error)
}
