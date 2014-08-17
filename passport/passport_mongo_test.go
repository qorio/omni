package passport

import (
	_ "encoding/json"
	"errors"
	"github.com/bmizerany/assert"
	api "github.com/qorio/api/passport"
	"github.com/qorio/omni/common"
	"github.com/qorio/omni/rest"
	"net/http"
	"testing"
)

func test_account() *api.Account {
	embed := true
	attr_type := api.Attribute_STRING

	return &api.Account{
		Primary: &api.Login{},
		Services: []*api.Service{
			&api.Service{
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
	service.dropDatabase()
}

func TestInsertGetAndDelete(t *testing.T) {
	service, err := NewService(default_settings())
	assert.Equal(t, nil, err)

	defer service.Close()
	t.Log("Started db client", service)

	service.dropDatabase()

	account := test_account()
	account.Id = ptr(common.NewUUID().String())
	account.Primary.Phone = ptr("111-222-3333")
	account.Services[0].Id = ptr(common.NewUUID().String())
	account.Services[0].Status = ptr("verified")
	account.Services[0].AccountId = ptr("app-1-account-1")
	account.Services[0].Attributes[0].Key = ptr("key-1")
	account.Services[0].Attributes[0].StringValue = ptr("value-1")

	err = service.SaveAccount(account)
	assert.Equal(t, nil, err)

	account2, err2 := service.GetAccount(common.UUIDFromString(*account.Id))
	assert.Equal(t, nil, err2)
	t.Log("account2", account2)
	assert.Equal(t, account.String(), account2.String()) // compare the string representation

	err5 := service.DeleteAccount(common.UUIDFromString(*account.Id))
	assert.Equal(t, nil, err5)

	_, err6 := service.GetAccount(common.UUIDFromString(*account.Id))
	assert.Equal(t, ERROR_NOT_FOUND, err6)
}

func TestFindByPhone(t *testing.T) {
	service, err := NewService(default_settings())
	assert.Equal(t, nil, err)
	service.dropDatabase()

	defer service.Close()
	t.Log("Started db client", service)

	uuid := common.NewUUID()
	err4 := service.DeleteAccount(uuid)
	assert.Equal(t, nil, err4)

	account := test_account()
	account.Id = ptr(uuid.String())
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
	service.dropDatabase()

	defer service.Close()
	t.Log("Started db client", service)

	uuid := common.NewUUID()
	err4 := service.DeleteAccount(uuid)
	assert.Equal(t, nil, err4)

	account := test_account()
	account.Id = ptr(uuid.String())
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
	service.dropDatabase()

	defer service.Close()
	t.Log("Started db client", service)

	uuid := common.NewUUID()
	err4 := service.DeleteAccount(uuid)
	assert.Equal(t, nil, err4)

	account := test_account()
	account.Id = ptr(uuid.String())
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

	// insert another
	account4 := &api.Account{}
	*account4 = *account
	account4.Primary.Phone = ptr("222-333-4444")
	uuid4 := common.NewUUID()
	account4.Id = ptr(uuid4.String())

	err = service.SaveAccount(account4)
	assert.Equal(t, nil, err)
	account5, err2 := service.FindAccountByPhone("222-333-4444")
	assert.Equal(t, nil, err2)
	t.Log("account5", account5)
	assert.Equal(t, account4.String(), account5.String())

}

func TestWebHooks(t *testing.T) {

	service, err := NewService(default_settings())
	assert.Equal(t, nil, err)
	service.dropDatabase()
	service.Close()

	// restart
	service, err = NewService(default_settings())

	defer service.Close()
	t.Log("Started db client", service)

	uuid := common.NewUUID()
	wait := start_server(t, ":9999", "/event/new-user-registration", "POST",
		func(resp http.ResponseWriter, req *http.Request) error {
			t.Log("Received post", req.Header)
			// check header
			if _, has := req.Header[rest.WebHookHmacHeader]; !has {
				return errors.New("no hmac header")
			}

			v := from_json(make(map[string]interface{}), req.Body, t).(map[string]interface{})
			t.Log("Post body", v, "err", err)

			if err != nil {
				return err
			}

			if id, has := v["id"]; has {
				if id != uuid.String() {
					return errors.New("id does not match")
				}
			} else {
				return errors.New("no id property")
			}
			return nil
		})

	account := test_account()
	account.Id = ptr(uuid.String())
	account.Primary.Phone = ptr("111-222-5555")
	account.Services[0].Id = ptr("app-1")
	account.Services[0].Status = ptr("verified")
	account.Services[0].AccountId = ptr("app-1-account-by-phone-1")
	account.Services[0].Attributes[0].Key = ptr("key-1")
	account.Services[0].Attributes[0].StringValue = ptr("value-1")

	err = service.Send("test", "new-user-registration",
		struct{ Account *api.Account }{account},
		api.Methods[api.RegisterUser].CallbackBodyTemplate)

	assert.Equal(t, nil, err)

	testErr := wait(2)
	assert.Equal(t, nil, testErr)
}
