package passport

import (
	"errors"
)

var (
	ERROR_ACCOUNT_NOT_FOUND = errors.New("account-not-found")
)

type Service interface {
	FindAccountByEmail(email string) (account *Account, err error)
	FindAccountByPhone(phone string) (account *Account, err error)
}

func NewService(settings Settings) Service {
	return &serviceImpl{settings: settings}
}

type serviceImpl struct {
	settings Settings
}

func (this *serviceImpl) FindAccountByEmail(email string) (account *Account, err error) {
	return nil, ERROR_ACCOUNT_NOT_FOUND
}

func (this *serviceImpl) FindAccountByPhone(email string) (account *Account, err error) {
	return nil, ERROR_ACCOUNT_NOT_FOUND
}
