package organisation

import (
	"fhpbioguide/pkg/entity"
)

//Service  interface
type Service struct {
	repo Repository
}

//NewService create new use case
func NewService(r Repository) *Service {
	return &Service{
		repo: r,
	}
}

func (s *Service) PostToD365(endpoint, data string) ([]byte, error) {
	return s.repo.PostToD365(endpoint, data)
}
func (s *Service) FetchFromD365() ([]*entity.DynamicsBooking, error) {
	return s.repo.FetchFromD365()
}
