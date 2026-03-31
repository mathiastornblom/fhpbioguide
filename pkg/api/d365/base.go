// This code defines a package for interacting with Dynamics 365 APIs, including methods for authentication and making GET, POST, and PATCH requests.
package d365

// Import necessary packages
import (
	"encoding/json" // for encoding and decoding JSON data
	"fmt"           // for formatting strings
	"time"          // for token expiry tracking

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
	ExpiresAt    time.Time     // When the cached token expires
}

// isTokenValid returns true if we have a non-empty token that has not yet expired.
func (d *D365) isTokenValid() bool {
	return d.AccessToken != "" && time.Now().Before(d.ExpiresAt)
}

// AuthenticateApi performs OAuth authentication to obtain an access token.
// Returns an error if the request fails or no access token is returned.
func (d *D365) AuthenticateApi() error {
	resp, err := d.Resty.R().
		SetHeader("Content-Type", "application/x-www-form-urlencoded").
		SetFormData(map[string]string{
			"client_id":     d.ClientID,
			"resource":      d.URL,
			"client_secret": d.ClientSecret,
			"grant_type":    "client_credentials"}).
		Post("https://login.microsoftonline.com/" + d.TenantID + "/oauth2/token")
	if err != nil {
		return fmt.Errorf("D365 token request failed: %w", err)
	}

	token := Token{}
	if err := json.Unmarshal([]byte(resp.String()), &token); err != nil {
		return fmt.Errorf("D365 token parse failed: %w", err)
	}
	if token.AccessToken == "" {
		return fmt.Errorf("D365 returned empty access token (status %d): %s", resp.StatusCode(), resp.String())
	}

	d.AccessToken = token.AccessToken
	// Use a 60-second safety buffer so we refresh before the token actually expires.
	d.ExpiresAt = time.Now().Add(time.Duration(token.ExpiresIn-60) * time.Second)
	return nil
}

// ensureToken refreshes the token if it is missing or expired.
func (d *D365) ensureToken() error {
	if d.isTokenValid() {
		return nil
	}
	return d.AuthenticateApi()
}

// GetRequest makes an authenticated HTTP GET request to the specified endpoint
func (d *D365) GetRequest(endpoint string) ([]byte, error) {
	if err := d.ensureToken(); err != nil {
		return nil, err
	}
	resp, err := d.Resty.R().
		SetHeader("Authorization", fmt.Sprintf("Bearer %v", d.AccessToken)).
		Get(d.URL + "/api/data/v9.2/" + endpoint)

	fmt.Println(resp.String())
	return resp.Body(), err
}

// PostRequest makes an authenticated HTTP POST request to the specified endpoint with the given request body
func (d *D365) PostRequest(endpoint, values string) ([]byte, error) {
	if err := d.ensureToken(); err != nil {
		return nil, err
	}
	resp, err := d.Resty.R().
		SetHeader("Content-Type", "application/json; charset=utf-8").
		SetHeader("Authorization", fmt.Sprintf("Bearer %v", d.AccessToken)).
		SetHeader("Prefer", "return=representation").
		SetBody(values).
		Post(d.URL + "/api/data/v9.2/" + endpoint)

	fmt.Println(resp.String())
	return resp.Body(), err
}

// PatchRequest makes an authenticated HTTP PATCH request to the specified endpoint with the given request body
func (d *D365) PatchRequest(endpoint, values string) ([]byte, error) {
	if err := d.ensureToken(); err != nil {
		return nil, err
	}
	resp, err := d.Resty.R().
		SetHeader("Content-Type", "application/json; charset=utf-8").
		SetHeader("Authorization", fmt.Sprintf("Bearer %v", d.AccessToken)).
		SetHeader("Prefer", "return=representation").
		SetBody(values).
		Patch(d.URL + "/api/data/v9.2/" + endpoint)

	fmt.Println(endpoint)
	fmt.Println(resp.String())
	return resp.Body(), err
}
