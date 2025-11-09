package handler

import (
	"bytes"
	"encoding/json"
	"fmt"

	//"html"
	"log"
	"net/http"
	"strconv"
	"time"

	"fhpbioguide/pkg/entity"
	"fhpbioguide/pkg/usecase/reportform"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/spf13/viper"
)

// Payload represents the data structure sent as JSON in the POST request
type Payload struct {
	Booking  string `json:"Booking"`
	FormType string `json:"FormType"`
}

func MakeReportForms(app *fiber.App, service reportform.UseCase) {
	app.Get("/form/:ID", getForm(service))
	app.Post("/form-post/:ID", postFormResult(service))
	app.Get("/api/status", getStatus())
	app.Post("/api/genform/presale/:ID", createPresaleForm(service))
	app.Post("/api/genform/sold/:ID", createSoldForm(service))
	app.Post("/api/regenform/sold/:ID", recreateSoldForm(service))
	app.Post("/api/orderstatus", proxyRequest())
	//app.Post("/api/importcashreports/:DATE", importCashreports())
}

func getStatus() fiber.Handler {
	return func(c *fiber.Ctx) error {
		log.Printf("Sending status")
		return c.SendStatus(http.StatusOK)
	}
}

// proxyRequest returns a Fiber handler function for proxying HTTP requests.
func proxyRequest() fiber.Handler {
	// The returned function captures the Fiber context.
	return func(c *fiber.Ctx) error {
		log.Printf("Receiving order status from Movie Transit")

		// Validate Authorization header.
		// This checks the request's Authorization header against an expected value from the application's configuration.
		authHeader := c.Get("Authorization")
		expectedAuth := viper.GetString("proxy.Bearer") // viper is used for configuration management.
		if authHeader != expectedAuth {
			log.Printf("Unauthorized access attempt")
			// If the Authorization header doesn't match the expected value, return an HTTP 401 Unauthorized status code.
			return c.SendStatus(http.StatusUnauthorized)
		}

		// Retrieve the proxy URL from the application's configuration.
		url := viper.GetString("proxy.URL")

		// Create a new buffer with the raw body of the incoming request to forward it.
		reqBody := bytes.NewBuffer(c.BodyRaw())

		// Send the POST request to the proxied service.
		resp, err := http.Post(url, "application/json", reqBody)
		if err != nil {
			// Log the error and return an HTTP 502 Bad Gateway status code if the request to the proxied service fails.
			log.Printf("Failed to send POST request: %v", err)
			return c.Status(http.StatusBadGateway).SendString("Failed to proxy request")
		}
		// Ensure the response body is closed to prevent resource leaks.
		defer func() {
			if resp != nil {
				resp.Body.Close()
			}
		}()

		// Check if the status code from the proxied service is not OK (200).
		if resp.StatusCode != http.StatusOK {
			log.Printf("Received non-OK status from proxied service: %d", resp.StatusCode)
			// Forward the response status code from the proxied service and return an error message.
			return c.Status(resp.StatusCode).SendString("Proxied service responded with error")
		}

		// If the proxied request was successful, return an HTTP 200 OK status code.
		return c.SendStatus(http.StatusOK)
	}
}

func getForm(service reportform.UseCase) fiber.Handler {
	return func(c *fiber.Ctx) error {
		errorMessage := "Error reading form"
		formID, err := entity.StringToID(c.Params("ID"))
		if err != nil {
			c.Status(http.StatusNotFound).SendString(errorMessage)
			return c.Render("error", fiber.Map{
				"Title": "FHP - Id fel",
				"Error": errorMessage,
			})
		}
		form, err := service.GetForm(formID)
		// Check if form is valid
		if err != nil || form.ID == uuid.Nil {
			c.Status(http.StatusNotFound).SendString(errorMessage)
			return c.Render("error", fiber.Map{
				"Title": "FHP - Fel vid inläsningg av formulär",
				"Error": errorMessage,
			})
		}

		// Check what template file to use
		template := "presale"
		if form.Type == 1 {
			template = "sold"
		}

		return c.Render(template, fiber.Map{
			"Title":  form.Name,
			"Desc":   form.Text,
			"ID":     form.ID.String(),
			"Events": form.Events,
		})

	}
}

