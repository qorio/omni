package passport

import (
	"github.com/qorio/api"
)

const (
	AccountUpdate api.AuthScope = iota
	AccountReadOnly
	RegisterNewUser
)

var AuthScopes = api.AuthScopes{
	AccountUpdate:   "account_update",
	AccountReadOnly: "account_readonly",
	RegisterNewUser: "register_user",
}

const (
	AuthUser api.ServiceMethod = iota
	AuthUserForService
	RegisterUser

	FetchAccount
	DeleteAccount
	CreateOrUpdateAccount
	UpdateAccountPrimaryLogin
	AddOrUpdateAccountService
	AddOrUpdateServiceAttribute
)

var Methods = api.ServiceMethods{

	AuthUser: api.MethodSpec{
		Doc: `
Authentication endpoint.
`,
		UrlRoute:     "/api/v1/auth",
		HttpMethod:   "POST",
		ContentTypes: []string{"application/json", "application/protobuf"},
		RequestBody: func() interface{} {
			return Login{}
		},
		ResponseBody: func() interface{} {
			return AuthResponse{}
		},
	},

	AuthUserForService: api.MethodSpec{
		Doc: `
Authentication endpoint.
`,
		UrlRoute:     "/api/v1/auth/{service}",
		HttpMethod:   "POST",
		ContentTypes: []string{"application/json", "application/protobuf"},
		RequestBody: func() interface{} {
			return Login{}
		},
		ResponseBody: func() interface{} {
			return AuthResponse{}
		},
	},

	RegisterUser: api.MethodSpec{
		AuthScope: AuthScopes[RegisterNewUser],
		Doc: `
User account registration.  On successful registration, the webhook of the corresponding
service will be called.  It is up to the service to then create any additional account
or service-related data.  The service then calls this service's AddOrUpdateAccountService
endpoint to update the mapping of service account id and any custom data to be passed to
the service on successful login auth.  The webhook is keyed by the CallbackEvent property
and is registered for the particular service.
`,
		UrlRoute:     "/api/v1/register/{service}",
		HttpMethod:   "POST",
		ContentTypes: []string{"application/json", "application/protobuf"},
		RequestBody: func() interface{} {
			return Login{}
		},
		ResponseBody: func() interface{} {
			return Login{}
		},
		// Calls the url webhook of given key for given service
		CallbackEvent:        api.EventKey("new-user-registration"),
		CallbackBodyTemplate: `{"id": "{{.Account.Id}}" }`,
	},

	FetchAccount: api.MethodSpec{
		AuthScope: AuthScopes[AccountReadOnly],
		Doc: `
Returns the account object.
`,
		UrlRoute:     "/api/v1/account/{id}",
		HttpMethod:   "GET",
		ContentTypes: []string{"application/json", "application/protobuf"},
		RequestBody:  nil,
		ResponseBody: func() interface{} {
			return Account{}
		},
	},

	DeleteAccount: api.MethodSpec{
		AuthScope: AuthScopes[AccountUpdate],
		Doc: `
Deletes the account.
`,
		UrlRoute:     "/api/v1/account/{id}",
		HttpMethod:   "DELETE",
		RequestBody:  nil,
		ResponseBody: nil,
	},

	CreateOrUpdateAccount: api.MethodSpec{
		AuthScope: AuthScopes[AccountUpdate],
		Doc: `
Create or update account. If id is missing, a new record will be created;
otherwise, an existing record will be overwritten with the POST value.
`,
		UrlRoute:     "/api/v1/account",
		HttpMethod:   "POST",
		ContentTypes: []string{"application/json", "application/protobuf"},
		RequestBody: func() interface{} {
			return Account{}
		},
		ResponseBody: nil,
	},

	UpdateAccountPrimaryLogin: api.MethodSpec{
		AuthScope: AuthScopes[AccountUpdate],
		Doc: `
Update primary login for account.
`,
		UrlRoute:     "/api/v1/account/{id}/primary",
		HttpMethod:   "POST",
		ContentTypes: []string{"application/json", "application/protobuf"},
		RequestBody: func() interface{} {
			return Login{}
		},
		ResponseBody: nil,
	},

	AddOrUpdateAccountService: api.MethodSpec{
		AuthScope: AuthScopes[AccountUpdate],
		Doc: `
Create or update a service / application in an existing account
`,
		UrlRoute:     "/api/v1/account/{id}/services",
		HttpMethod:   "POST",
		ContentTypes: []string{"application/json", "application/protobuf"},
		RequestBody: func() interface{} {
			return Service{}
		},
		ResponseBody: nil,
	},

	AddOrUpdateServiceAttribute: api.MethodSpec{
		AuthScope: AuthScopes[AccountUpdate],
		Doc: `
Create or update a service / application attribute in an existing account and application.
`,
		UrlRoute:     "/api/v1/account/{id}/service/{service}/attributes",
		HttpMethod:   "POST",
		ContentTypes: []string{"application/json", "application/protobuf"},
		RequestBody: func() interface{} {
			return Attribute{}
		},
		ResponseBody: nil,
	},
}
