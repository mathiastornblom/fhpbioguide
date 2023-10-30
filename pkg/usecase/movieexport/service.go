package movieexport

import (
	"time"

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

func (s *Service) Export(startDate, endDate time.Time) (*entity.MovieExportList, error) {
	return s.repo.Export(startDate, endDate)
}
func (s *Service) PostToD365(data string) ([]byte, error) {
	return s.repo.PostToD365(data)
}
func (s *Service) FetchFromD365() ([]*entity.Product, error) {
	return s.repo.FetchFromD365()
}
func (s *Service) FilteredFetchD365(filter string) ([]*entity.Product, error) {
	return s.repo.FilteredFetchD365(filter)
}
