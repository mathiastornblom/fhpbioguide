package entity

import (
	"encoding/xml"
	"time"
)

// CashReportsDistributoristExport represents the XML structure of a cash report export.
type CashReportsDistributoristExport struct {
    // XMLName defines the root element name and the associated namespaces and attributes.
	XMLName xml.Name `xml:"Envelope"`
    // The remaining fields map the hierarchical structure of the XML document, capturing
    // information about the report, such as details on the distributor, the movies shown,
    // the salons/theatres where they were shown, and detailed ticketing information for each show.
    // Nested structs represent the depth of the XML structure, allowing for accurate parsing and data extraction.
	Text    string   `xml:",chardata"`
	Soap    string   `xml:"soap,attr"`
	Xsi     string   `xml:"xsi,attr"`
	Xsd     string   `xml:"xsd,attr"`
	Body    struct {
		Text           string `xml:",chardata"`
		ExportResponse struct {
			Text     string `xml:",chardata"`
			Xmlns    string `xml:"xmlns,attr"`
			Document struct {
				Text        string `xml:",chardata"`
				Information struct {
					Text             string `xml:",chardata"`
					Name             string `xml:"name"`
					Description      string `xml:"description"`
					LogID            string `xml:"log-id"`
					Created          string `xml:"created"`
					Server           string `xml:"server"`
					Ip               string `xml:"ip"`
					PreviousDocument struct {
						Text        string `xml:",chardata"`
						Name        string `xml:"name"`
						Description string `xml:"description"`
						LogID       string `xml:"log-id"`
						Created     string `xml:"created"`
						Server      string `xml:"server"`
						Ip          string `xml:"ip"`
					} `xml:"previous-document"`
				} `xml:"information"`
				Data struct {
					Text        string `xml:",chardata"`
					Type        string `xml:"type,attr"`
					Cashreports struct {
						Text       string `xml:",chardata"`
						Cashreport []struct {
							Text             string `xml:",chardata"`
							ID               string `xml:"id,attr"`
							CashreportNumber string `xml:"cashreport-number"`
							Salon            struct {
								Text          string `xml:",chardata"`
								ID            string `xml:"id,attr"`
								TheatreName   string `xml:"theatre-name"`
								TheatreNumber string `xml:"theatre-number"`
								SalonName     string `xml:"salon-name"`
								SalonNumber   string `xml:"salon-number"`
								City          string `xml:"city"`
								OwnerNumber   string `xml:"owner-number"`
								FkbNumber     string `xml:"fkb-number"`
								GreenSalon    string `xml:"green-salon"`
								VatFree       string `xml:"vat-free"`
							} `xml:"salon"`
							Playweek struct {
								Text      string `xml:",chardata"`
								ID        string `xml:"id,attr"`
								StartDate string `xml:"start-date"`
								EndDate   string `xml:"end-date"`
							} `xml:"playweek"`
							Movie struct {
								Text            string `xml:",chardata"`
								ID              string `xml:"id,attr"`
								FullMovieNumber string `xml:"full-movie-number"`
								Title           string `xml:"title"`
								OriginalTitle   string `xml:"original-title"`
								PictureFormat   string `xml:"picture-format"`
								MovieFormat     string `xml:"movie-format"`
								Resolution      string `xml:"resolution"`
								Fps             string `xml:"fps"`
								Sound           string `xml:"sound"`
							} `xml:"movie"`
							Distributor struct {
								Text   string `xml:",chardata"`
								ID     string `xml:"id,attr"`
								Name   string `xml:"name"`
								Number string `xml:"number"`
							} `xml:"distributor"`
							GreenByDecision string `xml:"green-by-decision"`
							Parallel        string `xml:"parallel"`
							BookingNbr      string `xml:"booking-nbr"`
							Shows           struct {
								Text string `xml:",chardata"`
								Show []struct {
									Text          string `xml:",chardata"`
									ID            string `xml:"id,attr"`
									StartDateTime string `xml:"start-date-time"`
									TicketDetails struct {
										Text   string `xml:",chardata"`
										Detail []struct {
											Text     string `xml:",chardata"`
											ID       string `xml:"id,attr"`
											TypeID   string `xml:"type-id"`
											Category string `xml:"category"`
											Quantity string `xml:"quantity"`
											Price    string `xml:"price"`
											Total    string `xml:"total"`
										} `xml:"detail"`
									} `xml:"ticket-details"`
									TotalTickets           string `xml:"total-tickets"`
									TotalDistributorAmount string `xml:"total-distributor-amount"`
									TotalCashAmount        string `xml:"total-cash-amount"`
								} `xml:"show"`
							} `xml:"shows"`
							Approved struct {
								Text            string `xml:",chardata"`
								CinemaDate      string `xml:"cinema-date"`
								CinemaName      string `xml:"cinema-name"`
								DistributorDate string `xml:"distributor-date"`
								DistributorName string `xml:"distributor-name"`
							} `xml:"approved"`
							Messages struct {
								Text        string `xml:",chardata"`
								CinemaOwner string `xml:"cinema-owner"`
								Distributor string `xml:"distributor"`
								Sfi         string `xml:"sfi"`
							} `xml:"messages"`
							Debit struct {
								Text                     string `xml:",chardata"`
								DebitPercentage1         string `xml:"debit-percentage1"`
								DebitMoney1              string `xml:"debit-money1"`
								DebitPercentage2         string `xml:"debit-percentage2"`
								DebitMoney2              string `xml:"debit-money2"`
								DebitPercentage3         string `xml:"debit-percentage3"`
								DebitMoney3              string `xml:"debit-money3"`
								DebitGuaranteedMovieRent string `xml:"debit-guaranteed-movie-rent"`
								DebitOther               string `xml:"debit-other"`
							} `xml:"debit"`
							TotalShows struct {
								Text    string `xml:",chardata"`
								Day     string `xml:"day"`
								Evening string `xml:"evening"`
								Night   string `xml:"night"`
								Total   string `xml:"total"`
							} `xml:"total-shows"`
							TotalTickets struct {
								Text    string `xml:",chardata"`
								Day     string `xml:"day"`
								Evening string `xml:"evening"`
								Night   string `xml:"night"`
								Total   string `xml:"total"`
							} `xml:"total-tickets"`
							TotalDistributorVat struct {
								Text    string `xml:",chardata"`
								Day     string `xml:"day"`
								Evening string `xml:"evening"`
								Night   string `xml:"night"`
								Total   string `xml:"total"`
							} `xml:"total-distributor-vat"`
							TotalDistributorSfifee struct {
								Text    string `xml:",chardata"`
								Day     string `xml:"day"`
								Evening string `xml:"evening"`
								Night   string `xml:"night"`
								Total   string `xml:"total"`
							} `xml:"total-distributor-sfifee"`
							TotalCashVat struct {
								Text    string `xml:",chardata"`
								Day     string `xml:"day"`
								Evening string `xml:"evening"`
								Night   string `xml:"night"`
								Total   string `xml:"total"`
							} `xml:"total-cash-vat"`
							TotalCashSfifee struct {
								Text    string `xml:",chardata"`
								Day     string `xml:"day"`
								Evening string `xml:"evening"`
								Night   string `xml:"night"`
								Total   string `xml:"total"`
							} `xml:"total-cash-sfifee"`
							TotalDistributorAmountExVat struct {
								Text    string `xml:",chardata"`
								Day     string `xml:"day"`
								Evening string `xml:"evening"`
								Night   string `xml:"night"`
								Total   string `xml:"total"`
							} `xml:"total-distributor-amount-ex-vat"`
							TotalCashAmountExVat struct {
								Text    string `xml:",chardata"`
								Day     string `xml:"day"`
								Evening string `xml:"evening"`
								Night   string `xml:"night"`
								Total   string `xml:"total"`
							} `xml:"total-cash-amount-ex-vat"`
							UpdatedDate string `xml:"updated-date"`
						} `xml:"cashreport"`
					} `xml:"cashreports"`
				} `xml:"data"`
				Debug struct {
					Text    string   `xml:",chardata"`
					Message []string `xml:"message"`
				} `xml:"debug"`
			} `xml:"document"`
		} `xml:"ExportResponse"`
	} `xml:"Body"`
}

