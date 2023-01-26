package entity

import "github.com/google/uuid"

//ID entity ID
type ID = uuid.UUID

//NewID create a entity ID
func NewID() ID {
	return ID(uuid.New())
}

//StringToID convert a string to an entity ID
func StringToID(s string) (ID, error) {
	id, err := uuid.Parse(s)
	return ID(id), err
}

//StringToID convert a string to an entity ID
func UnsafeStringToID(s string) ID {
	id, _ := uuid.Parse(s)
	return ID(id)
}
