package passport

import (
	omni_http "github.com/qorio/omni/http"
)

var (
	AuthenticateUser = &omni_http.ServiceMethod{
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
		Doc: `
Authentication endpoint.
`,
	}

	FetchAccount = &omni_http.ServiceMethod{
		Name:         "FetchAccount",
		UrlRoute:     "/api/v1/account/{id}",
		HttpMethod:   "GET",
		ContentTypes: []string{"application/json", "application/protobuf"},
		RequestBody:  nil,
		ResponseBody: func() interface{} {
			return Account{}
		},
		Doc: `
Returns the account object.
`,
	}

	DeleteAccount = &omni_http.ServiceMethod{
		Name:         "DeleteAccount",
		UrlRoute:     "/api/v1/account/{id}",
		HttpMethod:   "DELETE",
		RequestBody:  nil,
		ResponseBody: nil,
		Doc: `
Deletes the account.
`,
	}

	CreateOrUpdateAccount = &omni_http.ServiceMethod{
		Name:         "CreateOrUpdateAccount",
		UrlRoute:     "/api/v1/account",
		HttpMethod:   "POST",
		ContentTypes: []string{"application/json", "application/protobuf"},
		RequestBody: func() interface{} {
			return Account{}
		},
		ResponseBody: nil,
		Doc: `
Create or update account. If id is missing, a new record will be created;
otherwise, an existing record will be overwritten with the POST value.
`,
	}

	UpdateAccountPrimaryLogin = &omni_http.ServiceMethod{
		Name:         "UpdateAccountPrimaryLogin",
		UrlRoute:     "/api/v1/account/{id}/primary",
		HttpMethod:   "POST",
		ContentTypes: []string{"application/json", "application/protobuf"},
		RequestBody: func() interface{} {
			return Login{}
		},
		ResponseBody: nil,
		Doc: `
Update primary login for account.
`,
	}

	AddOrUpdateAccountService = &omni_http.ServiceMethod{
		Name:         "AddOrUpdateUpdateAccountService",
		UrlRoute:     "/api/v1/account/{id}/services",
		HttpMethod:   "POST",
		ContentTypes: []string{"application/json", "application/protobuf"},
		RequestBody: func() interface{} {
			return Application{}
		},
		ResponseBody: nil,
		Doc: `
Create or update a service / application in an existing account
`,
	}

	AddOrUpdateServiceAttribute = &omni_http.ServiceMethod{
		Name:         "AddOrUpdateUpdateServiceAttribute",
		UrlRoute:     "/api/v1/account/{id}/service/{applicationId}/attributes",
		HttpMethod:   "POST",
		ContentTypes: []string{"application/json", "application/protobuf"},
		RequestBody: func() interface{} {
			return Attribute{}
		},
		ResponseBody: nil,
		Doc: `
Create or update a service / application attribute in an existing account and application.
`,
	}

	UserAuth = omni_http.Publish(
		AuthenticateUser,
	)

	ApiService = omni_http.Publish(
		FetchAccount,
		DeleteAccount,
		CreateOrUpdateAccount,
		UpdateAccountPrimaryLogin,
		AddOrUpdateAccountService,
		AddOrUpdateServiceAttribute,
	)
)
