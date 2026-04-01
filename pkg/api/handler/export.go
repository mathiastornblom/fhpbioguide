// Package handler contains functions for handling various exports.
package handler

import (
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	// Import the required packages for handling entities and use cases.
	"fhpbioguide/pkg/entity"
	"fhpbioguide/pkg/usecase/cashreports"
	"fhpbioguide/pkg/usecase/movieexport"
	"fhpbioguide/pkg/usecase/theatreexport"
)

func ExecuteExports(lastSync time.Time, movieService movieexport.UseCase, cashreportService cashreports.UseCase, theatreService theatreexport.UseCase) error {
	MovieExport(movieService)
	return CashExport(lastSync, cashreportService, movieService, theatreService)
}

func MovieExport(service movieexport.UseCase) error {
	// Fetch movies updated between 1st of January 2018 and 31st of december 2030
	data, err := service.Export(time.Date(2018, time.January, 1, 0, 0, 0, 0, time.Local), time.Date(2030, time.December, 31, 0, 0, 0, 0, time.Local))
	if err != nil {
		return fmt.Errorf("failed to fetch exported data: %w", err)
	}

	// Fetch all movies from Dynamics 365
	movies, err := service.FetchFromD365()
	if err != nil {
		return fmt.Errorf("failed to fetch movies from Dynamics 365: %w", err)
	}

	// Get the total number of movies to be processed
	numTotal := len(data.Body.ExportResponse.Document.Data.Movies.Movie)

	// Loop through each movie in the export data
	for id, movie := range data.Body.ExportResponse.Document.Data.Movies.Movie {
		fmt.Printf("Working on movie %v/%v \n\r", id, numTotal)

		// Check if the movie's distributor is "Folkets Hus och Parker"
		if movie.Distributor.Name != "" && movie.Distributor.Name == "Folkets Hus och Parker" {
			newMovie := true

			// Check if movie is already in Dynamics 365
			for _, d365Movie := range movies {
				if strings.Contains(movie.FullMovieNumber, "-") {
					parts := strings.Split(movie.FullMovieNumber, "-")
					if len(parts) >= 3 && parts[2] == d365Movie.Productnumber {
						fmt.Println("Movie already in Dynamics")
						newMovie = false
						break
					}
				}
			}

			// If the movie already exists in Dynamics, move to the next movie
			if !newMovie {
				continue
			}

			// Convert premiere date and runtime to appropriate format
			var premiereDate time.Time
			if movie.PremiereDate != "" {
				if d, err := time.Parse("2006-01-02T15:04:05", movie.PremiereDate); err == nil {
					premiereDate = d
				} else {
					fmt.Printf("Skipping movie with invalid premiere date: %v\n\r", err)
					continue
				}
			} else {
				fmt.Println("Skipping movie with no premiere date")
				continue
			}

			var runtime int
			if movie.Runtime != "" {
				if r, err := strconv.Atoi(movie.Runtime); err == nil {
					runtime = r
				} else {
					fmt.Printf("Skipping movie with invalid runtime: %v\n\r", err)
					continue
				}
			} else {
				fmt.Println("Skipping movie with no runtime")
				continue
			}

			// Create a new Product entity using the movie data
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

			// Export the Product entity to Dynamics 365
			jsData, err := json.Marshal(item)
			if err != nil {
				return fmt.Errorf("failed to marshal product to JSON: %w", err)
			}
			if _, err := service.PostToD365(string(jsData)); err != nil {
				return fmt.Errorf("failed to post product to D365: %w", err)
			}
		}
	}

	return nil
}

/*
*
censurToID maps a given censorship value to a corresponding ID in Dynamics 365.
@param value The censorship value to be mapped.
@return The corresponding ID in Dynamics 365.
*/
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

