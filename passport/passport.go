package passport

import (
	"errors"
)

var (
	ERROR_ACCOUNT_NOT_FOUND = errors.New("account-not-found")
)

type Service struct {
	settings Settings
}

func NewService(settings Settings) *Service {
	return &Service{settings: settings}
}

func (this *Service) FindAccountByEmail(email string) (account *Account, err error) {
	return &Account{}, nil
}
