// Package bioguide provides a SOAP client for the BioGuiden API.
package bioguide

import (
	"html"
	"log/slog"

	"github.com/go-resty/resty/v2"
)

// BioGuiden is a SOAP client for BioGuiden requests.
type BioGuiden struct {
	Resty    *resty.Client
	URL      string
	Username string
	Password string
	Logger   *slog.Logger // optional; nil disables response body logging
}

// SOAPRequest sends an XML document to the given BioGuiden endpoint via SOAP.
// The response body is logged at DEBUG level when Logger is set and verbose=true.
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

	if d.Logger != nil {
		d.Logger.Debug("BioGuiden SOAP response", "endpoint", endpoint, "body", resp.String())
	}

	return resp.Body(), err
}