func TheatreExport(service theatreexport.UseCase) {
	data, err := service.Export("")

	if err != nil {
		fmt.Println(err.Error())
	}

	// Fetch existing data from Dynamics 365
	d365Lokals, _ := service.FetchFromD365()

	// Loop over all theatres and their associated salons
	numTotal := len(data.Body.ExportResponse.Document.Data.Theatres.Theatre)
	for id, theatre := range data.Body.ExportResponse.Document.Data.Theatres.Theatre {
		fmt.Printf("Working on theatre %v/%v \n\r", id, numTotal)

		numTotal2 := len(theatre.Salons.Salon)
		for id, salon := range theatre.Salons.Salon {
			fmt.Printf("Working on salon %v/%v \n\r", id, numTotal2)

			// Check if salon is already in Dynamics 365
			newLocation := true
			for _, d365Salon := range d365Lokals {
				if salon.FkbNumber == d365Salon.FkbNum {
					fmt.Println("Salon already in dynamics")
					newLocation = false
				}
			}
			if !newLocation {
				// Skip if the salon already exists in Dynamics 365
				continue
			}

			// Create a new LokalDynamics object
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

			// Loop over the supported technologies and set the corresponding flags
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

			// Export the data to JSON and send it to Dynamics 365
			jsData2, _ := json.Marshal(location)
			service.PostToD365("new_lokals", string(jsData2))
		}
	}
}

// CashExport fetches all dates with approved cash-report changes since lastSync
// and processes each one using CashExportWithDate. Using ExportList means we
// only call BioGuiden for dates that actually had activity, and CashExportWithDate
// deduplicates against D365 so re-running any date is safe.
//
// Returns an error only if the ExportList call itself fails — individual date
// failures are logged and skipped so the rest of the backfill continues.
func CashExport(lastSync time.Time, service cashreports.UseCase, movieService movieexport.UseCase, theatreService theatreexport.UseCase) error {
	log.Printf("CashExport: fetching activity dates since %s", lastSync.Format("2006-01-02"))

	list, err := service.ExportList(lastSync, time.Now())
	if err != nil {
		return fmt.Errorf("CashExport: ExportList failed: %w", err)
	}

	dates := list.Body.ExportResponse.Document.Data.Dates.Date
	log.Printf("CashExport: %d date(s) to process", len(dates))

	for i, d := range dates {
		t, parseErr := time.Parse("2006-01-02", d.UpdatedDate)
		if parseErr != nil {
			log.Printf("CashExport: skipping unparseable date %q: %v", d.UpdatedDate, parseErr)
			continue
		}
		log.Printf("CashExport: processing date %d/%d: %s (%s report(s))", i+1, len(dates), d.UpdatedDate, d.NumberOfCashreports)
		CashExportWithDate(t, service, movieService, theatreService)
	}

	return nil
}

