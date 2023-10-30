package organisation

import (
	"fhpbioguide/pkg/entity"
)

//Reader interface
type Reader interface {
	FetchFromD365() ([]*entity.DynamicsBooking, error)
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
	FetchFromD365() ([]*entity.DynamicsBooking, error)
	PostToD365(endpoint, data string) ([]byte, error)
}
