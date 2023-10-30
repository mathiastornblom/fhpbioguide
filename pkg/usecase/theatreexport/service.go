package theatreexport

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

func (s *Service) Export(data string) (*entity.TheatreExportList, error) {
	return s.repo.Export(data)
}
func (s *Service) PostToD365(endpoint, data string) ([]byte, error) {
	return s.repo.PostToD365(endpoint, data)
}

func (s *Service) FetchFromD365() ([]*entity.LokalDynamics, error) {
	return s.repo.FetchFromD365()
}
func (s *Service) FilteredFetchD365(filter string) ([]*entity.LokalDynamics, error) {
	return s.repo.FilteredFetchD365(filter)
}
