package entity

import "time"

type Form struct {
	ID     ID `gorm:"primary_key"`
	Name   string
	Text   string
	Type   int
	Date   time.Time
	Events []Event
}

type Event struct {
	ID             ID `gorm:"primary_key"`
	Form_type      int
	Name           string
	Text           string
	Date           string
	Amount         int
	Url            string
	ExpirationTime time.Time
	FormID         ID `gorm:"primary_key"`
	Discounts      string
}

type Booking struct {
	ID                 string `json:"new_bokningarkundid"`
	Name               string `json:"new_name"`
	ShowDate           string `json:"new_showdate"`
	Url                string `json:"new_forkopsurl"`
	Owningbusinessunit string `json:"_owningbusinessunit_value"`
	Discounts          string `json:"new_rabatter@OData.Community.Display.V1.FormattedValue"`
	CustomerID         string `json:"_new_customer_value"`
	Customer           struct {
		Name      string `json:"name"`
		Accountid string `json:"accountid"`
	} `json:"new_customer_account"`
	Product struct {
		Name      string `json:"name"`
		Productid string `json:"productid"`
	} `json:"new_product"`
}

type Bookings struct {
	Value []Booking `json:"value"`
}

type Booking2 struct {
	Bookingnumber string `json:"new_bookingnumber"`
	ProductID     string `json:"_new_product_value"`
	Tax           int    `json:"new_vat"`
	Tocreator     int    `json:"new_tosupplier"`
	Member        int    `json:"new_member"`
	Showdate      string `json:"new_showdate"`
	ID            string `json:"new_bokningarkundid"`
	Productnumber string `json:"new_productnumber"`
	CustomerID    string `json:"_new_customer_value"`
	LokalID       string `json:"_new_lokaler_value"`
	Name          string `json:"new_name"`
	Theaternumber string `json:"_new_theaternumber_value"`
}