// CashReportsDistributoristListExport defines the structure for exporting a list of cash reports.
// It includes fields for parsing the XML structure that summarizes the cash report data,
// such as the number of cash reports per date, salon, or movie, and includes a search interval for the report generation.
type CashReportsDistributoristListExport struct {
	XMLName xml.Name `xml:"Envelope"`
	Text    string   `xml:",chardata"`
	Soap    string   `xml:"soap,attr"`
	Xsi     string   `xml:"xsi,attr"`
	Xsd     string   `xml:"xsd,attr"`
	Body    struct {
		Text           string `xml:",chardata"`
		ExportResponse struct {
			Text     string `xml:",chardata"`
			Xmlns    string `xml:"xmlns,attr"`
			Document struct {
				Text        string `xml:",chardata"`
				Information struct {
					Text             string `xml:",chardata"`
					Name             string `xml:"name"`
					Description      string `xml:"description"`
					LogID            string `xml:"log-id"`
					Created          string `xml:"created"`
					Server           string `xml:"server"`
					Ip               string `xml:"ip"`
					PreviousDocument struct {
						Text        string `xml:",chardata"`
						Name        string `xml:"name"`
						Description string `xml:"description"`
						LogID       string `xml:"log-id"`
						Created     string `xml:"created"`
						Server      string `xml:"server"`
						Ip          string `xml:"ip"`
					} `xml:"previous-document"`
				} `xml:"information"`
				Data struct {
					Text           string `xml:",chardata"`
					Type           string `xml:"type,attr"`
					SearchInterval struct {
						Text      string `xml:",chardata"`
						StartDate string `xml:"start-date"`
						EndDate   string `xml:"end-date"`
					} `xml:"search-interval"`
					Dates struct {
						Text string `xml:",chardata"`
						Date []struct {
							Text                string `xml:",chardata"`
							NumberOfCashreports string `xml:"number-of-cashreports,attr"`
							UpdatedDate         string `xml:"updated-date,attr"`
						} `xml:"date"`
					} `xml:"dates"`
					Salons struct {
						Text  string `xml:",chardata"`
						Salon []struct {
							Text                string `xml:",chardata"`
							NumberOfCashreports string `xml:"number-of-cashreports,attr"`
							FkbNumber           string `xml:"fkb-number,attr"`
							SalonNumber         string `xml:"salon-number,attr"`
							TheatreNumber       string `xml:"theatre-number,attr"`
							OwnerNumber         string `xml:"owner-number,attr"`
						} `xml:"salon"`
					} `xml:"salons"`
					Movies struct {
						Text  string `xml:",chardata"`
						Movie []struct {
							Text                string `xml:",chardata"`
							NumberOfCashreports string `xml:"number-of-cashreports,attr"`
							FullMovieNumber     string `xml:"full-movie-number,attr"`
							ID                  string `xml:"id,attr"`
						} `xml:"movie"`
					} `xml:"movies"`
					ReportGenerationDate string `xml:"report-generation-date"`
				} `xml:"data"`
				Debug struct {
					Text    string   `xml:",chardata"`
					Message []string `xml:"message"`
				} `xml:"debug"`
			} `xml:"document"`
		} `xml:"ExportResponse"`
	} `xml:"Body"`
}

