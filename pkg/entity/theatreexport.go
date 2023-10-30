package entity

import (
	"encoding/xml"
	"time"
)

type TheatreExportList struct {
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
						Text    string `xml:",chardata"`
						Name    string `xml:"name"`
						LogID   string `xml:"log-id"`
						Created string `xml:"created"`
						Server  string `xml:"server"`
						Ip      string `xml:"ip"`
					} `xml:"previous-document"`
				} `xml:"information"`
				Data struct {
					Text     string `xml:",chardata"`
					Type     string `xml:"type,attr"`
					Theatres struct {
						Text    string `xml:",chardata"`
						Theatre []struct {
							Text          string `xml:",chardata"`
							ID            string `xml:"id,attr"`
							TheatreNumber string `xml:"theatre-number"`
							TheatreName   string `xml:"theatre-name"`
							Municipality  string `xml:"municipality"`
							Address       struct {
								Text    string `xml:",chardata"`
								Street0 string `xml:"street0"`
								Street1 string `xml:"street1"`
								Zip     string `xml:"zip"`
								City    string `xml:"city"`
							} `xml:"address"`
							ContactInfo struct {
								Text  string `xml:",chardata"`
								Phone string `xml:"phone"`
								Fax   string `xml:"fax"`
								Email string `xml:"email"`
							} `xml:"contact-info"`
							UpdatedDate string `xml:"updated-date"`
							Salons      struct {
								Text  string `xml:",chardata"`
								Salon []struct {
									Text                  string `xml:",chardata"`
									ID                    string `xml:"id,attr"`
									OwnerNumber           string `xml:"owner-number"`
									SalonName             string `xml:"salon-name"`
									SalonNumber           string `xml:"salon-number"`
									FkbNumber             string `xml:"fkb-number"`
									VatFree               string `xml:"vat-free"`
									Seats                 string `xml:"seats"`
									UpdatedDate           string `xml:"updated-date"`
									SupportedTechnologies struct {
										Text       string `xml:",chardata"`
										Technology []struct {
											Text              string `xml:",chardata"`
											ID                string `xml:"id,attr"`
											Supported         string `xml:"supported,attr"`
											IsPictureFormat   string `xml:"isPictureFormat,attr"`
											IsSound           string `xml:"isSound,attr"`
											IsMovieFormat     string `xml:"isMovieFormat,attr"`
											IsMovieResolution string `xml:"isMovieResolution,attr"`
											IsMovieFPS        string `xml:"isMovieFPS,attr"`
										} `xml:"technology"`
									} `xml:"supported-technologies"`
								} `xml:"salon"`
							} `xml:"salons"`
						} `xml:"theatre"`
					} `xml:"theatres"`
				} `xml:"data"`
				Debug struct {
					Text    string   `xml:",chardata"`
					Message []string `xml:"message"`
				} `xml:"debug"`
			} `xml:"document"`
		} `xml:"ExportResponse"`
	} `xml:"Body"`
}

type LokalDynamics struct {
	LoakalID               string  `json:"new_lokalid,omitempty"`
	InternalID             int     `json:"-"`
	OrgID                  int     `json:"-"`
	Name                   string  `json:"new_name"`
	Adress1                string  `json:"new_deliveryaddress"`
	Adress2                string  `json:"new_deliveryaddress2"`
	ZipCode                string  `json:"new_postnummer"`
	City                   string  `json:"new_ort"`
	VisitAdress            string  `json:"new_gatuadress"`
	VisitCity              string  `json:"new_city"`
	Country                int     `json:"new_country"`
	Email                  string  `json:"new_mail"`
	Phone                  string  `json:"new_phone"`
	Fax                    string  `json:"new_fax"`
	Facebook               string  `json:"new_facebook"`
	Twitter                string  `json:"new_twitter"`
	Instagram              string  `json:"new_instagram"`
	TheatreNum             string  `json:"new_theatrenumber"`
	OwnerNum               string  `json:"new_salonowner"`
	SalonNum               string  `json:"new_salon"`
	FkbNum                 string  `json:"new_fkbid"`
	Lati                   float64 `json:"new_latitude"`
	Long                   float64 `json:"new_longitude"`
	NumberOfSeats          string  `json:"new_numberofseats"`
	CanShow2D              bool    `json:"new_show2d"`
	CanShow3D              bool    `json:"new_show3d"`
	CanShowAtmos           bool    `json:"new_showatmos"`
	CanShowImax            bool    `json:"new_showimax"`
	CanShow35mm            bool    `json:"new_show35mm"`
	CanShow70mm            bool    `json:"new_show70mm"`
	CanShow51Sound         bool    `json:"new_show5_1sound"`
	CanShow71Sound         bool    `json:"new_show7_1sound"`
	CanShowHFR             bool    `json:"new_showhfr"`
	CanShow4DX             bool    `json:"new_show4dx"`
	CanShowLiveBioLive     bool    `json:"new_showlivebiolive"`
	CanShowLiveBioRecorded bool    `json:"new_showlivebiorecorded"`
	CanShowScen            bool    `json:"new_showscen"`
	CanShowUtstallnng      bool    `json:"new_showexhibition"`
	CanShowCourseRoom      bool    `json:"new_showcourseroom"`
	Language               string  `json:"new_language"`
	ShortDesc              string  `json:"new_shortdescription"`
	LongDesc               string  `json:"new_longdescription"`
	LinkToAds              string  `json:"new_linktoads"`
	Customer               string  `json:"new_Konto@odata.bind"`
	BusinessUnit           string  `json:"owningbusinessunit@odata.bind,omitempty"`
	AccountData            struct {
		Name      string `json:"name,omitempty"`
		Accountid string `json:"accountid,omitempty"`
	} `json:"new_Konto,omitempty"`
}

