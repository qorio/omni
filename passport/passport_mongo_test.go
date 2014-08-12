package passport

import (
	"github.com/bmizerany/assert"
	api "github.com/qorio/api/passport"
	"testing"
)

func default_settings() Settings {
	return Settings{
		Mongo: DbSettings{
			Hosts: []string{"localhost"},
			Db:    "passport_test",
		},
	}
}

func test_account() *api.Account {
	embed := true
	attr_type := api.Attribute_STRING

	return &api.Account{
		Primary: &api.Login{},
		Services: []*api.Application{
			&api.Application{
				Attributes: []*api.Attribute{
					&api.Attribute{
						Type:             &attr_type,
						EmbedSigninToken: &embed,
					},
				},
			},
		},
	}
}

func TestNewService(t *testing.T) {
	service, err := NewService(default_settings())
	assert.Equal(t, nil, err)

	defer service.Close()
	t.Log("Started db client", service)
}

func TestInsertGetAndDelete(t *testing.T) {
	service, err := NewService(default_settings())
	assert.Equal(t, nil, err)

	defer service.Close()
	t.Log("Started db client", service)

	err4 := service.DeleteAccount("account-1")
	assert.Equal(t, nil, err4)

	account := test_account()
	account.Id = ptr("account-1")
	account.Primary.Phone = ptr("111-222-3333")
	account.Services[0].Id = ptr("app-1")
	account.Services[0].Status = ptr("verified")
	account.Services[0].AccountId = ptr("app-1-account-1")
	account.Services[0].Attributes[0].Key = ptr("key-1")
	account.Services[0].Attributes[0].StringValue = ptr("value-1")

	err = service.SaveAccount(account)
	assert.Equal(t, nil, err)

	account2, err2 := service.GetAccount("account-1")
	assert.Equal(t, nil, err2)
	t.Log("account2", account2)
	assert.Equal(t, account.String(), account2.String()) // compare the string representation

	err5 := service.DeleteAccount("account-1")
	assert.Equal(t, nil, err5)

	_, err6 := service.GetAccount("account-1")
	assert.Equal(t, ERROR_NOT_FOUND, err6)
}

func TestFindByPhone(t *testing.T) {
	service, err := NewService(default_settings())
	assert.Equal(t, nil, err)

	defer service.Close()
	t.Log("Started db client", service)

	err4 := service.DeleteAccount("account-by-phone-1")
	assert.Equal(t, nil, err4)

	account := test_account()
	account.Id = ptr("account-by-phone-1")
	account.Primary.Phone = ptr("111-222-4444")
	account.Services[0].Id = ptr("app-1")
	account.Services[0].Status = ptr("verified")
	account.Services[0].AccountId = ptr("app-1-account-by-phone-1")
	account.Services[0].Attributes[0].Key = ptr("key-1")
	account.Services[0].Attributes[0].StringValue = ptr("value-1")

	err = service.SaveAccount(account)
	assert.Equal(t, nil, err)

	account2, err2 := service.FindAccountByPhone("111-222-4444")
	assert.Equal(t, nil, err2)
	t.Log("account2", account2)
	assert.Equal(t, account.String(), account2.String()) // compare the string representation

	account3, err3 := service.FindAccountByPhone("222-111-2222")
	assert.Equal(t, (*api.Account)(nil), account3)
	assert.Equal(t, ERROR_NOT_FOUND, err3)
}

func TestFindByEmail(t *testing.T) {
	service, err := NewService(default_settings())
	assert.Equal(t, nil, err)

	defer service.Close()
	t.Log("Started db client", service)

	err4 := service.DeleteAccount("account-by-email-1")
	assert.Equal(t, nil, err4)

	account := test_account()
	account.Id = ptr("account-by-email-1")
	account.Primary.Email = ptr("foo@bar.com")
	account.Services[0].Id = ptr("app-1")
	account.Services[0].Status = ptr("verified")
	account.Services[0].AccountId = ptr("app-1-account-by-email-1")
	account.Services[0].Attributes[0].Key = ptr("key-1")
	account.Services[0].Attributes[0].StringValue = ptr("value-1")

	err = service.SaveAccount(account)
	assert.Equal(t, nil, err)

	account2, err2 := service.FindAccountByEmail("foo@bar.com")
	assert.Equal(t, nil, err2)
	t.Log("account2", account2)
	assert.Equal(t, account.String(), account2.String()) // compare the string representation

	account3, err3 := service.FindAccountByEmail("notfound@gone.com")
	assert.Equal(t, (*api.Account)(nil), account3)
	assert.Equal(t, ERROR_NOT_FOUND, err3)
}

func TestFindByPhoneAndUpdate(t *testing.T) {
	service, err := NewService(default_settings())
	assert.Equal(t, nil, err)

	defer service.Close()
	t.Log("Started db client", service)

	err4 := service.DeleteAccount("account-by-phone-2")
	assert.Equal(t, nil, err4)

	account := test_account()
	account.Id = ptr("account-by-phone-2")
	account.Primary.Phone = ptr("111-222-5555")
	account.Services[0].Id = ptr("app-1")
	account.Services[0].Status = ptr("verified")
	account.Services[0].AccountId = ptr("app-1-account-by-phone-1")
	account.Services[0].Attributes[0].Key = ptr("key-1")
	account.Services[0].Attributes[0].StringValue = ptr("value-1")

	err = service.SaveAccount(account)
	assert.Equal(t, nil, err)

	account2, err2 := service.FindAccountByPhone("111-222-5555")
	assert.Equal(t, nil, err2)
	t.Log("account2", account2)
	assert.Equal(t, account.String(), account2.String()) // compare the string representation

	// change the properties
	account2.Primary.Password = ptr("password")
	err = service.SaveAccount(account2)
	assert.Equal(t, nil, err)

	account3, err2 := service.FindAccountByPhone("111-222-5555")
	assert.Equal(t, nil, err2)
	t.Log("account2", account3)
	assert.Equal(t, "password", account3.GetPrimary().GetPassword())

}
