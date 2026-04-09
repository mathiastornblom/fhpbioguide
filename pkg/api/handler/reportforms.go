package handler

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
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

// receiptLine holds one ticket category row for the thank-you receipt.
type receiptLine struct {
	Category string
	Quantity int
	Price    float64
	HasPrice bool
}

// receiptEvent holds the submitted values for one event on the thank-you receipt.
type receiptEvent struct {
	Name  string
	Date  string
	Lines []receiptLine
	Url   string // presale only
}

func MakeReportForms(app *fiber.App, service reportform.UseCase, log *slog.Logger) {
	app.Get("/form/:ID", getForm(service))
	app.Post("/form-post/:ID", postFormResult(service, log))
	app.Get("/api/status", getStatus(log))
	app.Post("/api/genform/presale/:ID", createPresaleForm(service, log))
	app.Post("/api/genform/sold/:ID", createSoldForm(service, log))
	app.Post("/api/regenform/sold/:ID", recreateSoldForm(service, log))
	app.Post("/api/orderstatus", proxyRequest(log))
	app.Post("/api/sync/trigger", triggerSync(log))
}

func getStatus(log *slog.Logger) fiber.Handler {
	return func(c *fiber.Ctx) error {
		log.Debug("status check")
		return c.SendStatus(http.StatusOK)
	}
}

