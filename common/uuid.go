package common

import (
	"code.google.com/p/go-uuid/uuid"
)

type UUID uuid.UUID

func NewUUID() UUID {
	return UUID(uuid.NewUUID())
}

func UUIDFromString(s string) UUID {
	return UUID(uuid.Parse(s))
}

func (i UUID) String() string {
	return uuid.UUID(i).String()
}

func Equals(uuid1, uuid2 UUID) bool {
	return uuid.Equal(uuid.UUID(uuid1), uuid.UUID(uuid2))
}
