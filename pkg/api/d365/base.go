package d365

import (
	"encoding/json"
	"fmt"

	"github.com/go-resty/resty/v2"
)

type Token struct {
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	ExtExpiresIn int    `json:"ext_expires_in"`
	AccessToken  string `json:"access_token"`
}

type D365 struct {
	Resty        *resty.Client
	URL          string
	TenantID     string
	ClientID     string
	ClientSecret string
	AccessToken  string
}

func (d *D365) AuthenticateApi() {
	resp, _ := d.Resty.R().
		SetHeader("Content-Type", "application/x-www-form-urlencoded").
		SetFormData(map[string]string{
			"client_id":     d.ClientID,
			"resource":      d.URL,
			"client_secret": d.ClientSecret,
			"grant_type":    "client_credentials"}).
		Post("https://login.microsoftonline.com/" + d.TenantID + "/oauth2/token")

	token := Token{}

	json.Unmarshal([]byte(resp.String()), &token)

	d.AccessToken = token.AccessToken
}

func (d *D365) GetRequest(endpoint string) ([]byte, error) {
	resp, err := d.Resty.R().
		SetHeader("Authorization", fmt.Sprintf("Bearer %v", d.AccessToken)).
		Get(d.URL + "/api/data/v9.2/" + endpoint)

	if err != nil {
		return nil, err
	}
	fmt.Println(resp.String())

	return resp.Body(), nil
}

func (d *D365) PostRequest(endpoint, values string) ([]byte, error) {
	resp, err := d.Resty.R().
		SetHeader("Content-Type", "application/json; charset=utf-8").
		SetHeader("Authorization", fmt.Sprintf("Bearer %v", d.AccessToken)).
		SetHeader("Prefer", "return=representation").
		SetBody(values).
		Post(d.URL + "/api/data/v9.2/" + endpoint)

	fmt.Println(resp.String())

	return resp.Body(), err
}

func (d *D365) PatchRequest(endpoint, values string) ([]byte, error) {
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
