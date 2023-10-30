package theatreexport

import (
	"fhpbioguide/pkg/entity"
)

//Reader interface
type Reader interface {
	Export(data string) (*entity.TheatreExportList, error)
	FetchFromD365() ([]*entity.LokalDynamics, error)
	FilteredFetchD365(filter string) ([]*entity.LokalDynamics, error)
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
	Export(data string) (*entity.TheatreExportList, error)
	PostToD365(endpoint, data string) ([]byte, error)
	FetchFromD365() ([]*entity.LokalDynamics, error)
	FilteredFetchD365(filter string) ([]*entity.LokalDynamics, error)
}
