package bioguide

import (
	"html"
	"log"

	"github.com/go-resty/resty/v2"
)

type BioGuiden struct {
	Resty    *resty.Client
	URL      string
	Username string
	Password string
	Logger   *log.Logger
}

func (d *BioGuiden) SOAPRequest(endpoint, data string) ([]byte, error) {
	values := `<?xml version="1.0" encoding="utf-8"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xmlns:xsd="http://www.w3.org/2001/XMLSchema">
    <soap:Body>
        <Export xmlns="` + d.URL + `/">
            <username>` + d.Username + `</username>
            <password>` + d.Password + `</password>
            <xmlDocument>` + html.EscapeString(data) + `</xmlDocument>
        </Export>
    </soap:Body>
</soap:Envelope>`
	resp, err := d.Resty.R().
		SetHeader("Content-Type", "text/xml; charset=iso8859-1").
		SetHeader("SOAPAction", d.URL+"/Export").
		SetBody(values).
		Post(d.URL + "/" + endpoint)

	d.Logger.Println(resp.String())

	return resp.Body(), err
}
