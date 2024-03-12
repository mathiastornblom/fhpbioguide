// Define the package `bioguide`
package bioguide

// Import necessary packages:
// - "html" for escaping HTML entities in strings,
// - "log" for logging messages,
// - "github.com/go-resty/resty/v2" for making HTTP requests using Resty, a simple HTTP and REST client library for Go.
import (
	"html"
	"log"
	"github.com/go-resty/resty/v2"
)

// BioGuiden struct definition: this acts as a configuration and client structure for interacting with a specific API or service.
type BioGuiden struct {
	Resty    *resty.Client // HTTP client from Resty for making requests.
	URL      string        // Base URL of the API or service.
	Username string        // Username for authentication.
	Password string        // Password for authentication.
	Logger   *log.Logger   // Logger instance for logging.
}

// SOAPRequest is a method defined on the BioGuiden struct. It prepares and sends a SOAP request to the specified endpoint with the provided data.
func (d *BioGuiden) SOAPRequest(endpoint, data string) ([]byte, error) {
	// Prepare the SOAP request body as a string with XML formatting. This includes the necessary SOAP envelope, body, and includes the XML data to be sent.
	// `html.EscapeString` is used to escape any HTML characters within the `data` string to prevent XML injection attacks.
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

	// Use the Resty client to send a POST request to the service. The request is configured with necessary headers and the prepared SOAP body.
	resp, err := d.Resty.R().
		SetHeader("Content-Type", "text/xml; charset=iso8859-1"). // Set the content type to text/xml with charset iso8859-1.
		SetHeader("SOAPAction", d.URL+"/Export").                  // Set the SOAPAction header to indicate the action being called.
		SetBody(values).                                           // Include the prepared SOAP request body.
		Post(d.URL + "/" + endpoint)                               // Send the POST request to the full URL constructed from the base URL and the endpoint.

	// Log the response using the provided Logger instance. This helps in debugging and monitoring the responses from the API.
	d.Logger.Println(resp.String())

	// Return the response body as a byte slice and any error that occurred during the request.
	return resp.Body(), err
}