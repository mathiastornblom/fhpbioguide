package entity

import (
	"encoding/xml"
	"time"
)

type MovieExportList struct {
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
					Text   string `xml:",chardata"`
					Type   string `xml:"type,attr"`
					Movies struct {
						Text  string `xml:",chardata"`
						Movie []struct {
							Text            string `xml:",chardata"`
							MainID          string `xml:"main-id,attr"`
							ID              string `xml:"id,attr"`
							FullMovieNumber string `xml:"full-movie-number"`
							Title           string `xml:"title"`
							OriginalTitle   string `xml:"original-title"`
							ShortTitle      string `xml:"short-title"`
							Description     string `xml:"description"`
							Genres          struct {
								Text  string   `xml:",chardata"`
								Genre []string `xml:"genre"`
							} `xml:"genres"`
							AccessibilityAids struct {
								Text string `xml:",chardata"`
								Aid  []struct {
									Text           string `xml:",chardata"`
									Type           string `xml:"type"`
									Name           string `xml:"name"`
									Classification string `xml:"classification"`
									Fps            string `xml:"fps"`
									Language       string `xml:"language"`
								} `xml:"aid"`
							} `xml:"accessibility-aids"`
							Rating             string `xml:"rating"`
							Runtime            string `xml:"runtime"`
							Directors          string `xml:"directors"`
							Producers          string `xml:"producers"`
							MainActors         string `xml:"main-actors"`
							Language           string `xml:"language"`
							Subtitles          string `xml:"subtitles"`
							PictureFormat      string `xml:"picture-format"`
							MovieFormat        string `xml:"movie-format"`
							Resolution         string `xml:"resolution"`
							Fps                string `xml:"fps"`
							Sound              string `xml:"sound"`
							ProductionCountry  string `xml:"production-country"`
							CurrentlyLocked    string `xml:"currently-locked"`
							SelfBooking        string `xml:"self-booking"`
							PremiereDate       string `xml:"premiere-date"`
							PremiereType       string `xml:"premiere-type"`
							BookingInformation string `xml:"booking-information"`
							Distributor        struct {
								Text   string `xml:",chardata"`
								ID     string `xml:"id,attr"`
								Name   string `xml:"name"`
								Number string `xml:"number"`
							} `xml:"distributor"`
							Links struct {
								Text                   string `xml:",chardata"`
								Imdb                   string `xml:"imdb"`
								Twitter                string `xml:"twitter"`
								Facebook               string `xml:"facebook"`
								FacebookEnglish        string `xml:"facebook-english"`
								Youtube                string `xml:"youtube"`
								OfficialWebsite        string `xml:"official-website"`
								OfficialEnglishWebsite string `xml:"official-english-website"`
							} `xml:"links"`
							Biopasset   string `xml:"biopasset"`
							UpdatedDate string `xml:"updated-date"`
							RevokedDate string `xml:"revoked-date"`
						} `xml:"movie"`
					} `xml:"movies"`
				} `xml:"data"`
				Debug struct {
					Text    string   `xml:",chardata"`
					Message []string `xml:"message"`
				} `xml:"debug"`
			} `xml:"document"`
		} `xml:"ExportResponse"`
	} `xml:"Body"`
}

type Product struct {
	ID               string    `json:"productid,omitempty"`
	Name             string    `json:"name"`
	Status           bool      `json:"new_eventstatus"`
	Description      string    `json:"description"`
	Vendorname       string    `json:"vendorname"`
	Length           int       `json:"new_langdmin"`
	Distributor      int       `json:"new_distributor"`
	Censur           int       `json:"new_censur"`
	Productnumber    string    `json:"productnumber"`
	Producttypecode  int       `json:"producttypecode"`
	Productstructure int       `json:"productstructure"`
	ShowdateStart    time.Time `json:"new_showdatestart"`
	Premier          time.Time `json:"new_premier"`
	Tillleverantor   int       `json:"new_tillleverantor,omitempty"`
	Textning         bool      `json:"new_textning"`
	Moms             int       `json:"new_moms,omitempty"`
	UnitScheduleID   string    `json:"defaultuomscheduleid@odata.bind,omitempty"`
	UnitID           string    `json:"defaultuomid@odata.bind,omitempty"`
}

type Products struct {
	Items []*Product `json:"value"`
}
