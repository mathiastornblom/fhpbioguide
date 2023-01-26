package cashreports

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

func (s *Service) Export(startDate time.Time) (*entity.CashReportsDistributoristExport, error) {
	return s.repo.Export(startDate)
}
func (s *Service) ExportList(startDate, endDate time.Time) (*entity.CashReportsDistributoristListExport, error) {
	return s.repo.ExportList(startDate, endDate)
}
func (s *Service) PostToD365(endpoint, data string) ([]byte, error) {
	return s.repo.PostToD365(endpoint, data)
}
func (s *Service) FetchFromD365() ([]*entity.DynamicsBooking, error) {
	return s.repo.FetchFromD365()
}
func (s *Service) FindBookingD365(filter string) ([]*entity.DynamicsBooking, error) {
	return s.repo.FindBookingD365(filter)
}
func (s *Service) FilteredFetchD365(filter string) ([]*entity.DynamicsCashReport, error) {
	return s.repo.FilteredFetchD365(filter)
}
