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

func ExecuteExports(movieService movieexport.UseCase, cashreportService cashreports.UseCase, theatreService theatreexport.UseCase) {
	MovieExport(movieService)
	CashExport(cashreportService, movieService, theatreService)
	// CashListExport(cashreportService, movieService, theatreService)
	// TheatreExport(theatreService)
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

func CashExport(service cashreports.UseCase, movieService movieexport.UseCase, theatreService theatreexport.UseCase) {
	// Export cash reports from the past 24 hours
	data, _ := service.Export(time.Now().Add(1 * (-24 * time.Hour)))

	// Loop over all cash reports
	numTotal1 := len(data.Body.ExportResponse.Document.Data.Cashreports.Cashreport)
	for id1, report := range data.Body.ExportResponse.Document.Data.Cashreports.Cashreport {
		fmt.Printf("Working on report %v / %v \n\r", id1, numTotal1)
		log.Printf("Working on report %v / %v \n\r", id1, numTotal1)

		// Loop over all shows in the current cash report
		numTotal := len(report.Shows.Show)
		for id, show := range report.Shows.Show {
			fmt.Printf("Working on show %v / %v \n\r", id, numTotal)
			log.Printf("Working on show %v / %v \n\r", id, numTotal)

			// Extract necessary data from the cash report and create a new DynamicsCashReport object
			movieNum := "" + strings.Split(report.Movie.FullMovieNumber, "-")[2]
			playweekStart, _ := time.Parse("2006-01-02T15:04:05", report.Playweek.StartDate)
			playweekEnd, _ := time.Parse("2006-01-02T15:04:05", report.Playweek.EndDate)
			recordedAmount, _ := strconv.ParseFloat(strings.ReplaceAll(show.TotalCashAmount, ",", "."), 32)
			reportLine := entity.DynamicsCashReport{
				FKBID:           report.Salon.FkbNumber,
				Source:          100000000,
				Playweek:        playweekStart.Format("2006-01-02") + " - " + playweekEnd.Format("2006-01-02"),
				RecordedAmount:  recordedAmount,
				ReportNum:       report.CashreportNumber,
				ShowNum:         id + 1,
				FullMovieNumber: movieNum,
			}

			// Fetch necessary data from Dynamics 365 for the current show
			movies, _ := movieService.FilteredFetchD365("productnumber%20eq%20'" + movieNum + "'")
			theatres, _ := theatreService.FilteredFetchD365("new_fkbid%20eq%20'" + report.Salon.FkbNumber + "'")

			// Set the event, lokal, and account fields in the reportLine object based on the fetched data
			if len(movies) > 0 {
				reportLine.Event = "/products(" + movies[0].ID + ")"
			}
			if len(theatres) > 0 {
				reportLine.Lokal = "/new_lokals(" + theatres[0].LoakalID + ")"
				reportLine.Account = "/accounts(" + theatres[0].AccountData.Accountid + ")"
			}

			// Check if a booking exists for the current show and export it to Dynamics 365 if it does
			showTime, errParseTime := time.Parse("2006-01-02T15:04:05", show.StartDateTime)
			if (errParseTime != nil ) {
			fmt.Printf("showTime: %v \n\r show.StartDateTime: %v \n\r", showTime.String(), show.StartDateTime)
			log.Printf("showTime: %v \n\r show.StartDateTime: %v \n\r", showTime.String(), show.StartDateTime)
			}

			if len(theatres) > 0 && len(movies) > 0 {
				bookings, _ := service.FindBookingD365(
					"_new_customer_value%20eq%20" + theatres[0].AccountData.Accountid +
					"%20and%20_new_product_value%20eq%20" + movies[0].ID +
					"%20and%20new_showdate%20eq%20"  + showTime.Format("2006-01-02"))
			
				if len(bookings) > 0 &&
					shouldLinkBooking(service, bookings[0].ID, report.CashreportNumber) {
			
					// koppla raden till bokningen
					reportLine.Booking = "/new_bokningarkunds(" + bookings[0].ID + ")"
			
					// uppdatera bokningen med lokal
					service.PostToD365(
						"new_bokningarkunds("+bookings[0].ID+")",
						`{"new_Lokaler@odata.bind":"/new_lokals(` + theatres[0].LoakalID + `)"}`)
				}
			}

			// Set the vat-free field in the reportLine object
			reportLine.VatFree = report.Salon.VatFree == "1"
			
			reportLine.ShowDate = showTime // Set the ShowDate field of the DynamicsCashReport object to the parsed show start date and time
			
			// Loop over all ticket details for the current show and create a new DynamicsCashReport object for each one
			for _, ticketDetail := range show.TicketDetails.Detail {
				quantity, _ := strconv.Atoi(ticketDetail.Quantity)
				price, _ := strconv.ParseFloat(strings.ReplaceAll(ticketDetail.Price, ",", "."), 32)

				// Set the necessary fields in the new DynamicsCashReport object
				reportLine.Name = ticketDetail.Category
				reportLine.TicketName = ticketDetail.Category
				reportLine.TicketQuantity = quantity
				reportLine.TicketPrice = price

				// Export the new DynamicsCashReport object to Dynamics 365 as JSON
				jsData, _ := json.Marshal(reportLine)
				service.PostToD365("new_cashreports", string(jsData))
				fmt.Printf("Saving cashreport: %v \n\r", string(jsData))
				log.Printf("Saving cashreport:%v \n\r", string(jsData) )
			}
		}
	}
}

func CashExportWithDate(updatedDate time.Time, service cashreports.UseCase, movieService movieexport.UseCase, theatreService theatreexport.UseCase) {
	data, _ := service.Export(updatedDate)                                          // Export cashreport data based on the given updatedDate
	numTotal1 := len(data.Body.ExportResponse.Document.Data.Cashreports.Cashreport) // Get the total number of cashreports to be processed

	// Iterate through each cashreport and process them
	for id1, report := range data.Body.ExportResponse.Document.Data.Cashreports.Cashreport {
		fmt.Printf("Working on report %v / %v \n\r", id1, numTotal1)
		newCashreport := true
		d365Cashreport, _ := service.FilteredFetchD365("new_cashreportnumber%20eq%20'" + report.CashreportNumber + "'") // Check if Cashreport is already in Dynamics365
		for _, d365Cr := range d365Cashreport {
			if report.CashreportNumber == d365Cr.ReportNum {
				fmt.Println("Report already in dynamics")
				newCashreport = false // If the cashreport is already in Dynamics365, set the newCashreport flag to false
			}
		}
		if newCashreport {
			numTotal := len(report.Shows.Show) // Get the total number of shows in the cashreport
			for id, show := range report.Shows.Show {
				fmt.Printf("Working on show %v / %v \n\r", id, numTotal)
				movieNum := "" + strings.Split(report.Movie.FullMovieNumber, "-")[2]                            // Extract the movie number from the FullMovieNumber field
				playweekStart, _ := time.Parse("2006-01-02T15:04:05", report.Playweek.StartDate)                // Parse the start date of the playweek
				playweekEnd, _ := time.Parse("2006-01-02T15:04:05", report.Playweek.EndDate)                    // Parse the end date of the playweek
				recordedAmount, _ := strconv.ParseFloat(strings.ReplaceAll(show.TotalCashAmount, ",", "."), 32) // Parse the total cash amount for the show
				reportLine := entity.DynamicsCashReport{                                                        // Create a new DynamicsCashReport object
					FKBID:           report.Salon.FkbNumber,
					Source:          100000000,
					Playweek:        playweekStart.Format("2006-01-02") + " - " + playweekEnd.Format("2006-01-02"),
					RecordedAmount:  recordedAmount,
					ReportNum:       report.CashreportNumber,
					ShowNum:         id + 1,
					FullMovieNumber: movieNum,
				}

				movies, _ := movieService.FilteredFetchD365("productnumber%20eq%20'" + movieNum + "'")               // Fetch movies from Dynamics365 based on the movie number
				theatres, _ := theatreService.FilteredFetchD365("new_fkbid%20eq%20'" + report.Salon.FkbNumber + "'") // Fetch theatres from Dynamics365 based on the FKBID

				// Set the Event, Lokal, and Account fields of the DynamicsCashReport object based on the fetched movies and theatres
				if len(movies) > 0 {
					reportLine.Event = "/products(" + movies[0].ID + ")"
				}
				if len(theatres) > 0 {
					reportLine.Lokal = "/new_lokals(" + theatres[0].LoakalID + ")"
					reportLine.Account = "/accounts(" + theatres[0].AccountData.Accountid + ")"
				}

				showTime, _ := time.Parse("2006-01-02T15:04:05", show.StartDateTime) // Parse the start date and time of the show

				if len(theatres) > 0 && len(movies) > 0 {
					bookings, _ := service.FindBookingD365(
						"_new_customer_value%20eq%20" + theatres[0].AccountData.Accountid +
						"%20and%20_new_product_value%20eq%20" + movies[0].ID +
						"%20and%20new_showdate%20eq%20"  + showTime.Format("2006-01-02"))
				
					if len(bookings) > 0 &&
						shouldLinkBooking(service, bookings[0].ID, report.CashreportNumber) {
				
						// koppla raden till bokningen
						reportLine.Booking = "/new_bokningarkunds(" + bookings[0].ID + ")"
				
						// uppdatera bokningen med lokal
						service.PostToD365(
							"new_bokningarkunds("+bookings[0].ID+")",
							`{"new_Lokaler@odata.bind":"/new_lokals(` + theatres[0].LoakalID + `)"}`)
					}
				}

				reportLine.VatFree = report.Salon.VatFree == "1" // Set the VatFree field of the DynamicsCashReport object based on the VatFree field of the salon in the cashreport

				reportLine.ShowDate = showTime // Set the ShowDate field of the DynamicsCashReport object to the parsed show start date and time

				// Loop through each ticket detail in the show and set the ticket-related fields of the DynamicsCashReport object
				for _, ticketDetail := range show.TicketDetails.Detail {
					quantity, _ := strconv.Atoi(ticketDetail.Quantity)
					price, _ := strconv.ParseFloat(strings.ReplaceAll(ticketDetail.Price, ",", "."), 32)

					reportLine.Name = ticketDetail.Category
					reportLine.TicketName = ticketDetail.Category
					reportLine.TicketQuantity = quantity
					reportLine.TicketPrice = price

					// Export the DynamicsCashReport object to JSON and send it to Dynamics365 for each ticket
					jsData, _ := json.Marshal(reportLine)
					service.PostToD365("new_cashreports", string(jsData))
				}
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

// CashListExport exports and processes cash list data for a given date range, with added error handling.
func CashListExport(service cashreports.UseCase, movieService movieexport.UseCase, theatreService theatreexport.UseCase) {
	fmt.Printf("CashListExport") // Initial log message indicating the start of the export process.
	// Define the start and end dates for the export.
	startDate := time.Date(2024, 4, 04, 0, 0, 0, 0, time.Local)
	endDate := time.Date(2024, 4, 15, 23, 59, 59, 99, time.Local)
	// Export the list of cash reports within the defined date range.
	data, err := service.ExportList(startDate, endDate)
	if err != nil {
		log.Printf("Error exporting list: %v\n", err)
		return // Exit the function if there is an error.
	}

	// Get the total number of dates in the cash report list data.
	numTotal1 := len(data.Body.ExportResponse.Document.Data.Dates.Date)
	// Iterate through each date in the cash report data.
	for id1, report := range data.Body.ExportResponse.Document.Data.Dates.Date {
		// Log the current date being processed.
		fmt.Printf("Working on date %v : %v / %v \n\r", report.UpdatedDate, id1, numTotal1)
		log.Printf("Working on date %v : %v / %v \n\r", report.UpdatedDate, id1, numTotal1)
		// Parse the date from the report.
		d, parseErr := time.Parse("2006-01-02T15:04:05", report.UpdatedDate+"T00:00:00")
		if parseErr != nil {
			log.Printf("Error parsing date %v: %v\n", report.UpdatedDate, parseErr)
			continue // Skip this iteration if there's an error parsing the date.
		}
		// Process and export cash data for the parsed date.
		CashExportWithDate(d, service, movieService, theatreService)
		// Update cash reports for the parsed date.
		UpdateCashreports(d, service, movieService, theatreService)
	}
}

// shouldLinkBooking tells if the current cash-report row may link to a booking.
//
// Link is allowed when
//   • the booking has no linked rows, or
//   • every linked row (we read the first) shares the same cash-report number.
//
// When a mismatch is found we print and log a note, then refuse the link.
//
// Params
//   s         – service that talks to Dynamics 365  
//   bookingID – booking GUID, no braces or quotes  
//   reportNum – cash-report number of the row we process
//
// Returns
//   true  – link is allowed  
//   false – link is denied
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