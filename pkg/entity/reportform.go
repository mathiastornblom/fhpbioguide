package entity

import "time"

type Form struct {
	ID     ID `gorm:"primary_key"` // Unique identifier for the Form, marked as a primary key for GORM.
	Name   string                  // Human-readable name of the Form.
	Text   string                  // Descriptive text associated with the Form.
	Type   int                     // Numeric code or identifier indicating the Form's type.
	Date   time.Time               // Timestamp indicating when the Form was created or relevant for.
	Events []Event                 // Slice of Event structs associated with this Form.
}


type Event struct {
	ID             ID `gorm:"primary_key"`   // Unique identifier for the Event.
	Form_type      int                      // May indicate the type of form this event is associated with.
	Name           string                   // Human-readable name of the Event.
	Text           string                   // Descriptive text associated with the Event.
	Date           string                   // The date (as a string) the Event is scheduled for or relevant.
	Amount         int                      // An amount associated with the Event, possibly a cost or a limit.
	Url            string                   // A URL for more information or for an action related to the Event.
	ExpirationTime time.Time                // Timestamp indicating when the Event expires or is no longer relevant.
	FormID         ID `gorm:"primary_key"`  // The ID of the Form this Event is associated with.
	Discounts      string                   // Information about any discounts associated with the Event.
}

// Fields map directly to a Dynamics 365 schema for booking entities, with JSON tags indicating
// how each Go field maps to JSON fields in Dynamics 365 API responses
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

// A wrapper for a slice of Booking structs.
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
