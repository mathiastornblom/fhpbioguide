// Package handler contains functions for handling various exports.
package handler

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"fhpbioguide/pkg/entity"
	"fhpbioguide/pkg/usecase/cashreports"
	"fhpbioguide/pkg/usecase/movieexport"
	"fhpbioguide/pkg/usecase/theatreexport"
)

func ExecuteExports(lastSync time.Time, movieService movieexport.UseCase, cashreportService cashreports.UseCase, theatreService theatreexport.UseCase, log *slog.Logger) error {
	// NOTE: TheatreExport is intentionally NOT called here — matches original behaviour.
	MovieExport(movieService, log)
	return CashExport(lastSync, cashreportService, movieService, theatreService, log)
}

func MovieExport(service movieexport.UseCase, log *slog.Logger) error {
	l := log.With("component", "MovieExport")
	l.Info("starting")

	data, err := service.Export(time.Date(2018, time.January, 1, 0, 0, 0, 0, time.Local), time.Date(2030, time.December, 31, 0, 0, 0, 0, time.Local))
	if err != nil {
		l.Error("failed to fetch BioGuiden export", "err", err)
		return fmt.Errorf("failed to fetch exported data: %w", err)
	}

	movies, err := service.FetchFromD365()
	if err != nil {
		l.Error("failed to fetch movies from D365", "err", err)
		return fmt.Errorf("failed to fetch movies from Dynamics 365: %w", err)
	}

	all := data.Body.ExportResponse.Document.Data.Movies.Movie
	numTotal := len(all)
	processed := 0
	start := time.Now()

	for id, movie := range all {
		l.Debug("processing movie", "progress", fmt.Sprintf("%d/%d", id+1, numTotal), "title", movie.Title)

		if movie.Distributor.Name != "" && movie.Distributor.Name == "Folkets Hus och Parker" {
			newMovie := true
			for _, d365Movie := range movies {
				if strings.Contains(movie.FullMovieNumber, "-") {
					parts := strings.Split(movie.FullMovieNumber, "-")
					if len(parts) >= 3 && parts[2] == d365Movie.Productnumber {
						l.Debug("movie already in D365, skipping", "productnumber", parts[2])
						newMovie = false
						break
					}
				}
			}
			if !newMovie {
				continue
			}

			var premiereDate time.Time
			if movie.PremiereDate != "" {
				if d, err := time.Parse("2006-01-02T15:04:05", movie.PremiereDate); err == nil {
					premiereDate = d
				} else {
					l.Debug("skipping movie with invalid premiere date", "title", movie.Title, "err", err)
					continue
				}
			} else {
				l.Debug("skipping movie with no premiere date", "title", movie.Title)
				continue
			}

			var runtime int
			if movie.Runtime != "" {
				if r, err := strconv.Atoi(movie.Runtime); err == nil {
					runtime = r
				} else {
					l.Debug("skipping movie with invalid runtime", "title", movie.Title, "err", err)
					continue
				}
			} else {
				l.Debug("skipping movie with no runtime", "title", movie.Title)
				continue
			}

			item := entity.Product{
				Name:             movie.Title,
				Description:      movie.Description,
				Vendorname:       movie.Distributor.Name,
				Length:           runtime,
				Distributor:      100000000,
				Censur:           censurToID(movie.Rating),
				Productnumber:    strings.Split(movie.FullMovieNumber, "-")[2],
				Producttypecode:  100000001,
				Productstructure: 1,
				Premier:          premiereDate,
				ShowdateStart:    premiereDate,
				Textning:         movie.Subtitles == "Svenska",
				UnitScheduleID:   "/uomschedules(77267707-68dc-4f61-be8a-b29394aeffb2)",
				UnitID:           "/uoms(38b503b0-94cb-4a79-ab8f-1fa060e1e5e2)",
			}

			jsData, err := json.Marshal(item)
			if err != nil {
				l.Error("failed to marshal product to JSON", "title", movie.Title, "err", err)
				return fmt.Errorf("failed to marshal product to JSON: %w", err)
			}
			if _, err := service.PostToD365(string(jsData)); err != nil {
				l.Error("failed to post product to D365", "title", movie.Title, "err", err)
				return fmt.Errorf("failed to post product to D365: %w", err)
			}
			processed++
		}
	}

	l.Info("completed", "processed", processed, "total", numTotal, "duration", time.Since(start).Round(time.Second).String())
	return nil
}

