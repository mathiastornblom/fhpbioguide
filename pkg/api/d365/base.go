// Package d365 handles Dynamics 365 OAuth2 authentication and REST requests.
package d365

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/go-resty/resty/v2"
)

// flexInt unmarshals a JSON value that may be either a number or a quoted string.
// The legacy oauth2/token endpoint returns expires_in as a string.
type flexInt int

func (f *flexInt) UnmarshalJSON(b []byte) error {
	var n int
	if err := json.Unmarshal(b, &n); err == nil {
		*f = flexInt(n)
		return nil
	}
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return err
	}
	*f = flexInt(n)
	return nil
}

// Token maps the JSON structure of the OAuth token response.
type Token struct {
	TokenType    string  `json:"token_type"`
	ExpiresIn    flexInt `json:"expires_in"`
	ExtExpiresIn flexInt `json:"ext_expires_in"`
	AccessToken  string  `json:"access_token"`
}

// D365 holds configuration and state for Dynamics 365 API calls.
type D365 struct {
	Resty        *resty.Client
	URL          string
	TenantID     string
	ClientID     string
	ClientSecret string
	AccessToken  string
	ExpiresAt    time.Time
	Logger       *slog.Logger // optional; nil disables D365-level debug logging
}

func (d *D365) isTokenValid() bool {
	return d.AccessToken != "" && time.Now().Before(d.ExpiresAt)
}

// AuthenticateApi performs OAuth authentication to obtain an access token.
func (d *D365) AuthenticateApi() error {
	resp, err := d.Resty.R().
		SetHeader("Content-Type", "application/x-www-form-urlencoded").
		SetFormData(map[string]string{
			"client_id":     d.ClientID,
			"resource":      d.URL,
			"client_secret": d.ClientSecret,
			"grant_type":    "client_credentials",
		}).
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
	d.ExpiresAt = time.Now().Add(time.Duration(int(token.ExpiresIn)-60) * time.Second)

	if d.Logger != nil {
		d.Logger.Info("D365 auth success", "expires_in", token.ExpiresIn)
	}
	return nil
}

func (d *D365) ensureToken() error {
	if d.isTokenValid() {
		return nil
	}
	return d.AuthenticateApi()
}

// GetRequest makes an authenticated HTTP GET request.
func (d *D365) GetRequest(endpoint string) ([]byte, error) {
	if err := d.ensureToken(); err != nil {
		return nil, err
	}
	resp, err := d.Resty.R().
		SetHeader("Authorization", fmt.Sprintf("Bearer %v", d.AccessToken)).
		Get(d.URL + "/api/data/v9.2/" + endpoint)

	if d.Logger != nil {
		d.Logger.Debug("D365 GET", "endpoint", endpoint, "status", resp.StatusCode())
	}
	return resp.Body(), err
}

// PostRequest makes an authenticated HTTP POST request.
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

	if d.Logger != nil {
		d.Logger.Debug("D365 POST", "endpoint", endpoint, "status", resp.StatusCode())
	}
	return resp.Body(), err
}

// DeleteRequest makes an authenticated HTTP DELETE request.
func (d *D365) DeleteRequest(endpoint string) error {
	if err := d.ensureToken(); err != nil {
		return err
	}
	_, err := d.Resty.R().
		SetHeader("Authorization", fmt.Sprintf("Bearer %v", d.AccessToken)).
		Delete(d.URL + "/api/data/v9.2/" + endpoint)
	return err
}

// PatchRequest makes an authenticated HTTP PATCH request.
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

	if d.Logger != nil {
		d.Logger.Debug("D365 PATCH", "endpoint", endpoint, "status", resp.StatusCode())
	}
	return resp.Body(), err
}
