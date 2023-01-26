package handler

import (
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"fhpbioguide/pkg/entity"
	"fhpbioguide/pkg/usecase/cashreports"
	"fhpbioguide/pkg/usecase/movieexport"
	"fhpbioguide/pkg/usecase/theatreexport"
)

func ExecuteExports(movieService movieexport.UseCase, cashreportService cashreports.UseCase, theatreService theatreexport.UseCase) {
	MovieExport(movieService)
	//TheatreExport(theatreService)
	CashExport(cashreportService, movieService, theatreService)
	//CashListExport(cashreportService, movieService, theatreService)
}

func MovieExport(service movieexport.UseCase) {
	// Fetch movies updated between yesterday and today
	data, _ := service.Export(time.Date(2018, time.January, 1, 0, 0, 0, 0, time.Local), time.Date(2030, time.December, 31, 0, 0, 0, 0, time.Local))

	movies, _ := service.FetchFromD365()
	numTotal := len(data.Body.ExportResponse.Document.Data.Movies.Movie)
	for id, movie := range data.Body.ExportResponse.Document.Data.Movies.Movie {
		fmt.Printf("Working on movie %v/%v \n", id, numTotal)
		if movie.Distributor.Name == "Folkets Hus och Parker" {
			newMovie := true
			// Check if movie is already in dynamics 365
			for _, d365Movie := range movies {
				if strings.Split(movie.FullMovieNumber, "-")[2] == d365Movie.Productnumber {
					fmt.Println("Movie already in dynamics")
					newMovie = false
				}
			}
			if !newMovie {
				continue
			}

			premereDate, _ := time.Parse("2006-01-02T15:04:05", movie.PremiereDate)
			if premereDate.Before(time.Date(1900, time.January, 1, 0, 0, 0, 0, time.Local)) {
				fmt.Println("Skipping movie has PremiereDate before 1900")
				continue
			}
			runtime, _ := strconv.Atoi(movie.Runtime)
			item := entity.Product{
				Name:        movie.Title,
				Description: movie.Description,
				//Status:           false,
				Vendorname:       movie.Distributor.Name,
				Length:           runtime,
				Distributor:      100000000,
				Censur:           censurToID(movie.Rating),
				Productnumber:    strings.Split(movie.FullMovieNumber, "-")[2],
				Producttypecode:  100000001,
				Productstructure: 1,
				Premier:          premereDate,
				ShowdateStart:    premereDate,
				Textning:         movie.Subtitles == "Svenska",

				// Dynamics standard required column values
				UnitScheduleID: "/uomschedules(77267707-68dc-4f61-be8a-b29394aeffb2)",
				UnitID:         "/uoms(38b503b0-94cb-4a79-ab8f-1fa060e1e5e2)",
			}

			// Export to Json and send to Dynamics 365
			jsData, _ := json.Marshal(item)
			service.PostToD365(string(jsData))
		}
	}
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

func TheatreExport(service theatreexport.UseCase) {
	data, err := service.Export("")

	if err != nil {
		fmt.Println(err.Error())
	}

	d365Lokals, _ := service.FetchFromD365()

	numTotal := len(data.Body.ExportResponse.Document.Data.Theatres.Theatre)
	for id, theatre := range data.Body.ExportResponse.Document.Data.Theatres.Theatre {
		fmt.Printf("Working on theatre %v/%v \n", id, numTotal)

		numTotal2 := len(theatre.Salons.Salon)
		for id, salon := range theatre.Salons.Salon {
			fmt.Printf("Working on salon %v/%v \n", id, numTotal2)
			newLocation := true
			// Check if salon is already in dynamics 365
			for _, d365Salon := range d365Lokals {
				if salon.FkbNumber == d365Salon.FkbNum {
					fmt.Println("Salon already in dynamics")
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
				// Set correct buisness unit in dynamics 365 for the record
				BusinessUnit: "/businessunits(d7e6eb96-d458-ec11-8f8f-6045bd8aa5cb)",
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
			// Export to Json and send to Dynamics 365
			jsData2, _ := json.Marshal(location)
			service.PostToD365("new_lokals", string(jsData2))
		}
	}
}

func CashExport(service cashreports.UseCase, movieService movieexport.UseCase, theatreService theatreexport.UseCase) {
	data, _ := service.Export(time.Now().Add(1 * (-24 * time.Hour)))
	numTotal1 := len(data.Body.ExportResponse.Document.Data.Cashreports.Cashreport)
	for id1, report := range data.Body.ExportResponse.Document.Data.Cashreports.Cashreport {
		fmt.Printf("Working on report %v / %v \n", id1, numTotal1)
		log.Printf("Working on report %v / %v \n", id1, numTotal1)
		numTotal := len(report.Shows.Show)
		for id, show := range report.Shows.Show {
			fmt.Printf("Working on show %v / %v \n", id, numTotal)
			log.Printf("Working on show %v / %v \n", id, numTotal)
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
				bookings, _ := service.FindBookingD365("_new_customer_value%20eq%20" + theatres[0].AccountData.Accountid + "%20and%20_new_product_value%20eq%20" + movies[0].ID + "%20and%20" + "new_showdate%20eq%20" + showTime.Format("2006-01-02"))
				if len(bookings) > 0 {
					reportLine.Booking = "/new_bokningarkunds(" + bookings[0].ID + ")"
					service.PostToD365("new_bokningarkunds("+bookings[0].ID+")", `{"new_Lokaler@odata.bind":"/new_lokals(`+theatres[0].LoakalID+`)"}`)
				}
			}
			reportLine.VatFree = report.Salon.VatFree == "1"

			reportLine.ShowDate = showTime

			// Loop tickets and set name, quantity, price and keep rest of data from show intact
			for _, ticketDetail := range show.TicketDetails.Detail {
				quantity, _ := strconv.Atoi(ticketDetail.Quantity)
				price, _ := strconv.ParseFloat(strings.ReplaceAll(ticketDetail.Price, ",", "."), 32)

				reportLine.Name = ticketDetail.Category
				reportLine.TicketName = ticketDetail.Category
				reportLine.TicketQuantity = quantity
				reportLine.TicketPrice = price

				// Export to Json and send to Dynamics 365 for each ticket
				jsData, _ := json.Marshal(reportLine)
				service.PostToD365("new_cashreports", string(jsData))
			}
		}
	}

}

func CashExportWithDate(updatedDate time.Time, service cashreports.UseCase, movieService movieexport.UseCase, theatreService theatreexport.UseCase) {
	data, _ := service.Export(updatedDate)
	numTotal1 := len(data.Body.ExportResponse.Document.Data.Cashreports.Cashreport)
	for id1, report := range data.Body.ExportResponse.Document.Data.Cashreports.Cashreport {
		fmt.Printf("Working on report %v / %v \n", id1, numTotal1)
		newCashreport := true
		d365Cashreport, _ := service.FilteredFetchD365("new_cashreportnumber%20eq%20'" + report.CashreportNumber + "'")
		// Check if Cashreport is already in Dynamics365
		for _, d365Cr := range d365Cashreport {
			if report.CashreportNumber == d365Cr.ReportNum {
				fmt.Println("Report already in dynamics")
				newCashreport = false
			}
		}
		if newCashreport {

			numTotal := len(report.Shows.Show)
			for id, show := range report.Shows.Show {
				fmt.Printf("Working on show %v / %v \n", id, numTotal)
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
					bookings, _ := service.FindBookingD365("_new_customer_value%20eq%20" + theatres[0].AccountData.Accountid + "%20and%20_new_product_value%20eq%20" + movies[0].ID + "%20and%20" + "new_showdate%20eq%20" + showTime.Format("2006-01-02"))
					if len(bookings) > 0 {
						reportLine.Booking = "/new_bokningarkunds(" + bookings[0].ID + ")"
						service.PostToD365("new_bokningarkunds("+bookings[0].ID+")", `{"new_Lokaler@odata.bind":"/new_lokals(`+theatres[0].LoakalID+`)"}`)
					}
				}
				reportLine.VatFree = report.Salon.VatFree == "1"

				reportLine.ShowDate = showTime

				// Loop tickets and set name, quantity, price and keep rest of data from show intact
				for _, ticketDetail := range show.TicketDetails.Detail {
					quantity, _ := strconv.Atoi(ticketDetail.Quantity)
					price, _ := strconv.ParseFloat(strings.ReplaceAll(ticketDetail.Price, ",", "."), 32)

					reportLine.Name = ticketDetail.Category
					reportLine.TicketName = ticketDetail.Category
					reportLine.TicketQuantity = quantity
					reportLine.TicketPrice = price

					// Export to Json and send to Dynamics 365 for each ticket
					jsData, _ := json.Marshal(reportLine)
					service.PostToD365("new_cashreports", string(jsData))
				}
			}
		}
	}

}

func UpdateCashreports(updatedDate time.Time, service cashreports.UseCase, movieService movieexport.UseCase, theatreService theatreexport.UseCase) {
	data, _ := service.Export(updatedDate)
	numTotal1 := len(data.Body.ExportResponse.Document.Data.Cashreports.Cashreport)
	for id1, report := range data.Body.ExportResponse.Document.Data.Cashreports.Cashreport {
		fmt.Printf("Working on report %v / %v \n", id1, numTotal1)
		d365Cashreport, _ := service.FilteredFetchD365("new_source%20eq%20100000000%20and%20new_cashreportnumber%20eq%20'" + report.CashreportNumber + "'")
		// Check if Cashreport is already in Dynamics365
		for _, d365Cr := range d365Cashreport {
			if report.CashreportNumber == d365Cr.ReportNum {
				fmt.Println("Report " + d365Cr.ReportNum + "found in dynamics")
				movieNum := "" + strings.Split(report.Movie.FullMovieNumber, "-")[2]
				service.PostToD365("new_cashreports("+d365Cr.ID+")", `{"new_fullmovienumber":"`+movieNum+`"}`)
			}
		}
	}
}

func CashListExport(service cashreports.UseCase, movieService movieexport.UseCase, theatreService theatreexport.UseCase) {
	fmt.Printf("CashListExport")
	startDate := time.Date(2022, 5, 1, 0, 0, 0, 0, time.Local)
	endDate := time.Date(2022, 6, 15, 23, 59, 59, 99, time.Local)
	data, _ := service.ExportList(startDate, endDate)

	numTotal1 := len(data.Body.ExportResponse.Document.Data.Dates.Date)
	for id1, report := range data.Body.ExportResponse.Document.Data.Dates.Date {
		fmt.Printf("Working on date %v : %v / %v \n", report.UpdatedDate, id1, numTotal1)
		d, _ := time.Parse("2006-01-02T15:04:05", report.UpdatedDate+"T00:00:00")
		//CashExportWithDate(d, service, movieService, theatreService)
		UpdateCashreports(d, service, movieService, theatreService)
	}
}