func censurToID(value string) int {
	switch value {
	case "Barntillåten":
		return 100000000
	case "Från 7 år":
		return 100000001
	case "Från 11 år":
		return 100000002
	case "Från 15 år":
		return 100000000
	}
	return 100000004
}

func TheatreExport(service theatreexport.UseCase, log *slog.Logger) {
	l := log.With("component", "TheatreExport")
	l.Info("starting")
	start := time.Now()

	data, err := service.Export("")
	if err != nil {
		l.Error("BioGuiden export failed", "err", err)
		return
	}

	d365Lokals, _ := service.FetchFromD365()

	theatres := data.Body.ExportResponse.Document.Data.Theatres.Theatre
	numTheatres := len(theatres)
	processed := 0

	for id, theatre := range theatres {
		l.Debug("processing theatre", "progress", fmt.Sprintf("%d/%d", id+1, numTheatres), "name", theatre.TheatreNumber)

		numSalons := len(theatre.Salons.Salon)
		for sid, salon := range theatre.Salons.Salon {
			l.Debug("processing salon", "progress", fmt.Sprintf("%d/%d", sid+1, numSalons), "fkb", salon.FkbNumber)

			newLocation := true
			for _, d365Salon := range d365Lokals {
				if salon.FkbNumber == d365Salon.FkbNum {
					l.Debug("salon already in D365, skipping", "fkb", salon.FkbNumber)
					newLocation = false
				}
			}
			if !newLocation {
				continue
			}

			location := entity.LokalDynamics{
				LoakalID:      salon.ID,
				Name:          salon.SalonName,
				Adress1:       theatre.Address.Street0,
				Adress2:       theatre.Address.Street1,
				ZipCode:       theatre.Address.Zip,
				City:          theatre.Address.City,
				VisitAdress:   theatre.Address.Street1,
				VisitCity:     theatre.Address.City,
				Country:       100000000,
				Email:         theatre.ContactInfo.Email,
				Phone:         theatre.ContactInfo.Phone,
				Fax:           theatre.ContactInfo.Fax,
				TheatreNum:    theatre.TheatreNumber,
				OwnerNum:      salon.OwnerNumber,
				SalonNum:      salon.SalonNumber,
				FkbNum:        salon.FkbNumber,
				NumberOfSeats: salon.Seats,
				BusinessUnit:  "/businessunits(d7e6eb96-d458-ec11-8f8f-6045bd8aa5cb)",
			}

			for _, tech := range salon.SupportedTechnologies.Technology {
				switch tech.ID {
				case "2K":
					location.CanShow2D, _ = strconv.ParseBool(tech.Supported)
				case "3D":
					location.CanShow3D, _ = strconv.ParseBool(tech.Supported)
				case "Dolby Atmos":
					location.CanShowAtmos, _ = strconv.ParseBool(tech.Supported)
				case "IMAX":
					location.CanShowImax, _ = strconv.ParseBool(tech.Supported)
				case "35mm":
					location.CanShow35mm, _ = strconv.ParseBool(tech.Supported)
				case "70mm":
					location.CanShow70mm, _ = strconv.ParseBool(tech.Supported)
				case "Digital 5.1":
					location.CanShow51Sound, _ = strconv.ParseBool(tech.Supported)
				case "Digital 7.1":
					location.CanShow71Sound, _ = strconv.ParseBool(tech.Supported)
				case "4DX 2D", "4DX 3D":
					location.CanShow4DX, _ = strconv.ParseBool(tech.Supported)
				}
			}

			jsData2, _ := json.Marshal(location)
			service.PostToD365("new_lokals", string(jsData2))
			processed++
		}
	}

	l.Info("completed", "processed", processed, "duration", time.Since(start).Round(time.Second).String())
}