type Lokals struct {
	Items []*LokalDynamics `json:"value"`
}

type Account struct {
	DID                   string    `json:"accountid,omitempty"`
	ID                    int       `json:"new_internalid"`
	Name                  string    `json:"name"`
	OrgName               string    `json:"new_organisationname"`
	OrgNumber             string    `json:"new_organisationsnummer"`
	CustomerNumber        string    `json:"accountnumber"`
	Address               string    `json:"address2_line1"`
	Address2              string    `json:"address2_line2"`
	ZipCode               string    `json:"address2_postalcode"`
	City                  string    `json:"address2_city"`
	Country               int       `json:"new_country"`
	VisitAddress          string    `json:"address1_line1"`
	VisitCity             string    `json:"address1_city"`
	VisitZipCode          string    `json:"address1_postalcode"`
	Email                 string    `json:"emailaddress1"`
	Phone                 string    `json:"telephone1"`
	Phone2                string    `json:"telephone2"`
	Fax                   string    `json:"fax"`
	Website               string    `json:"websiteurl"`
	Associationsform      int       `json:"new_associationsform"`
	Facebook              string    `json:"new_facebook"`
	Twitter               string    `json:"primarytwitterid"`
	Notes                 string    `json:"new_notes"`
	Notes2                string    `json:"new_notes2"`
	Instagram             string    `json:"new_instagram"`
	FakturaAdress1        string    `json:"address1_line2"`
	FakturaAdress2        string    `json:"address1_line3"`
	FakturaZip            string    `json:"new_invoicezipcode"`
	FakturaCity           string    `json:"new_invoicecity"`
	FakturaExtra          string    `json:"new_invoiceextra"`
	BankgiroNummer        string    `json:"new_bankgironumber"`
	PostgiroNummer        string    `json:"new_postgironumber"`
	FakturaEmail          string    `json:"emailaddress2"`
	Accessibility         string    `json:"new_accessibilitypage"`
	TicketPage            string    `json:"new_ticketpurchasepage"`
	Kommunkod             string    `json:"new_municipalitycode"`
	KommunNamn            string    `json:"new_municipality"`
	County                string    `json:"new_county"`
	Region                string    `json:"new_region"`
	Medlem                int       `json:"new_member"`
	MedlemsUttradeType    int       `json:"new_withdrawalreason"`
	StartDate             time.Time `json:"new_memberdate"`
	HasLibrary            bool      `json:"new_library"`
	HasTheater            bool      `json:"new_movietheater"`
	HasHotel              bool      `json:"new_hotel"`
	HasKafe               bool      `json:"new_cafe"`
	HasKonsthall          bool      `json:"new_artgallery"`
	HasLiveBio            bool      `json:"new_livecinema"`
	HasVenue              bool      `json:"new_venue"`
	HasResturant          bool      `json:"new_restaurant"`
	HasScene              bool      `json:"new_scene"`
	FilmAvtal             bool      `json:"new_filmagreement"`
	VatFree               bool      `json:"new_momsbefriad"`
	BusinessUnit          string    `json:"owningbusinessunit@odata.bind,omitempty"`
	TransactionCurrencyId string    `json:"_transactioncurrencyid_value"`
}