func CashExportWithDate(updatedDate time.Time, service cashreports.UseCase, movieService movieexport.UseCase, theatreService theatreexport.UseCase) {
	data, _ := service.Export(updatedDate)
	numTotal1 := len(data.Body.ExportResponse.Document.Data.Cashreports.Cashreport)

	for id1, report := range data.Body.ExportResponse.Document.Data.Cashreports.Cashreport {
		fmt.Printf("Working on report %v / %v \n\r", id1, numTotal1)

		// Parse distributor-date from BioGuiden — used for correction detection.
		var incomingApproved *time.Time
		if report.Approved.DistributorDate != "" {
			if t, err := time.Parse("2006-01-02T15:04:05", report.Approved.DistributorDate); err == nil {
				incomingApproved = &t
			}
		}

		existing, _ := service.FilteredFetchD365("new_cashreportnumber%20eq%20'" + report.CashreportNumber + "'")

		isDuplicate := false

		if len(existing) > 0 {
			// Determine the stored approved date from any row that has one.
			var storedApproved *time.Time
			for _, row := range existing {
				if row.ApprovedDate != nil {
					storedApproved = row.ApprovedDate
					break
				}
			}

			// If incoming date is not newer than what is stored, nothing to do.
			if incomingApproved == nil || (storedApproved != nil && !incomingApproved.After(*storedApproved)) {
				log.Printf("CashExportWithDate: report %s already up to date, skipping", report.CashreportNumber)
				continue
			}

			// Correction detected — incoming distributor-date is newer.
			log.Printf("CashExportWithDate: correction detected for report %s (stored: %v, incoming: %v)",
				report.CashreportNumber, storedApproved, incomingApproved)

			anyInvoiced := false
			for _, row := range existing {
				if row.Invoiced {
					anyInvoiced = true
					break
				}
			}

			if anyInvoiced {
				// Cannot delete invoiced rows — mark old rows and incoming rows as duplicate.
				log.Printf("CashExportWithDate: report %s has invoiced rows, flagging as duplicate", report.CashreportNumber)
				isDuplicate = true
				for _, row := range existing {
					if !row.IsDuplicate {
						service.PostToD365("new_cashreports("+row.ID+")", `{"new_isduplicate":true}`)
					}
				}
			} else {
				// All un-invoiced — delete old rows and re-insert with updated data.
				log.Printf("CashExportWithDate: report %s — deleting %d stale row(s) for re-import", report.CashreportNumber, len(existing))
				for _, row := range existing {
					if err := service.DeleteFromD365("new_cashreports(" + row.ID + ")"); err != nil {
						log.Printf("CashExportWithDate: failed to delete row %s: %v", row.ID, err)
					}
				}
			}
		}

		// Insert rows for new report, corrected un-invoiced report, or duplicate correction.
		numTotal := len(report.Shows.Show)
		for id, show := range report.Shows.Show {
			fmt.Printf("Working on show %v / %v \n\r", id, numTotal)
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
					shouldLinkBooking(service, bookings[0].ID, report.CashreportNumber) {

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
	// Export cashreport data based on the given updatedDate
	data, _ := service.Export(updatedDate)

	// Get the total number of cashreports to be processed
	numTotal1 := len(data.Body.ExportResponse.Document.Data.Cashreports.Cashreport)

	// Iterate through each cashreport and check if it is already in Dynamics365
	for id1, report := range data.Body.ExportResponse.Document.Data.Cashreports.Cashreport {
		fmt.Printf("Working on report %v / %v \n\r", id1, numTotal1)
		log.Printf("Working on report %v / %v \n\r", id1, numTotal1)
		d365Cashreport, _ := service.FilteredFetchD365("new_source%20eq%20100000000%20and%20new_cashreportnumber%20eq%20'" + report.CashreportNumber + "'")

		for _, d365Cr := range d365Cashreport {
			if report.CashreportNumber == d365Cr.ReportNum {
				// If the cashreport is already in Dynamics365, update its FullMovieNumber field with the extracted movie number
				fmt.Println("Report " + d365Cr.ReportNum + " found in dynamics")

				// Extract the movie number from the FullMovieNumber field
				movieNum := "" + strings.Split(report.Movie.FullMovieNumber, "-")[2]

				// Update the FullMovieNumber field in Dynamics365
				service.PostToD365("new_cashreports("+d365Cr.ID+")", `{"new_fullmovienumber":"`+movieNum+`"}`)
			}
		}
	}
}

// shouldLinkBooking tells if the current cash-report row may link to a booking.
//
// Link is allowed when
//   - the booking has no linked rows, or
//   - every linked row (we read the first) shares the same cash-report number.
//
// When a mismatch is found we print and log a note, then refuse the link.
//
// Params
//
//	s         – service that talks to Dynamics 365
//	bookingID – booking GUID, no braces or quotes
//	reportNum – cash-report number of the row we process
//
// Returns
//
//	true  – link is allowed
//	false – link is denied
func shouldLinkBooking(s cashreports.UseCase, bookingID, reportNum string) bool {

	// Fetch rows already linked to this booking.
	existing, _ := s.FilteredFetchD365(
		"_new_booking_value%20eq%20" + bookingID)

	// No row linked → safe to link.
	if len(existing) == 0 {
		return true
	}

	// A row exists. Check its cash-report number.
	if existing[0].ReportNum != reportNum {
		fmt.Printf(
			"Skip link: booking %s already tied to cash-report %s, current %s\n",
			bookingID, existing[0].ReportNum, reportNum)
		log.Printf(
			"Skip link: booking %s already tied to cash-report %s, current %s",
			bookingID, existing[0].ReportNum, reportNum)
		return false
	}

	// Match found → link is allowed.
	return true
}