func createPresaleForm(service reportform.UseCase) fiber.Handler {
	return func(c *fiber.Ctx) error {
		authToken := c.GetReqHeaders()
		// Quick and easy auth token check
		if authToken["Authorization"][0] != viper.GetString("report.Bearer") {
			return c.SendStatus(http.StatusForbidden)
		}
		customerID, err := entity.StringToID(c.Params("ID"))
		if err != nil {
			c.Status(http.StatusNotFound)
			return c.Render("error", fiber.Map{
				"Title": "FHP - Fel vid inläsningg av url",
				"Error": "Could not found Customer",
			})
		}

		/*contactID, err := entity.StringToID(c.Params("CID"))
		if err != nil {
			c.Status(http.StatusNotFound)
			return c.Render("error", fiber.Map{
				"Title": "FHP - Fel vid inläsningg av url",
				"Error": "Could not found Contact",
			})
		}*/

		formID := entity.NewID()
		bookings := entity.Bookings{}
		// Fetch all bookings that is marked presale from the custommer
		data, _ := service.GetFromD365("new_bokningarkunds?$filter=_new_customer_value%20eq%20" + customerID.String() + "%20and%20new_state%20eq%20" + "100000000" + "%20and%20new_presales%20eq%20" + "true" + "&$expand=new_customer_account($select=name),new_product($select=name)")
		/* data, _ := service.GetFromD365("new_bokningarkunds?$filter=_new_customer_value%20eq%20" + customerID.String() + "%20and%20new_state%20eq%20" + "100000000" + "%20and%20new_presales%20eq%20" + "true" + "%20and%20n(_new_kontakt_value%20eq%20" + contactID.String() + "%20or%20_new_forkopskontakt_value%20eq%20" + contactID.String() + ")&$expand=new_customer_account($select=name),new_product($select=name)")*/
		err = json.Unmarshal(data, &bookings)
		if err != nil {
			fmt.Println(err.Error())
		}
		form := &entity.Form{
			ID:   formID,
			Name: bookings.Value[0].Customer.Name,
			Text: "Rapportera totala förköp för nedanstående evenemang",
			Date: time.Now(),
			Type: 0,
		}

		// Append bookings to form Events array
		for _, item := range bookings.Value {
			form.Events = append(form.Events, entity.Event{
				ID:             entity.UnsafeStringToID(item.ID),
				Form_type:      0,
				Name:           item.Product.Name,
				Text:           item.Name,
				Date:           item.ShowDate,
				Amount:         0,
				Url:            item.Url,
				ExpirationTime: time.Now().Add(24 * time.Hour),
				FormID:         formID,
				Discounts:      item.Discounts,
			})
		}

		// Create form in repository
		service.Create(form)

		// Return form url
		return c.SendString("https://" + viper.GetString("report.url") + "/form/" + formID.String())
	}
}

func createSoldForm(service reportform.UseCase) fiber.Handler {
	return func(c *fiber.Ctx) error {
		log.Printf("Generate sold form")
		authToken := c.GetReqHeaders()
		// Quick and easy auth token check
		if authToken["Authorization"][0] != viper.GetString("report.Bearer") {
			log.Printf("Not Authurized")
			return c.SendStatus(http.StatusForbidden)
		}
		customerID, err := entity.StringToID(c.Params("ID"))
		if err != nil {
			log.Printf("No ID parameter")
			c.Status(http.StatusNotFound)
			return err
		}

		formID := entity.NewID()
		bookings := entity.Bookings{}
		log.Printf("Fetch all bookings that is marked complete not invoiced yet from the custommer.")
		data, _ := service.GetFromD365("new_bokningarkunds?$filter=_new_customer_value%20eq%20" + customerID.String() + "%20and%20new_state%20eq%20" + "100000001" + "%20and%20new_slutredovisning%20eq%20true%20and%20new_showdate%20eq%20" + time.Now().Format("2006-01-02") + "&$expand=new_customer_account($select=name),new_product($select=name)")
		err = json.Unmarshal(data, &bookings)
		if err != nil {
			log.Println(err.Error())
			return err
		}
		log.Printf("Add data to form to create new form")
		form := &entity.Form{
			ID:   formID,
			Name: bookings.Value[0].Customer.Name,
			Text: "Rapportera biljetter sålda för nedanstående evenemang",
			Date: time.Now(),
			Type: 1,
		}

		log.Printf("Append bookings to form Events array")
		for _, item := range bookings.Value {
			form.Events = append(form.Events, entity.Event{
				ID:             entity.UnsafeStringToID(item.ID),
				Form_type:      1,
				Name:           item.Product.Name,
				Text:           item.Name,
				Date:           item.ShowDate,
				Amount:         0,
				ExpirationTime: time.Now().Add(24 * time.Hour),
				FormID:         formID,
			})
		}

		log.Printf("Create form in repository")
		service.Create(form)

		log.Printf("Return form url")
		return c.SendString("https://" + viper.GetString("report.url") + "/form/" + formID.String())
	}
}

