package reportform

import (
	"fhpbioguide/pkg/entity"
)

// Service  interface
type Service struct {
	repo Repository
}

// NewService create new use case
func NewService(r Repository) *Service {
	return &Service{
		repo: r,
	}
}

func (s *Service) PostToD365(endpoint, data string) ([]byte, error) {
	return s.repo.PostToD365(endpoint, data)
}
func (s *Service) GetFromD365(endpoint string) ([]byte, error) {
	return s.repo.GetFromD365(endpoint)
}

func (s *Service) GetForm(id entity.ID) (entity.Form, error) {
	return s.repo.GetForm(id)
}
func (s *Service) GetEvent(id entity.ID) (entity.Event, error) {
	return s.repo.GetEvent(id)
}
func (s *Service) Create(e *entity.Form) (entity.ID, error) {
	return s.repo.Create(e)
}
func (s *Service) Update(e *entity.Form) error {
	return s.repo.Update(e)
}
func (s *Service) CreateOrUpdate(e *entity.Form) error {
	return s.repo.CreateOrUpdate(e)
}
func (s *Service) Delete(e *entity.Form) error {
	return s.repo.Delete(e)
}
