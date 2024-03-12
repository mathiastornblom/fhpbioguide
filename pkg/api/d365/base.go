// This code defines a package for interacting with Dynamics 365 APIs, including methods for authentication and making GET, POST, and PATCH requests.
package d365

// Import necessary packages
import (
	"encoding/json" // for encoding and decoding JSON data
	"fmt"           // for formatting strings
	"github.com/go-resty/resty/v2" // resty, a simple HTTP and REST client library for Go
)

// Token struct to map the JSON structure of the OAuth token response
type Token struct {
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	ExtExpiresIn int    `json:"ext_expires_in"`
	AccessToken  string `json:"access_token"`
}

// D365 struct holds the configuration and state needed to interact with the Dynamics 365 APIs
type D365 struct {
	Resty        *resty.Client // REST client for making HTTP requests
	URL          string        // Base URL for the Dynamics 365 API
	TenantID     string        // Azure AD Tenant ID
	ClientID     string        // Azure AD Client ID
	ClientSecret string        // Azure AD Client Secret
	AccessToken  string        // Cached OAuth access token
}

// AuthenticateApi performs OAuth authentication to obtain an access token
func (d *D365) AuthenticateApi() {
	// Make an HTTP POST request to the Azure AD token endpoint
	resp, _ := d.Resty.R().
		SetHeader("Content-Type", "application/x-www-form-urlencoded").
		SetFormData(map[string]string{
			"client_id":     d.ClientID,
			"resource":      d.URL,
			"client_secret": d.ClientSecret,
			"grant_type":    "client_credentials"}).
		Post("https://login.microsoftonline.com/" + d.TenantID + "/oauth2/token")

	token := Token{}

	// Decode the JSON response into the Token struct
	json.Unmarshal([]byte(resp.String()), &token)

	// Cache the access token for use in subsequent API requests
	d.AccessToken = token.AccessToken
}

// GetRequest makes an authenticated HTTP GET request to the specified endpoint
func (d *D365) GetRequest(endpoint string) ([]byte, error) {
	// Make the HTTP GET request with the Authorization header set to "Bearer <access_token>"
	resp, err := d.Resty.R().
		SetHeader("Authorization", fmt.Sprintf("Bearer %v", d.AccessToken)).
		Get(d.URL + "/api/data/v9.2/" + endpoint)

	// Print the response for debugging purposes
	fmt.Println(resp.String())

	// Return the response body and any error
	return resp.Body(), err
}

// PostRequest makes an authenticated HTTP POST request to the specified endpoint with the given request body
func (d *D365) PostRequest(endpoint, values string) ([]byte, error) {
	// Make the HTTP POST request with appropriate headers and request body
	resp, err := d.Resty.R().
		SetHeader("Content-Type", "application/json; charset=utf-8").
		SetHeader("Authorization", fmt.Sprintf("Bearer %v", d.AccessToken)).
		SetHeader("Prefer", "return=representation").
		SetBody(values).
		Post(d.URL + "/api/data/v9.2/" + endpoint)

	// Print the response for debugging purposes
	fmt.Println(resp.String())

	// Return the response body and any error
	return resp.Body(), err
}

// PatchRequest makes an authenticated HTTP PATCH request to the specified endpoint with the given request body
func (d *D365) PatchRequest(endpoint, values string) ([]byte, error) {
	// Similar to PostRequest but uses the HTTP PATCH method for partial updates
	resp, err := d.Resty.R().
		SetHeader("Content-Type", "application/json; charset=utf-8").
		SetHeader("Authorization", fmt.Sprintf("Bearer %v", d.AccessToken)).
		SetHeader("Prefer", "return=representation").
		SetBody(values).
		Patch(d.URL + "/api/data/v9.2/" + endpoint)

	// Print the endpoint and response for debugging
	fmt.Println(endpoint)
	fmt.Println(resp.String())

	// Return the response body and any error
	return resp.Body(), err
}