// DynamicsBooking represents the structure of a booking record in Dynamics 365.
// It includes fields for the booking details, such as the show date, customer information, product details, and tickets sold.
type DynamicsBooking struct {
	ID                 string `json:"new_bokningarkundid,omitempty"`
	Name               string `json:"new_name,omitempty"`
	ShowDate           string `json:"new_showdate,omitempty"`
	Owningbusinessunit string `json:"_owningbusinessunit_value,omitempty"`
	CustomerID         string `json:"_new_customer_value,omitempty"`
	Customer           struct {
		Name      string `json:"name,omitempty"`
		Accountid string `json:"accountid,omitempty"`
	} `json:"new_customer_account,omitempty"`
	Product struct {
		Name      string `json:"name,omitempty"`
		Productid string `json:"productid,omitempty"`
	} `json:"new_product,omitempty"`
	Lokal struct {
		Name      string `json:"new_name,omitempty"`
		FkbNumber string `json:"new_fkbid,omitempty"`
		Lokalid   string `json:"new_lokalid,omitempty"`
	} `json:"new_Lokaler,omitempty"`
	TicketsSold int     `json:"new_ticketssoldbioguiden,omitempty"`
	TicketPrice float64 `json:"new_ticketpricebioguiden,omitempty"`
}

// DynamicsBookings is a container for a slice of DynamicsBooking pointers,
// allowing for the collection of multiple booking records.
type DynamicsBookings struct {
	Value []*DynamicsBooking `json:"value"`
}

// DynamicsBookingPost represents a structure used for posting updates to a booking record,
// specifically for updating tickets sold and ticket price information.
type DynamicsBookingPost struct {
	TicketsSold int     `json:"new_ticketssoldbioguiden,omitempty"`
	TicketPrice float64 `json:"new_ticketpricebioguiden,omitempty"`
}

// DynamicsCashReports is a container for a slice of DynamicsCashReport pointers,
// allowing for the collection of multiple cash reports records.
type DynamicsCashReports struct {
	Items []*DynamicsCashReport `json:"value"`
}

// DynamicsCashReport represents a structure used for posting updates to a cach report record,
// specifically for updating tickets sold and ticket price information.
type DynamicsCashReport struct {
	ID                    string    `json:"new_cashreportid,omitempty"`
	Name                  string    `json:"new_name"`
	FKBID                 string    `json:"new_fkbnumber"`
	Lokal                 string    `json:"new_saloon@odata.bind"`
	Account               string    `json:"new_organisation@odata.bind"`
	Event                 string    `json:"new_event_id@odata.bind,omitempty"`
	Booking               string    `json:"new_booking@odata.bind,omitempty"`
	ReportNum             string    `json:"new_cashreportnumber"`
	ShowNum               int       `json:"new_shownum"`
	FullMovieNumber       string    `json:"new_fullmovienumber"`
	TicketName            string    `json:"new_ticketcategory"`
	TicketQuantity        int       `json:"new_ticket_quantity"`
	TicketPrice           float64   `json:"new_ticket_price"`
	Source                int       `json:"new_source"`
	ShowDate              time.Time `json:"new_startdate"`
	Playweek              string    `json:"new_playweek"`
	RecordedAmount        float64   `json:"new_recordedamount"`
	VatFree               bool      `json:"new_vatfree"`
	TransactionCurrencyId string    `json:"transactioncurrencyid@odata.bind,omitempty"`
}
