// Package bioguide provides a SOAP client for the BioGuiden API.
package bioguide

import (
	"fmt"
	"html"
	"log/slog"
	"regexp"
	"strings"

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

// APIError is returned by SOAPRequest when BioGuiden answers with an HTTP
// error, a SOAP fault or an ErrorResult document (error codes 1000-1004).
type APIError struct {
	Endpoint string
	Status   int
	Code     string // BioGuiden error-nbr, empty for plain HTTP/SOAP failures
	Message  string
}

func (e *APIError) Error() string {
	if e.Code != "" {
		return fmt.Sprintf("bioguiden %s: error %s: %s", e.Endpoint, e.Code, e.Message)
	}
	return fmt.Sprintf("bioguiden %s: http %d: %s", e.Endpoint, e.Status, e.Message)
}

var (
	errNbrRe   = regexp.MustCompile(`(?s)(?:<|&lt;)error-nbr(?:>|&gt;)\s*(\d+)\s*(?:<|&lt;)/error-nbr`)
	errMsgRe   = regexp.MustCompile(`(?s)(?:<|&lt;)error-message(?:>|&gt;)(.*?)(?:<|&lt;)/error-message`)
	faultRe    = regexp.MustCompile(`(?s)<faultstring[^>]*>(.*?)</faultstring>`)
	maxBodyLog = 300
)

// SOAPRequest sends an XML document to the given BioGuiden endpoint via SOAP.
// The response body is logged at DEBUG level when Logger is set and verbose=true.
// Username, password and document are XML-escaped. An *APIError is returned
// for non-2xx responses and for ErrorResult/SOAP fault bodies, so callers never
// mistake a failed call (for example wrong login) for an empty result.
func (d *BioGuiden) SOAPRequest(endpoint, data string) ([]byte, error) {
	values := `<?xml version="1.0" encoding="utf-8"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xmlns:xsd="http://www.w3.org/2001/XMLSchema">
    <soap:Body>
        <Export xmlns="` + d.URL + `/">
            <username>` + html.EscapeString(d.Username) + `</username>
            <password>` + html.EscapeString(d.Password) + `</password>
            <xmlDocument>` + html.EscapeString(data) + `</xmlDocument>
        </Export>
    </soap:Body>
</soap:Envelope>`

	resp, err := d.Resty.R().
		SetHeader("Content-Type", "text/xml; charset=iso8859-1").
		SetHeader("SOAPAction", d.URL+"/Export").
		SetBody(values).
		Post(d.URL + "/" + endpoint)
	if err != nil {
		return nil, err
	}

	body := resp.Body()
	if d.Logger != nil {
		d.Logger.Debug("BioGuiden SOAP response", "endpoint", endpoint, "status", resp.StatusCode(), "bytes", len(body))
	}

	if apiErr := parseAPIError(endpoint, resp.StatusCode(), body); apiErr != nil {
		return body, apiErr
	}
	return body, nil
}

func parseAPIError(endpoint string, status int, body []byte) *APIError {
	s := string(body)
	if m := errNbrRe.FindStringSubmatch(s); m != nil {
		msg := ""
		if mm := errMsgRe.FindStringSubmatch(s); mm != nil {
			msg = strings.TrimSpace(html.UnescapeString(mm[1]))
		}
		return &APIError{Endpoint: endpoint, Status: status, Code: m[1], Message: msg}
	}
	if status < 200 || status > 299 {
		msg := ""
		if m := faultRe.FindStringSubmatch(s); m != nil {
			msg = strings.TrimSpace(html.UnescapeString(m[1]))
		}
		if msg == "" {
			msg = strings.TrimSpace(s)
			if len(msg) > maxBodyLog {
				msg = msg[:maxBodyLog] + "..."
			}
		}
		return &APIError{Endpoint: endpoint, Status: status, Message: msg}
	}
	return nil
}
