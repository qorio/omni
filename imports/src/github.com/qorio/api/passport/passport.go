package passport

import (
	"github.com/qorio/api"
)

const (
	AuthUser api.ServiceMethod = iota
	FetchAccount
	DeleteAccount
	CreateOrUpdateAccount
	UpdateAccountPrimaryLogin
	AddOrUpdateAccountService
	AddOrUpdateServiceAttribute
)

var Methods = map[api.ServiceMethod]*api.MethodSpec{

	AuthUser: &api.MethodSpec{
		RequiresAuth: false,
		Doc: `
Authentication endpoint.
`,
		Name:         "AuthenticateUser",
		UrlRoute:     "/api/v1/auth",
		HttpMethod:   "POST",
		ContentTypes: []string{"application/json", "application/protobuf"},
		RequestBody: func() interface{} {
			return AuthRequest{}
		},
		ResponseBody: func() interface{} {
			return AuthResponse{}
		},
	},

	FetchAccount: &api.MethodSpec{
		RequiresAuth: false,
		Doc: `
Returns the account object.
`,
		Name:         "FetchAccount",
		UrlRoute:     "/api/v1/account/{id}",
		HttpMethod:   "GET",
		ContentTypes: []string{"application/json", "application/protobuf"},
		RequestBody:  nil,
		ResponseBody: func() interface{} {
			return Account{}
		},
	},

	DeleteAccount: &api.MethodSpec{
		RequiresAuth: false,
		Doc: `
Deletes the account.
`,
		Name:         "DeleteAccount",
		UrlRoute:     "/api/v1/account/{id}",
		HttpMethod:   "DELETE",
		RequestBody:  nil,
		ResponseBody: nil,
	},

	CreateOrUpdateAccount: &api.MethodSpec{
		RequiresAuth: false,
		Doc: `
Create or update account. If id is missing, a new record will be created;
otherwise, an existing record will be overwritten with the POST value.
`,
		Name:         "CreateOrUpdateAccount",
		UrlRoute:     "/api/v1/account",
		HttpMethod:   "POST",
		ContentTypes: []string{"application/json", "application/protobuf"},
		RequestBody: func() interface{} {
			return Account{}
		},
		ResponseBody: nil,
	},

	UpdateAccountPrimaryLogin: &api.MethodSpec{
		RequiresAuth: false,
		Doc: `
Update primary login for account.
`,
		Name:         "UpdateAccountPrimaryLogin",
		UrlRoute:     "/api/v1/account/{id}/primary",
		HttpMethod:   "POST",
		ContentTypes: []string{"application/json", "application/protobuf"},
		RequestBody: func() interface{} {
			return Login{}
		},
		ResponseBody: nil,
	},

	AddOrUpdateAccountService: &api.MethodSpec{
		RequiresAuth: false,
		Doc: `
Create or update a service / application in an existing account
`,
		Name:         "AddOrUpdateUpdateAccountService",
		UrlRoute:     "/api/v1/account/{id}/services",
		HttpMethod:   "POST",
		ContentTypes: []string{"application/json", "application/protobuf"},
		RequestBody: func() interface{} {
			return Application{}
		},
		ResponseBody: nil,
	},

	AddOrUpdateServiceAttribute: &api.MethodSpec{
		RequiresAuth: false,
		Doc: `
Create or update a service / application attribute in an existing account and application.
`,
		Name:         "AddOrUpdateUpdateServiceAttribute",
		UrlRoute:     "/api/v1/account/{id}/service/{applicationId}/attributes",
		HttpMethod:   "POST",
		ContentTypes: []string{"application/json", "application/protobuf"},
		RequestBody: func() interface{} {
			return Attribute{}
		},
		ResponseBody: nil,
	},
}