func recreateSoldForm(service reportform.UseCase) fiber.Handler {
	return func(c *fiber.Ctx) error {
		log.Printf("Regenerate sold form")
		authToken := c.GetReqHeaders()
		// Quick and easy auth token check
		if authToken["Authorization"][0] != "Bearer eyJ0eXAiOiJKV1QiLCJhbGciOiJSUzI1NiIsIng1dCI6ImpTMVhvMU9XRGpfNTJ2YndHTmd2UU8yVnpNYyIsImtpZCI6ImpTMVhvMU9XRGpfNTJ2YndHTmd2UU8yVnpNYyJ9.eyJhdWQiOiJodHRwczovL2ZvbGtldHNodXNvY2hwYXJrZXIuY3JtNC5keW5hbWljcy5jb20iLCJpc3MiOiJodHRwczovL3N0cy53aW5kb3dzLm5ldC9hOWI5NmE4OC00NWY1LTRmNWEtYTA5Yy01ZmM1NGQ5ZGVhOTQvIiwiaWF0IjoxNjQ2NzMxMzI1LCJuYmYiOjE2NDY3MzEzMjUsImV4cCI6MTY0NjczNTIyNSwiYWlvIjoiRTJaZ1lDamJaTCs5OUxhK2VkOW5WK1BIbFJiOUFBPT0iLCJhcHBpZCI6IjAwYjcyNWU0LTEwNzEtNDQ1NC05NThiLWM5MGZlZTg2ZTY5OCIsImFwcGlkYWNyIjoiMSIsImlkcCI6Imh0dHBzOi8vc3RzLndpbmRvd3MubmV0L2E5Yjk2YTg4LTQ1ZjUtNGY1YS1hMDljLTVmYzU0ZDlkZWE5NC8iLCJvaWQiOiJkZmYyYTc3OC01NjBkLTQ1NzQtYTVhMi1kMGYzNDI0Mjg3MDgiLCJyaCI6IjAuQVRrQWlHcTVxZlZGV2stZ25GX0ZUWjNxbEFjQUFBQUFBQUFBd0FBQUFBQUFBQUE1QUFBLiIsInN1YiI6ImRmZjJhNzc4LTU2MGQtNDU3NC1hNWEyLWQwZjM0MjQyODcwOCIsInRpZCI6ImE5Yjk2YTg4LTQ1ZjUtNGY1YS1hMDljLTVmYzU0ZDlkZWE5NCIsInV0aSI6Im5zRS1ua3BqMFV1Rk1haldMTTQ5QUEiLCJ2ZXIiOiIxLjAifQ.hQv7UTxJvDfXWsDBp4p_e038pZqSBXQWqBTGpVLX-nptBy9bRX63-4z3yugrarA7SfydZCg6cEZsOlNfp_9DJPZ1jnnPR72JlY1hmvUtwyVFziX4o2-pQE9dwfpGcy1ai1p1ZfMjCqrLaLb8R5pZdIY1PnjPjOlboeHDGoV1qjr0-5P6Z9jKGJGNrYzg3Lze0KTPqZytteilZuIQ6XDZbwgt_gumFgyjGywBmWl1rke5k7wEoCtnx_aZKh49xSHEYuLGue4hZbVSCOzfztIo2XAWqKICKPPc6QO4VEca-9m3-YMjgeEJOzmP5McwqtFvdLYF9mGXfY5qgPtn4JjtAA" {
			log.Printf("Not Authurized")
			return c.SendStatus(http.StatusForbidden)
		}
		bID, err := entity.StringToID(c.Params("ID"))
		if err != nil {
			log.Printf("No ID for booking")
			c.Status(http.StatusNotFound)
			return err
		}

		b, err := service.GetEvent(bID)
		// Check if booking is valid
		if err != nil {
			return err
		}
		if b.ID != uuid.Nil {
			// Return form url
			log.Printf("Return allready created form")
			return c.SendString("https://" + viper.GetString("report.url") + "/form/" + b.FormID.String())
		}

		formID := entity.NewID()
		booking := entity.Booking{}
		log.Printf("Fetch booking")
		data, _ := service.GetFromD365("new_bokningarkunds(" + bID.String() + ")?$expand=new_customer_account($select=name),new_product($select=name)")
		err = json.Unmarshal(data, &booking)
		if err != nil {
			log.Println(err.Error())
			return err
		}

		log.Printf("Add data to form to create new form")
		form := &entity.Form{
			ID:   formID,
			Name: booking.Customer.Name,
			Text: "Rapportera biljetter sålda för nedanstående evenemang",
			Date: time.Now(),
			Type: 1,
		}

		log.Printf("Append bookings to form Events array")
		form.Events = append(form.Events, entity.Event{
			ID:             entity.UnsafeStringToID(booking.ID),
			Form_type:      1,
			Name:           booking.Product.Name,
			Text:           booking.Name,
			Date:           booking.ShowDate,
			Amount:         0,
			ExpirationTime: time.Now().Add(24 * time.Hour),
			FormID:         formID,
		})

		log.Printf("Create form in repository")
		service.Create(form)

		log.Printf("Return form url")
		return c.SendString("https://" + viper.GetString("report.url") + "/form/" + formID.String())
	}
}