// proxyRequest returns a Fiber handler function for proxying HTTP requests.
func proxyRequest(log *slog.Logger) fiber.Handler {
	l := log.With("component", "ReportForms")
	return func(c *fiber.Ctx) error {
		l.Debug("proxying order status to Movie Transit")

		authHeader := c.Get("Authorization")
		expectedAuth := viper.GetString("proxy.Bearer")
		if authHeader != expectedAuth {
			l.Warn("unauthorized proxy request", "ip", c.IP())
			return c.SendStatus(http.StatusUnauthorized)
		}

		url := viper.GetString("proxy.URL")
		reqBody := bytes.NewBuffer(c.BodyRaw())

		resp, err := http.Post(url, "application/json", reqBody)
		if err != nil {
			l.Error("failed to proxy request to Movie Transit", "err", err)
			return c.Status(http.StatusBadGateway).SendString("Failed to proxy request")
		}
		defer func() {
			if resp != nil {
				resp.Body.Close()
			}
		}()

		if resp.StatusCode != http.StatusOK {
			l.Warn("proxied service responded with non-OK status", "status", resp.StatusCode)
			return c.Status(resp.StatusCode).SendString("Proxied service responded with error")
		}

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

func createPresaleForm(service reportform.UseCase, log *slog.Logger) fiber.Handler {
	l := log.With("component", "ReportForms")
	return func(c *fiber.Ctx) error {
		if c.Get("Authorization") != viper.GetString("report.Bearer") {
			l.Warn("unauthorized createPresaleForm request", "ip", c.IP())
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
		l.Debug("fetching presale bookings", "customer_id", customerID.String())
		data, _ := service.GetFromD365("new_bokningarkunds?$filter=_new_customer_value%20eq%20" + customerID.String() + "%20and%20new_state%20eq%20" + "100000000" + "%20and%20new_presales%20eq%20" + "true" + "&$expand=new_customer_account($select=name),new_product($select=name)")
		/* data, _ := service.GetFromD365("new_bokningarkunds?$filter=_new_customer_value%20eq%20" + customerID.String() + "%20and%20new_state%20eq%20" + "100000000" + "%20and%20new_presales%20eq%20" + "true" + "%20and%20n(_new_kontakt_value%20eq%20" + contactID.String() + "%20or%20_new_forkopskontakt_value%20eq%20" + contactID.String() + ")&$expand=new_customer_account($select=name),new_product($select=name)")*/
		err = json.Unmarshal(data, &bookings)
		if err != nil {
			l.Error("failed to unmarshal presale bookings", "customer_id", customerID.String(), "err", err)
		}
		form := &entity.Form{
			ID:   formID,
			Name: bookings.Value[0].Customer.Name,
			Text: "Rapportera totala förköp för nedanstående evenemang",
			Date: time.Now(),
			Type: 0,
		}

		// Append bookings to form Events array
		l.Debug("appending presale bookings to form events", "count", len(bookings.Value))
		for _, item := range bookings.Value {
			// Look up the most recent presale report for this booking so we can
			// show it as "last reported" text and prefill the quantity field.
			var prevAmount int
			forkopsData, forkopsErr := service.GetFromD365(
				"new_forkops?$filter=_new_boking_value%20eq%20" + item.ID +
					"&$orderby=createdon%20desc&$top=1&$select=new_unit")
			if forkopsErr == nil {
				forkops := entity.DynamicsForkops{}
				if json.Unmarshal(forkopsData, &forkops) == nil && len(forkops.Value) > 0 {
					prevAmount = forkops.Value[0].Unit
				}
			}

			form.Events = append(form.Events, entity.Event{
				ID:             entity.UnsafeStringToID(item.ID),
				Form_type:      0,
				Name:           item.Product.Name,
				Text:           item.Name,
				Date:           item.ShowDate,
				Amount:         prevAmount,
				Url:            item.Url,
				ExpirationTime: time.Now().Add(24 * time.Hour),
				FormID:         formID,
				Discounts:      item.Discounts,
			})
		}

		service.Create(form)
		formURL := "https://" + viper.GetString("report.url") + "/form/" + formID.String()

		for _, item := range bookings.Value {
			if _, err := service.PostToD365("new_bokningarkunds("+item.ID+")", `{"new_forkopsurl":"`+formURL+`"}`); err != nil {
				l.Error("failed to update booking URL in D365", "booking_id", item.ID, "err", err)
			} else {
				l.Debug("updated booking URL in D365", "booking_id", item.ID)
			}
		}

		l.Info("presale form created", "form_id", formID.String(), "customer", form.Name, "events", len(form.Events))
		return c.SendString(formURL)
	}
}

func createSoldForm(service reportform.UseCase, log *slog.Logger) fiber.Handler {
	l := log.With("component", "ReportForms")
	return func(c *fiber.Ctx) error {
		if c.Get("Authorization") != viper.GetString("report.Bearer") {
			l.Warn("unauthorized createSoldForm request", "ip", c.IP())
			return c.SendStatus(http.StatusForbidden)
		}
		customerID, err := entity.StringToID(c.Params("ID"))
		if err != nil {
			l.Warn("invalid customer ID in createSoldForm", "param", c.Params("ID"))
			c.Status(http.StatusNotFound)
			return err
		}

		formID := entity.NewID()
		bookings := entity.Bookings{}
		data, _ := service.GetFromD365("new_bokningarkunds?$filter=_new_customer_value%20eq%20" + customerID.String() + "%20and%20new_state%20eq%20" + "100000001" + "%20and%20new_slutredovisning%20eq%20true%20and%20new_showdate%20eq%20" + time.Now().Format("2006-01-02") + "&$expand=new_customer_account($select=name),new_product($select=name)")
		err = json.Unmarshal(data, &bookings)
		if err != nil {
			l.Error("failed to unmarshal sold bookings", "customer_id", customerID.String(), "err", err)
			return err
		}
		form := &entity.Form{
			ID:   formID,
			Name: bookings.Value[0].Customer.Name,
			Text: "Rapportera biljetter sålda för nedanstående evenemang",
			Date: time.Now(),
			Type: 1,
		}

		l.Debug("appending sold bookings to form events", "count", len(bookings.Value))
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
				Discounts:      item.Rabatter,
				MinPrice:       item.MinBiljettpris,
			})
		}

		service.Create(form)
		formURL := "https://" + viper.GetString("report.url") + "/form/" + formID.String()
		l.Info("sold form created", "form_id", formID.String(), "customer", form.Name, "events", len(form.Events))
		return c.SendString(formURL)
	}
}

