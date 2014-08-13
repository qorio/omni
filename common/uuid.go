package common

import (
	"code.google.com/p/go-uuid/uuid"
)

func NewUUID() uuid.UUID {
	return uuid.NewUUID()
}

func UUIDFromString(s string) uuid.UUID {
	return uuid.Parse(s)
}

func Equals(uuid1, uuid2 uuid.UUID) bool {
	return uuid.Equal(uuid1, uuid2)
}