func postFormResult(service reportform.UseCase) fiber.Handler {
	return func(c *fiber.Ctx) error {
		errorMessage := "Error reading form"
		formID, err := entity.StringToID(c.Params("ID"))
		// check if ID is defined
		if err != nil {
			log.Printf("For does not exist: %s", formID.String())
			c.Status(http.StatusNotFound).SendString(errorMessage)
			return err
		}
		form, err := service.GetForm(formID)
		// check if form was found or not
		if err != nil || form.ID == uuid.Nil {
			log.Printf("Can't get form: %s", formID.String())
			c.Status(http.StatusNotFound).SendString(errorMessage)
			return err
		}

		if form.Type == 0 {
			for id, event := range form.Events {
				service.PostToD365("new_forkops", `{"new_boking@odata.bind":"new_bokningarkunds(`+event.ID.String()+`)","new_forkopsurl":"`+c.FormValue("10_"+strconv.Itoa(id))+`","new_unit":`+c.FormValue("00_"+strconv.Itoa(id), "0")+`}`)
				// Define the API endpoint
				url := "https://dcf5d3602d82484caa8f70f597e2c3.59.environment.api.powerplatform.com:443/powerautomate/automations/direct/workflows/26b97772aeb345009eecf08a0e8ed059/triggers/manual/paths/invoke?api-version=1&sp=%2Ftriggers%2Fmanual%2Frun&sv=1.0&sig=-Q28UL28_Y4BiYQm4Y9hXuz7jKFmZVDbUkXvgqsqs_Y"

				// Create the payload with the required data
				payload := Payload{
					Booking:  event.ID.String(),
					FormType: strconv.Itoa(form.Type),
				}

				// Marshal the payload into JSON format
				jsonData, err := json.Marshal(payload)
				if err != nil {
					log.Printf("Error marshalling JSON: %v", err)
				}

				// Send the POST request with the JSON data
				resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
				if err != nil {
					log.Printf("Error sending POST request: %v", err)
				}
				defer resp.Body.Close() // Ensure the response body is closed after the function returns
			}
		} else {
			for id, event := range form.Events {
				booking := entity.Booking2{}
				data, _ := service.GetFromD365("new_bokningarkunds(" + event.ID.String() + ")")
				json.Unmarshal(data, &booking)

				reportLine := entity.DynamicsCashReport{
					Source:    100000001,
					ReportNum: booking.Bookingnumber,
					Account:   "/accounts(" + booking.CustomerID + ")",
					Booking:   "/new_bokningarkunds(" + event.ID.String() + ")",
				}
				account := entity.Account{}
				data3, _ := service.GetFromD365("accounts(" + booking.CustomerID + ")")
				json.Unmarshal(data3, &account)
				reportLine.VatFree = account.VatFree
				reportLine.TransactionCurrencyId = "/transactioncurrencies(" + account.TransactionCurrencyId + ")"

				showDate, _ := time.Parse("2006-01-02", booking.Showdate)
				reportLine.ShowDate = showDate

				if booking.ProductID != "" {
					reportLine.Event = "/products(" + booking.ProductID + ")"
				}
				if booking.LokalID != "" {
					lokal := entity.LokalDynamics{}
					data2, _ := service.GetFromD365("new_lokals(" + booking.LokalID + ")")
					json.Unmarshal(data2, &lokal)
					reportLine.FKBID = lokal.FkbNum
					reportLine.Lokal = "/new_lokals(" + booking.LokalID + ")"
				}

				standardTicket := c.FormValue("00_"+strconv.Itoa(id), "0")
				standardPrice := c.FormValue("01_"+strconv.Itoa(id), "")
				freeTicket := c.FormValue("10_"+strconv.Itoa(id), "0")
				kidsTicket := c.FormValue("20_"+strconv.Itoa(id), "0")
				kidsPrice := c.FormValue("21_"+strconv.Itoa(id), "")
				playsTicket := c.FormValue("30_"+strconv.Itoa(id), "0")
				playsPrice := c.FormValue("31_"+strconv.Itoa(id), "")
				scenTicket := c.FormValue("40_"+strconv.Itoa(id), "0")
				scenPrice := c.FormValue("41_"+strconv.Itoa(id), "")
				konstTicket := c.FormValue("50_"+strconv.Itoa(id), "0")
				konstPrice := c.FormValue("51_"+strconv.Itoa(id), "")
				metTicket := c.FormValue("80_"+strconv.Itoa(id), "0")
				metPrice := c.FormValue("81_"+strconv.Itoa(id), "")
				otherTicket := c.FormValue("60_"+strconv.Itoa(id), "0")
				otherPrice := c.FormValue("61_"+strconv.Itoa(id), "")
				other2Ticket := c.FormValue("70_"+strconv.Itoa(id), "0")
				other2Price := c.FormValue("71_"+strconv.Itoa(id), "")

				if standardTicket == "0" && freeTicket == "0" && kidsTicket == "0" && playsTicket == "0" && scenTicket == "0" && konstTicket == "0" && metTicket == "0" && otherTicket == "0" && other2Ticket == "0" {
					reportLine.Name = "Ordinarie"
					reportLine.TicketName = "Ordinarie"
					reportLine.TicketPrice = 0
					reportLine.TicketQuantity = 0
					// Export to Json and send to Dynamics 365 for each ticket
					jsData, _ := json.Marshal(reportLine)
					service.PostToD365("new_cashreports", string(jsData))

				}
				if standardTicket != "0" {
					reportLine.Name = "Ordinarie"
					reportLine.TicketName = "Ordinarie"
					price, _ := strconv.ParseFloat(standardPrice, 64)
					amount, _ := strconv.Atoi(standardTicket)
					reportLine.TicketPrice = price
					reportLine.TicketQuantity = amount
					// Export to Json and send to Dynamics 365 for each ticket
					jsData, _ := json.Marshal(reportLine)
					service.PostToD365("new_cashreports", string(jsData))

				}
				if freeTicket != "0" {
					reportLine.Name = "Fribiljett"
					reportLine.TicketName = "Fribiljett"
					amount, _ := strconv.Atoi(freeTicket)
					reportLine.TicketPrice = 0
					reportLine.TicketQuantity = amount
					// Export to Json and send to Dynamics 365 for each ticket
					jsData, _ := json.Marshal(reportLine)
					service.PostToD365("new_cashreports", string(jsData))

				}
				if kidsTicket != "0" {
					reportLine.Name = "Barn/Ungdom under 26 år: 25%"
					reportLine.TicketName = "Barn/Ungdom under 26 år: 25%"
					price, _ := strconv.ParseFloat(kidsPrice, 64)
					amount, _ := strconv.Atoi(kidsTicket)
					reportLine.TicketPrice = price
					reportLine.TicketQuantity = amount
					// Export to Json and send to Dynamics 365 for each ticket
					jsData, _ := json.Marshal(reportLine)
					service.PostToD365("new_cashreports", string(jsData))

				}
				if playsTicket != "0" {
					reportLine.Name = "Abonnemang på minst 5 föreställningar: 25%"
					reportLine.TicketName = "Abonnemang på minst 5 föreställningar: 25%"
					price, _ := strconv.ParseFloat(playsPrice, 64)
					amount, _ := strconv.Atoi(playsTicket)
					reportLine.TicketPrice = price
					reportLine.TicketQuantity = amount
					// Export to Json and send to Dynamics 365 for each ticket
					jsData, _ := json.Marshal(reportLine)
					service.PostToD365("new_cashreports", string(jsData))

				}
				if scenTicket != "0" {
					reportLine.Name = "Scenpass Sverige: 10%"
					reportLine.TicketName = "Scenpass Sverige: 10%"
					price, _ := strconv.ParseFloat(scenPrice, 64)
					amount, _ := strconv.Atoi(scenTicket)
					reportLine.TicketPrice = price
					reportLine.TicketQuantity = amount
					// Export to Json and send to Dynamics 365 for each ticket
					jsData, _ := json.Marshal(reportLine)
					service.PostToD365("new_cashreports", string(jsData))

				}
				if konstTicket != "0" {
					reportLine.Name = "Sveriges konstföreningar: 10%"
					reportLine.TicketName = "Sveriges konstföreningar: 10%"
					price, _ := strconv.ParseFloat(konstPrice, 64)
					amount, _ := strconv.Atoi(konstTicket)
					reportLine.TicketPrice = price
					reportLine.TicketQuantity = amount
					// Export to Json and send to Dynamics 365 for each ticket
					jsData, _ := json.Marshal(reportLine)
					service.PostToD365("new_cashreports", string(jsData))

				}
				if metTicket != "0" {
					reportLine.Name = "Met-rabatt: 10%"
					reportLine.TicketName = "Met-rabatt: 10%"
					price, _ := strconv.ParseFloat(metPrice, 64)
					amount, _ := strconv.Atoi(metTicket)
					reportLine.TicketPrice = price
					reportLine.TicketQuantity = amount
					// Export to Json and send to Dynamics 365 for each ticket
					jsData, _ := json.Marshal(reportLine)
					service.PostToD365("new_cashreports", string(jsData))

				}
				if otherTicket != "0" {
					reportLine.Name = "Annan"
					reportLine.TicketName = "Annan"
					price, _ := strconv.ParseFloat(otherPrice, 64)
					amount, _ := strconv.Atoi(otherTicket)
					reportLine.TicketPrice = price
					reportLine.TicketQuantity = amount
					// Export to Json and send to Dynamics 365 for each ticket
					jsData, _ := json.Marshal(reportLine)
					service.PostToD365("new_cashreports", string(jsData))

				}
				if other2Ticket != "0" {
					reportLine.Name = "Annan 2"
					reportLine.TicketName = "Annan 2"
					price, _ := strconv.ParseFloat(other2Price, 64)
					amount, _ := strconv.Atoi(other2Ticket)
					reportLine.TicketPrice = price
					reportLine.TicketQuantity = amount
					// Export to Json and send to Dynamics 365 for each ticket
					jsData, _ := json.Marshal(reportLine)
					service.PostToD365("new_cashreports", string(jsData))

				}
				// Define the API endpoint
				url := "https://dcf5d3602d82484caa8f70f597e2c3.59.environment.api.powerplatform.com:443/powerautomate/automations/direct/workflows/26b97772aeb345009eecf08a0e8ed059/triggers/manual/paths/invoke?api-version=1&sp=%2Ftriggers%2Fmanual%2Frun&sv=1.0&sig=-Q28UL28_Y4BiYQm4Y9hXuz7jKFmZVDbUkXvgqsqs_Y"

				// Create the payload with the required data
				payload := Payload{
					Booking:  event.ID.String(),
					FormType: strconv.Itoa(form.Type),
				}

				// Marshal the payload into JSON format
				jsonData, err := json.Marshal(payload)
				if err != nil {
					log.Printf("Error marshalling JSON: %v", err)
				}

				// Send the POST request with the JSON data
				resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
				if err != nil {
					log.Printf("Error sending POST request: %v", err)
				}
				defer resp.Body.Close() // Ensure the response body is closed after the function returns

			}
		}

		err = service.Delete(&form)
		if err != nil {
			log.Printf("Error deleting form %s", formID.String())
			log.Printf("Error: %s %s", err, errorMessage)
			c.Status(http.StatusNotFound).SendString(errorMessage)
			return err
		}

		//Return a thank you page
		return c.Render("thankyou", fiber.Map{
			"Title": "Tack",
			"Desc":  "Vi har sparat uppgifterna!",
		})
	}
}