func recreateSoldForm(service reportform.UseCase, log *slog.Logger) fiber.Handler {
	l := log.With("component", "ReportForms")
	return func(c *fiber.Ctx) error {
		if c.Get("Authorization") != viper.GetString("report.Bearer") {
			l.Warn("unauthorized recreateSoldForm request", "ip", c.IP())
			return c.SendStatus(http.StatusForbidden)
		}
		bID, err := entity.StringToID(c.Params("ID"))
		if err != nil {
			l.Warn("invalid booking ID in recreateSoldForm", "param", c.Params("ID"))
			c.Status(http.StatusNotFound)
			return err
		}

		b, err := service.GetEvent(bID)
		// Check if booking is valid
		if err != nil {
			return err
		}
		if b.ID != uuid.Nil {
			// Refresh MinPrice and Discounts from D365 in case they were added after form creation.
			refreshData, refreshErr := service.GetFromD365("new_bokningarkunds(" + bID.String() + ")?$select=new_bokningarkundid,new_rabatter,new_minimipris")
			if refreshErr == nil {
				refreshed := entity.Booking{}
				if json.Unmarshal(refreshData, &refreshed) == nil && refreshed.ID != "" {
					b.Discounts = refreshed.Rabatter
					b.MinPrice = refreshed.MinBiljettpris
					if updateErr := service.UpdateEvent(&b); updateErr != nil {
						l.Error("failed to refresh event data", "booking_id", bID.String(), "err", updateErr)
					} else {
						l.Debug("refreshed event MinPrice and Discounts", "booking_id", bID.String(), "minprice", refreshed.MinBiljettpris, "rabatter", refreshed.Rabatter)
					}
				} else {
					l.Warn("D365 returned unexpected data during event refresh", "booking_id", bID.String())
				}
			}
			return c.SendString("https://" + viper.GetString("report.url") + "/form/" + b.FormID.String())
		}

		formID := entity.NewID()
		booking := entity.Booking{}
		l.Debug("fetching booking from D365", "booking_id", bID.String())
		data, _ := service.GetFromD365("new_bokningarkunds(" + bID.String() + ")?$expand=new_customer_account($select=name),new_product($select=name)")
		err = json.Unmarshal(data, &booking)
		if err != nil {
			l.Error("failed to unmarshal booking", "booking_id", bID.String(), "err", err)
			return err
		}

		form := &entity.Form{
			ID:   formID,
			Name: booking.Customer.Name,
			Text: "Rapportera biljetter sålda för nedanstående evenemang",
			Date: time.Now(),
			Type: 1,
		}

		form.Events = append(form.Events, entity.Event{
			ID:             entity.UnsafeStringToID(booking.ID),
			Form_type:      1,
			Name:           booking.Product.Name,
			Text:           booking.Name,
			Date:           booking.ShowDate,
			Amount:         0,
			ExpirationTime: time.Now().Add(24 * time.Hour),
			FormID:         formID,
			Discounts:      booking.Rabatter,
			MinPrice:       booking.MinBiljettpris,
		})

		service.Create(form)
		formURL := "https://" + viper.GetString("report.url") + "/form/" + formID.String()
		l.Info("sold form recreated", "form_id", formID.String(), "booking_id", bID.String(), "customer", form.Name)
		return c.SendString(formURL)
	}
}