// CashExport fetches all dates with approved cash-report changes since lastSync
// and processes each one using CashExportWithDate.
func CashExport(lastSync time.Time, service cashreports.UseCase, movieService movieexport.UseCase, theatreService theatreexport.UseCase, log *slog.Logger) error {
	l := log.With("component", "CashExport")
	l.Info("fetching activity dates", "since", lastSync.Format("2006-01-02"))
	start := time.Now()

	list, err := service.ExportList(lastSync, time.Now())
	if err != nil {
		l.Error("ExportList failed", "err", err)
		return fmt.Errorf("CashExport: ExportList failed: %w", err)
	}

	dates := list.Body.ExportResponse.Document.Data.Dates.Date
	l.Info("dates to process", "count", len(dates))

	for i, d := range dates {
		t, parseErr := time.Parse("2006-01-02", d.UpdatedDate)
		if parseErr != nil {
			l.Warn("skipping unparseable date", "date", d.UpdatedDate, "err", parseErr)
			continue
		}
		l.Info("processing date", "progress", fmt.Sprintf("%d/%d", i+1, len(dates)), "date", d.UpdatedDate, "reports", d.NumberOfCashreports)
		CashExportWithDate(t, service, movieService, theatreService, l)
	}

	l.Info("completed", "dates", len(dates), "duration", time.Since(start).Round(time.Second).String())
	return nil
}

func CashExportWithDate(updatedDate time.Time, service cashreports.UseCase, movieService movieexport.UseCase, theatreService theatreexport.UseCase, log *slog.Logger) {
	l := log.With("component", "CashExportWithDate")
	data, _ := service.Export(updatedDate)
	reports := data.Body.ExportResponse.Document.Data.Cashreports.Cashreport
	numTotal := len(reports)

	for id1, report := range reports {
		l.Debug("processing report", "progress", fmt.Sprintf("%d/%d", id1+1, numTotal), "report", report.CashreportNumber)

		var incomingApproved *time.Time
		if report.Approved.DistributorDate != "" {
			if t, err := time.Parse("2006-01-02T15:04:05", report.Approved.DistributorDate); err == nil {
				incomingApproved = &t
			}
		}

		existing, _ := service.FilteredFetchD365("new_cashreportnumber%20eq%20'" + report.CashreportNumber + "'")

		isDuplicate := false

		if len(existing) > 0 {
			var storedApproved *time.Time
			for _, row := range existing {
				if row.ApprovedDate != nil {
					storedApproved = row.ApprovedDate
					break
				}
			}

			if incomingApproved == nil || (storedApproved != nil && !incomingApproved.After(*storedApproved)) {
				l.Debug("report already up to date, skipping", "report", report.CashreportNumber)
				continue
			}

			l.Info("correction detected", "report", report.CashreportNumber, "stored", storedApproved, "incoming", incomingApproved)

			anyInvoiced := false
			for _, row := range existing {
				if row.Invoiced {
					anyInvoiced = true
					break
				}
			}

			if anyInvoiced {
				l.Warn("report has invoiced rows — flagging as duplicate", "report", report.CashreportNumber)
				isDuplicate = true
				for _, row := range existing {
					if !row.IsDuplicate {
						service.PostToD365("new_cashreports("+row.ID+")", `{"new_isduplicate":true}`)
					}
				}
			} else {
				l.Info("deleting stale rows for re-import", "report", report.CashreportNumber, "rows", len(existing))
				for _, row := range existing {
					if err := service.DeleteFromD365("new_cashreports(" + row.ID + ")"); err != nil {
						l.Error("failed to delete stale row", "row_id", row.ID, "err", err)
					}
				}
			}
		}

		numShows := len(report.Shows.Show)
		for id, show := range report.Shows.Show {
			l.Debug("processing show", "progress", fmt.Sprintf("%d/%d", id+1, numShows), "report", report.CashreportNumber)

			movieNum := "" + strings.Split(report.Movie.FullMovieNumber, "-")[2]
			playweekStart, _ := time.Parse("2006-01-02T15:04:05", report.Playweek.StartDate)
			playweekEnd, _ := time.Parse("2006-01-02T15:04:05", report.Playweek.EndDate)
			recordedAmount, _ := strconv.ParseFloat(strings.ReplaceAll(show.TotalDistributorAmount, ",", "."), 32)

			reportLine := entity.DynamicsCashReport{
				FKBID:           report.Salon.FkbNumber,
				Source:          100000000,
				Playweek:        playweekStart.Format("2006-01-02") + " - " + playweekEnd.Format("2006-01-02"),
				RecordedAmount:  recordedAmount,
				ReportNum:       report.CashreportNumber,
				ShowNum:         id + 1,
				FullMovieNumber: movieNum,
				ApprovedDate:    incomingApproved,
				IsDuplicate:     isDuplicate,
			}

			movies, _ := movieService.FilteredFetchD365("productnumber%20eq%20'" + movieNum + "'")
			theatres, _ := theatreService.FilteredFetchD365("new_fkbid%20eq%20'" + report.Salon.FkbNumber + "'")

			if len(movies) > 0 {
				reportLine.Event = "/products(" + movies[0].ID + ")"
			}
			if len(theatres) > 0 {
				reportLine.Lokal = "/new_lokals(" + theatres[0].LoakalID + ")"
				reportLine.Account = "/accounts(" + theatres[0].AccountData.Accountid + ")"
			}

			showTime, _ := time.Parse("2006-01-02T15:04:05", show.StartDateTime)

			if len(theatres) > 0 && len(movies) > 0 {
				bookings, _ := service.FindBookingD365(
					"_new_customer_value%20eq%20" + theatres[0].AccountData.Accountid +
						"%20and%20_new_product_value%20eq%20" + movies[0].ID +
						"%20and%20new_showdate%20eq%20" + showTime.Format("2006-01-02"))

				if len(bookings) > 0 &&
					shouldLinkBooking(service, bookings[0].ID, report.CashreportNumber, log) {
					reportLine.Booking = "/new_bokningarkunds(" + bookings[0].ID + ")"
					service.PostToD365(
						"new_bokningarkunds("+bookings[0].ID+")",
						`{"new_Lokaler@odata.bind":"/new_lokals(`+theatres[0].LoakalID+`)"}`)
				}
			}

			reportLine.VatFree = report.Salon.VatFree == "1"
			reportLine.ShowDate = showTime

			for _, ticketDetail := range show.TicketDetails.Detail {
				quantity, _ := strconv.Atoi(ticketDetail.Quantity)
				price, _ := strconv.ParseFloat(strings.ReplaceAll(ticketDetail.Price, ",", "."), 32)

				reportLine.Name = ticketDetail.Category
				reportLine.TicketName = ticketDetail.Category
				reportLine.TicketQuantity = quantity
				reportLine.TicketPrice = price

				jsData, _ := json.Marshal(reportLine)
				service.PostToD365("new_cashreports", string(jsData))
			}
		}
	}
}

