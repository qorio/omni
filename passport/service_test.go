package passport

import (
	"code.google.com/p/goprotobuf/proto"
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
	return &api.Account{
		Primary: &api.Identity{},
		Services: []*api.Service{
			&api.Service{
				Attributes: []*api.Attribute{
					&api.Attribute{
						Type:         api.Attribute_STRING.Enum(),
						EmbedInToken: proto.Bool(true),
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
	account.Id = proto.String(common.NewUUID().String())
	account.Primary.Phone = proto.String("111-222-3333")
	account.Services[0].Id = proto.String(common.NewUUID().String())
	account.Services[0].Status = proto.String("verified")
	account.Services[0].AccountId = proto.String("app-1-account-1")
	account.Services[0].Attributes[0].Key = proto.String("key-1")
	account.Services[0].Attributes[0].StringValue = proto.String("value-1")

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
	account.Id = proto.String(uuid.String())
	account.Primary.Phone = proto.String("111-222-4444")
	account.Services[0].Id = proto.String("app-1")
	account.Services[0].Status = proto.String("verified")
	account.Services[0].AccountId = proto.String("app-1-account-by-phone-1")
	account.Services[0].Attributes[0].Key = proto.String("key-1")
	account.Services[0].Attributes[0].StringValue = proto.String("value-1")

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
	account.Id = proto.String(uuid.String())
	account.Primary.Email = proto.String("foo@bar.com")
	account.Services[0].Id = proto.String("app-1")
	account.Services[0].Status = proto.String("verified")
	account.Services[0].AccountId = proto.String("app-1-account-by-email-1")
	account.Services[0].Attributes[0].Key = proto.String("key-1")
	account.Services[0].Attributes[0].StringValue = proto.String("value-1")

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

func TestFindByUsername(t *testing.T) {
	service, err := NewService(default_settings())
	assert.Equal(t, nil, err)
	service.dropDatabase()

	defer service.Close()
	t.Log("Started db client", service)

	uuid := common.NewUUID()
	err4 := service.DeleteAccount(uuid)
	assert.Equal(t, nil, err4)

	account := test_account()
	account.Id = proto.String(uuid.String())
	account.Primary.Username = proto.String("foouser")
	account.Services[0].Id = proto.String("app-1")
	account.Services[0].Status = proto.String("verified")
	account.Services[0].AccountId = proto.String("app-1-account-by-email-1")
	account.Services[0].Attributes[0].Key = proto.String("key-1")
	account.Services[0].Attributes[0].StringValue = proto.String("value-1")

	err = service.SaveAccount(account)
	assert.Equal(t, nil, err)

	account2, err2 := service.FindAccountByUsername("foouser")
	assert.Equal(t, nil, err2)
	t.Log("account2", account2)
	assert.Equal(t, account.String(), account2.String()) // compare the string representation

	account3, err3 := service.FindAccountByUsername("notfound")
	assert.Equal(t, (*api.Account)(nil), account3)
	assert.Equal(t, ERROR_NOT_FOUND, err3)
}

func testFindByOAuth2(t *testing.T) {
	service, err := NewService(default_settings())
	assert.Equal(t, nil, err)
	service.dropDatabase()

	defer service.Close()
	t.Log("Started db client", service)

	uuid := common.NewUUID()
	err4 := service.DeleteAccount(uuid)
	assert.Equal(t, nil, err4)

	account := test_account()
	account.Id = proto.String(uuid.String())
	account.Primary.Oauth2Provider = proto.String("oauth2_provider-1")
	account.Primary.Oauth2AccountId = proto.String("oauth2_account_id-1")
	account.Primary.Email = proto.String("foouser")
	account.Services[0].Id = proto.String("app-1")
	account.Services[0].Status = proto.String("verified")
	account.Services[0].AccountId = proto.String("app-1-account-by-email-1")
	account.Services[0].Attributes[0].Key = proto.String("key-1")
	account.Services[0].Attributes[0].StringValue = proto.String("value-1")

	err = service.SaveAccount(account)
	assert.Equal(t, nil, err)

	account2, err2 := service.FindAccountByUsername("foouser")
	assert.Equal(t, nil, err2)
	t.Log("account2", account2)
	assert.Equal(t, account.String(), account2.String()) // compare the string representation

	account3, err3 := service.FindAccountByOAuth2("oauth2_provider-1", "oauth2_account_id-1")
	assert.Equal(t, nil, err3)
	t.Log("account3", account3)
	assert.Equal(t, account.String(), account3.String()) // compare the string representation
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
	account.Id = proto.String(uuid.String())
	account.Primary.Phone = proto.String("111-222-5555")
	account.Services[0].Id = proto.String("app-1")
	account.Services[0].Status = proto.String("verified")
	account.Services[0].AccountId = proto.String("app-1-account-by-phone-1")
	account.Services[0].Attributes[0].Key = proto.String("key-1")
	account.Services[0].Attributes[0].StringValue = proto.String("value-1")

	err = service.SaveAccount(account)
	assert.Equal(t, nil, err)

	account2, err2 := service.FindAccountByPhone("111-222-5555")
	assert.Equal(t, nil, err2)
	t.Log("account2", account2)
	assert.Equal(t, account.String(), account2.String()) // compare the string representation

	// change the properties
	account2.Primary.Password = proto.String("password")
	err = service.SaveAccount(account2)
	assert.Equal(t, nil, err)

	account3, err2 := service.FindAccountByPhone("111-222-5555")
	assert.Equal(t, nil, err2)
	t.Log("account2", account3)
	assert.Equal(t, "password", account3.GetPrimary().GetPassword())

	// insert another
	account4 := &api.Account{}
	*account4 = *account
	account4.Primary.Phone = proto.String("222-333-4444")
	uuid4 := common.NewUUID()
	account4.Id = proto.String(uuid4.String())

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
	wait := start_server(t, "test", "new-user-registration", "/event/new-user-registration", "POST",
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
	account.Id = proto.String(uuid.String())
	account.Primary.Phone = proto.String("111-222-5555")
	account.Services[0].Id = proto.String("app-1")
	account.Services[0].Status = proto.String("verified")
	account.Services[0].AccountId = proto.String("app-1-account-by-phone-1")
	account.Services[0].Attributes[0].Key = proto.String("key-1")
	account.Services[0].Attributes[0].StringValue = proto.String("value-1")

	err = service.Send("test", "new-user-registration",
		struct{ Account *api.Account }{account},
		api.Methods[api.RegisterUser].CallbackBodyTemplate)

	assert.Equal(t, nil, err)

	testErr := wait(2)
	assert.Equal(t, nil, testErr)
}