func postFormResult(service reportform.UseCase, log *slog.Logger) fiber.Handler {
	l := log.With("component", "ReportForms")
	return func(c *fiber.Ctx) error {
		errorMessage := "Error reading form"
		formID, err := entity.StringToID(c.Params("ID"))
		// check if ID is defined
		if err != nil {
			l.Warn("invalid form ID in URL", "form_id", formID.String())
			c.Status(http.StatusNotFound).SendString(errorMessage)
			return err
		}
		form, err := service.GetForm(formID)
		// check if form was found or not
		if err != nil || form.ID == uuid.Nil {
			l.Warn("form not found", "form_id", formID.String())
			c.Status(http.StatusNotFound).SendString(errorMessage)
			return err
		}

		httpClient := &http.Client{Timeout: 10 * time.Second}
		var receipt []receiptEvent

		if form.Type == 0 {
			for id, event := range form.Events {
				qty, _ := strconv.Atoi(c.FormValue("00_"+strconv.Itoa(id), "0"))
				ticketURL := c.FormValue("10_" + strconv.Itoa(id))

				service.PostToD365("new_forkops", `{"new_boking@odata.bind":"new_bokningarkunds(`+event.ID.String()+`)","new_forkopsurl":"`+ticketURL+`","new_unit":`+strconv.Itoa(qty)+`}`)
				if _, err := service.PostToD365("new_bokningarkunds("+event.ID.String()+")", `{"new_forkopsurl":"`+ticketURL+`"}`); err != nil {
					l.Error("failed to update booking URL in D365", "booking_id", event.ID.String(), "err", err)
				}

				receipt = append(receipt, receiptEvent{
					Name: event.Name,
					Date: event.Date,
					Url:  ticketURL,
					Lines: []receiptLine{
						{Category: "Totalt antal förköp", Quantity: qty},
					},
				})

				// Define the API endpoint
				url := "https://dcf5d3602d82484caa8f70f597e2c3.59.environment.api.powerplatform.com:443/powerautomate/automations/direct/workflows/26b97772aeb345009eecf08a0e8ed059/triggers/manual/paths/invoke?api-version=1&sp=%2Ftriggers%2Fmanual%2Frun&sv=1.0&sig=-Q28UL28_Y4BiYQm4Y9hXuz7jKFmZVDbUkXvgqsqs_Y"

				payload := Payload{
					Booking:  event.ID.String(),
					FormType: strconv.Itoa(form.Type),
				}
				jsonData, err := json.Marshal(payload)
				if err != nil {
					l.Error("failed to marshal webhook payload", "err", err)
				}
				resp, err := httpClient.Post(url, "application/json", bytes.NewBuffer(jsonData))
				if err != nil {
					l.Error("failed to send Power Automate webhook", "err", err)
					continue
				}
				resp.Body.Close()
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

				// Build receipt for this event.
				ev := receiptEvent{Name: event.Name, Date: event.Date}
				addLine := func(cat, qtyStr, priceStr string, hasPrice bool) {
					qty, _ := strconv.Atoi(qtyStr)
					if qty == 0 {
						return
					}
					price, _ := strconv.ParseFloat(priceStr, 64)
					ev.Lines = append(ev.Lines, receiptLine{
						Category: cat, Quantity: qty, Price: price, HasPrice: hasPrice,
					})
				}
				addLine("Ordinarie", standardTicket, standardPrice, true)
				addLine("Fribiljett", freeTicket, "", false)
				addLine("Barn/Ungdom under 26 år: 25%", kidsTicket, kidsPrice, true)
				addLine("Abonnemang på minst 5 föreställningar: 25%", playsTicket, playsPrice, true)
				addLine("Scenpass Sverige: 10%", scenTicket, scenPrice, true)
				addLine("Sveriges konstföreningar: 10%", konstTicket, konstPrice, true)
				addLine("Met-rabatt: 10%", metTicket, metPrice, true)
				addLine("Annan", otherTicket, otherPrice, true)
				addLine("Annan 2", other2Ticket, other2Price, true)
				if len(ev.Lines) == 0 {
					ev.Lines = append(ev.Lines, receiptLine{Category: "Ordinarie", Quantity: 0})
				}
				receipt = append(receipt, ev)

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
					l.Error("failed to marshal webhook payload", "err", err)
				}

				// Send the POST request with the JSON data
				resp, err := httpClient.Post(url, "application/json", bytes.NewBuffer(jsonData))
				if err != nil {
					l.Error("failed to send Power Automate webhook", "err", err)
					continue
				}
				resp.Body.Close()
			}
		}

		err = service.Delete(&form)
		if err != nil {
			l.Error("failed to delete form after submission", "form_id", formID.String(), "err", err)
			c.Status(http.StatusNotFound).SendString(errorMessage)
			return err
		}

		l.Info("form submitted successfully", "form_id", formID.String(), "type", form.Type)
		//Return a thank you page
		return c.Render("thankyou", fiber.Map{
			"Title":    "Tack",
			"Desc":     "Vi har sparat uppgifterna!",
			"FormType": form.Type,
			"Receipt":  receipt,
		})
	}
}

// triggerSync writes a trigger file that fhpbioguide's poller picks up within 60s.
// Both processes must be configured with the same sync.triggerFile and sync.lockFile paths.
//
// Responses:
//
//	202 Accepted          – trigger file created, sync will start within 60s
//	202 Accepted          – trigger file already pending (previous trigger not yet consumed)
//	409 Conflict          – sync is currently running (lock file held)
//	401 Unauthorized      – missing or wrong Authorization header
//	500 Internal Error    – could not write trigger file
func triggerSync(log *slog.Logger) fiber.Handler {
	l := log.With("component", "ReportForms")
	return func(c *fiber.Ctx) error {
		if c.Get("Authorization") != viper.GetString("report.Bearer") {
			l.Warn("unauthorized sync trigger", "ip", c.IP())
			return c.SendStatus(http.StatusUnauthorized)
		}

		lockFile := viper.GetString("sync.lockFile")
		triggerFile := viper.GetString("sync.triggerFile")

		// If lock file is present and not stale, a sync is actively running.
		if info, err := os.Stat(lockFile); err == nil {
			if time.Since(info.ModTime()) <= 4*time.Hour {
				l.Warn("sync trigger rejected — sync already running", "lock_age", time.Since(info.ModTime()).Round(time.Second))
				return c.Status(http.StatusConflict).SendString("sync already in progress")
			}
		}

		// Try to create the trigger file exclusively.
		f, err := os.OpenFile(triggerFile, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
		if err != nil {
			if os.IsExist(err) {
				l.Info("sync trigger already pending")
				return c.Status(http.StatusAccepted).SendString("sync trigger already pending, will start within 60s")
			}
			l.Error("failed to write sync trigger file", "err", err)
			return c.Status(http.StatusInternalServerError).SendString("failed to create trigger")
		}
		f.Close()

		l.Info("sync trigger created", "path", triggerFile)
		return c.Status(http.StatusAccepted).SendString("sync triggered, will start within 60s")
	}
}