func UpdateCashreports(updatedDate time.Time, service cashreports.UseCase, movieService movieexport.UseCase, theatreService theatreexport.UseCase) {
	data, _ := service.Export(updatedDate)
	numTotal1 := len(data.Body.ExportResponse.Document.Data.Cashreports.Cashreport)

	for id1, report := range data.Body.ExportResponse.Document.Data.Cashreports.Cashreport {
		fmt.Printf("Working on report %v / %v \n\r", id1, numTotal1)
		d365Cashreport, _ := service.FilteredFetchD365("new_source%20eq%20100000000%20and%20new_cashreportnumber%20eq%20'" + report.CashreportNumber + "'")

		for _, d365Cr := range d365Cashreport {
			if report.CashreportNumber == d365Cr.ReportNum {
				movieNum := "" + strings.Split(report.Movie.FullMovieNumber, "-")[2]
				service.PostToD365("new_cashreports("+d365Cr.ID+")", `{"new_fullmovienumber":"`+movieNum+`"}`)
			}
		}
	}
}

// shouldLinkBooking returns true if the cash-report row may link to the booking.
func shouldLinkBooking(s cashreports.UseCase, bookingID, reportNum string, log *slog.Logger) bool {
	existing, _ := s.FilteredFetchD365("_new_booking_value%20eq%20" + bookingID)

	if len(existing) == 0 {
		return true
	}

	if existing[0].ReportNum != reportNum {
		log.Warn("booking already tied to different cash report — skipping link",
			"booking", bookingID,
			"existing_report", existing[0].ReportNum,
			"incoming_report", reportNum)
		return false
	}

	return true
}
