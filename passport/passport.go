package passport

import (
	"errors"
)

var (
	ERROR_ACCOUNT_NOT_FOUND = errors.New("account-not-found")
)

type Service interface {
	FindAccountByEmail(email string) (account *Account, err error)
}

type serviceImpl struct {
	settings Settings
}

func NewService(settings Settings) Service {
	return &serviceImpl{settings: settings}
}

func (this *serviceImpl) FindAccountByEmail(email string) (account *Account, err error) {
	return &Account{}, nil
}
