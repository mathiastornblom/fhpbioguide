package handler

import "strings"

// ticketTypeNames are the ticket type ids documented for CashReportsDistributorExport 1.4+
// (ticketdetail type-id). BioGuiden may add ids without notice but never changes or removes
// them, so an unknown id is handled by ticketCategory instead of being dropped.
var ticketTypeNames = map[string]string{
	"1000": "Ordinarie",
	"1010": "Matiné",
	"2010": "Företagsbiljett: Filmstaden",
	"2020": "Företagsbiljett: Svenska Bio",
	"2030": "Företagsbiljett: Övriga",
	"3010": "Fribiljett: Passepartout",
	"3020": "Fribiljett: Personal",
	"3030": "Fribiljett: Press",
	"3040": "Fribiljett: Uttag biografägare",
	"3050": "Fribiljett: Uttag distributör",
	"3060": "Fribiljett: Övriga",
	"4000": "Rabatt",
	"5010": "Medlemsbiljett: Biopasset",
	"5011": "Medlemsbiljett: Biopasset",
}

// ticketCategory builds the text stored in D365 new_name / new_ticketcategory for one
// ticket detail. Versions before 1.4 delivered a free-text category. From 1.4 the
// category is replaced by type-id plus specification, where the specification carries the
// theatre's own wording (and the old category text when a theatre still reports the old
// way). The specification is therefore preferred, then the name of the type-id, then the
// raw id, so a new type-id never produces an empty category.
func ticketCategory(typeID, specification, legacyCategory string) string {
	if s := strings.TrimSpace(specification); s != "" {
		return s
	}
	if c := strings.TrimSpace(legacyCategory); c != "" {
		return c
	}
	typeID = strings.TrimSpace(typeID)
	if name, ok := ticketTypeNames[typeID]; ok {
		return name
	}
	if typeID != "" {
		return "Biljettyp " + typeID
	}
	return ""
}
