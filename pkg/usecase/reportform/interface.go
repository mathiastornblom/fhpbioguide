package reportform

import (
	"fhpbioguide/pkg/entity"
)

// Reader interface
type Reader interface {
	GetFromD365(endpoint string) ([]byte, error)
	GetForm(id entity.ID) (entity.Form, error)
	GetEvent(id entity.ID) (entity.Event, error)
}

// Writer user writer
type Writer interface {
	PostToD365(endpoint, data string) ([]byte, error)
	Create(e *entity.Form) (entity.ID, error)
	Update(e *entity.Form) error
	CreateOrUpdate(e *entity.Form) error
	Delete(e *entity.Form) error
}

// Repository interface
type Repository interface {
	Reader
	Writer
}

// UseCase interface
type UseCase interface {
	GetFromD365(endpoint string) ([]byte, error)
	GetForm(id entity.ID) (entity.Form, error)
	GetEvent(id entity.ID) (entity.Event, error)
	PostToD365(endoint, data string) ([]byte, error)
	Create(e *entity.Form) (entity.ID, error)
	Update(e *entity.Form) error
	CreateOrUpdate(e *entity.Form) error
	Delete(e *entity.Form) error
}
