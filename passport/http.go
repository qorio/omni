package passport

import (
	"github.com/golang/glog"
	"github.com/gorilla/mux"
	omni_auth "github.com/qorio/omni/auth"
	omni_common "github.com/qorio/omni/common"
	omni_http "github.com/qorio/omni/http"
	"net/http"
	"strings"
)

type EndPoint struct {
	settings Settings
	service  Service
	engine   omni_http.Engine
}

func defaultResolveApplicationId(req *http.Request) string {
	return req.URL.Host
}

func NewApiEndPoint(settings Settings, auth *omni_auth.Service, service Service) (api *EndPoint, err error) {
	api = &EndPoint{
		settings: settings,
		service:  service,
		engine:   omni_http.NewEngine(auth),
	}

	AuthenticateUser.Handler = api.ApiAuthenticate

	FetchAccount.Handler = api.ApiGetAccount

	CreateOrUpdateAccount.Handler = api.ApiSaveAccount
	UpdateAccountPrimaryLogin.Handler = api.ApiSaveAccountPrimary
	AddOrUpdateAccountService.Handler = api.ApiSaveAccountService
	AddOrUpdateServiceAttribute.Handler = api.ApiSaveAccountServiceAttribute
	DeleteAccount.Handler = api.ApiDeleteAccount

	// engine itself hosts two services
	api.engine.Bind(UserAuth)
	api.engine.Bind(ApiService)

	return api, nil
}

func (this *EndPoint) ServeHTTP(resp http.ResponseWriter, request *http.Request) {
	this.engine.ServeHTTP(resp, request)
}

// Authenticates and returns a token as the response
func (this *EndPoint) ApiAuthenticate(resp http.ResponseWriter, req *http.Request) {
	omni_http.SetCORSHeaders(resp)

	request := AuthenticateUser.RequestBody().(AuthRequest)
	err := this.engine.Unmarshal(req, &request)
	if err != nil {
		this.engine.RenderJsonError(resp, req, err.Error(), http.StatusBadRequest)
		return
	}

	// do the lookup here...
	account, err := this.findAccount(request.GetEmail(), request.GetPhone())

	switch {
	case err == ERROR_NOT_FOUND:
		this.engine.RenderJsonError(resp, req, "error-account-not-found", http.StatusUnauthorized)
		return
	case err == ERROR_MISSING_INPUT:
		this.engine.RenderJsonError(resp, req, err.Error(), http.StatusBadRequest)
		return

	case err != nil:
		this.engine.RenderJsonError(resp, req, "error-lookup-account", http.StatusInternalServerError)
		return
	case err == nil && account.Primary.GetPassword() != request.GetPassword():
		this.engine.RenderJsonError(resp, req, "error-bad-credentials", http.StatusUnauthorized)
		return
	}

	// now look for the application
	requestedApplicationId := request.GetApplication()
	if requestedApplicationId == "" {
		if this.settings.ResolveApplicationId != nil {
			requestedApplicationId = this.settings.ResolveApplicationId(req)
		} else {
			requestedApplicationId = defaultResolveApplicationId(req)
		}
	}
	var application *Application
	for _, test := range account.GetServices() {
		if test.GetId() == requestedApplicationId {
			application = test
			break
		}
	}

	if application == nil {
		this.engine.RenderJsonError(resp, req, "error-not-a-member", http.StatusUnauthorized)
		return
	}

	// encode the token
	token := this.engine.NewAuthToken()
	token.Add("@id", application.GetId()).
		Add("@status", application.GetStatus()).
		Add("@accountId", application.GetAccountId()).
		Add("@permissions", strings.Join(application.GetPermissions(), ","))

	for _, attribute := range application.GetAttributes() {
		if attribute.GetEmbedSigninToken() {
			switch attribute.GetType() {
			case Attribute_STRING:
				token.Add(attribute.GetKey(), attribute.GetStringValue())
			case Attribute_NUMBER:
				token.Add(attribute.GetKey(), attribute.GetNumberValue())
			case Attribute_BOOL:
				token.Add(attribute.GetKey(), attribute.GetBoolValue())
			case Attribute_BLOB:
				token.Add(attribute.GetKey(), attribute.GetBlobValue())
			}
		}
	}
	tokenString, err := this.engine.SignedString(token)
	if err != nil {
		glog.Warningln("error-generating-auth-token", err)
		this.engine.RenderJsonError(resp, req, "cannot-generate-auth-token", http.StatusInternalServerError)
		return
	}

	// Response
	authResponse := AuthenticateUser.ResponseBody().(AuthResponse)
	authResponse.Token = &tokenString

	err = this.engine.Marshal(req, &authResponse, resp)
	if err != nil {
		this.engine.RenderJsonError(resp, req, "malformed-response", http.StatusInternalServerError)
		return
	}
}

func (this *EndPoint) ApiSaveAccount(resp http.ResponseWriter, req *http.Request) {
	omni_http.SetCORSHeaders(resp)

	account := CreateOrUpdateAccount.RequestBody().(Account)
	err := this.engine.Unmarshal(req, &account)
	if err != nil {
		this.engine.RenderJsonError(resp, req, err.Error(), http.StatusBadRequest)
		return
	}

	if account.GetPrimary().GetPhone() == "" && account.GetPrimary().GetEmail() == "" {
		this.engine.RenderJsonError(resp, req, ERROR_MISSING_INPUT.Error(), http.StatusBadRequest)
		return
	}

	hasLoginId := account.GetPrimary().GetId() != ""
	hasAccountId := account.GetId() != ""

	switch {
	case hasLoginId && hasAccountId:
		// simple update case

	case hasLoginId && !hasAccountId:
		// not allowed -- should not start a new one.
		this.engine.RenderJsonError(resp, req, "cannot-transfer-login-to-new-account",
			http.StatusBadRequest)
		return

	case !hasLoginId && hasAccountId:
		// this is changing primary login of the account
		// check availability of phone/email
		existing, _ := this.findAccount(account.GetPrimary().GetEmail(),
			account.GetPrimary().GetPhone())
		if existing != nil {
			this.engine.RenderJsonError(resp, req, "error-duplicate", http.StatusConflict)
			return
		}

		// Ok - assign login id
		uuid, _ := omni_common.NewUUID()
		account.GetPrimary().Id = &uuid

	case !hasLoginId && !hasAccountId:
		// this is new login and account
		// check availability of phone/email
		existing, _ := this.findAccount(account.GetPrimary().GetEmail(),
			account.GetPrimary().GetPhone())
		if existing != nil {
			this.engine.RenderJsonError(resp, req, "error-duplicate", http.StatusConflict)
			return
		}

		// Ok - assign new login id
		uuid, _ := omni_common.NewUUID()
		account.GetPrimary().Id = &uuid
		// Ok - assign new account id
		uuid, _ = omni_common.NewUUID()
		account.Id = &uuid

	}
	if account.GetPrimary().GetId() == "" {

		// If no id, then check to see if the email or phone has
		// been taken already

	} else {

		if account.GetId() == "" {

		}
	}

	if account.GetId() == "" {

		uuid, _ := omni_common.NewUUID()
		account.Id = &uuid
	}
	err = this.service.SaveAccount(&account)
	if err != nil {
		this.engine.RenderJsonError(resp, req, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (this *EndPoint) ApiSaveAccountPrimary(resp http.ResponseWriter, req *http.Request) {
	omni_http.SetCORSHeaders(resp)
	vars := mux.Vars(req)
	id := vars["id"]

	login := UpdateAccountPrimaryLogin.RequestBody().(Login)
	err := this.engine.Unmarshal(req, &login)
	if err != nil {
		this.engine.RenderJsonError(resp, req, err.Error(), http.StatusBadRequest)
		return
	}

	if id == "" {
		this.engine.RenderJsonError(resp, req, "error-missing-id", http.StatusBadRequest)
		return
	}

	account, err := this.service.GetAccount(id)

	switch {
	case err == ERROR_NOT_FOUND:
		this.engine.RenderJsonError(resp, req, "error-account-not-found", http.StatusNotFound)
		return
	case err != nil:
		this.engine.RenderJsonError(resp, req, err.Error(), http.StatusInternalServerError)
		return
	}

	switch {
	case login.GetPhone() == "" && login.GetEmail() == "":
		this.engine.RenderJsonError(resp, req, "error-missing-email-or-phone", http.StatusBadRequest)
		return
	case login.GetPassword() == "":
		this.engine.RenderJsonError(resp, req, "error-missing-password", http.StatusBadRequest)
		return
	}

	if login.GetId() == "" {
		uuid, _ := omni_common.NewUUID()
		login.Id = &uuid
	}

	// update the primary
	account.Primary = &login

	err = this.service.SaveAccount(account)
	if err != nil {
		this.engine.RenderJsonError(resp, req, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (this *EndPoint) ApiSaveAccountService(resp http.ResponseWriter, req *http.Request) {
	omni_http.SetCORSHeaders(resp)
	vars := mux.Vars(req)
	id := vars["id"]

	application := AddOrUpdateAccountService.RequestBody().(Application)
	err := this.engine.Unmarshal(req, &application)
	if err != nil {
		this.engine.RenderJsonError(resp, req, err.Error(), http.StatusBadRequest)
		return
	}

	if id == "" {
		this.engine.RenderJsonError(resp, req, "error-missing-id", http.StatusBadRequest)
		return
	}

	account, err := this.service.GetAccount(id)

	switch {
	case err == ERROR_NOT_FOUND:
		this.engine.RenderJsonError(resp, req, "error-account-not-found", http.StatusNotFound)
		return
	case err != nil:
		this.engine.RenderJsonError(resp, req, err.Error(), http.StatusInternalServerError)
		return
	}

	// find the application by id and replace it
	if len(account.GetServices()) == 0 {
		account.Services = []*Application{
			&application,
		}
	} else {
		match := false
		for i, app := range account.GetServices() {
			if app.GetId() == application.GetId() {
				account.Services[i] = &application
				match = true
				break
			}
		}
		if !match {
			account.Services = append(account.Services, &application)
		}
	}

	err = this.service.SaveAccount(account)
	if err != nil {
		this.engine.RenderJsonError(resp, req, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (this *EndPoint) ApiSaveAccountServiceAttribute(resp http.ResponseWriter, req *http.Request) {
	omni_http.SetCORSHeaders(resp)
	vars := mux.Vars(req)
	id := vars["id"]
	applicationId := vars["applicationId"]

	attribute := AddOrUpdateServiceAttribute.RequestBody().(Attribute)
	err := this.engine.Unmarshal(req, &attribute)
	if err != nil {
		this.engine.RenderJsonError(resp, req, err.Error(), http.StatusBadRequest)
		return
	}

	if id == "" {
		this.engine.RenderJsonError(resp, req, "error-missing-id", http.StatusBadRequest)
		return
	}

	account, err := this.service.GetAccount(id)

	switch {
	case err == ERROR_NOT_FOUND:
		this.engine.RenderJsonError(resp, req, "error-account-not-found", http.StatusNotFound)
		return
	case err != nil:
		this.engine.RenderJsonError(resp, req, err.Error(), http.StatusInternalServerError)
		return
	}

	// find the application by id and update its attributes
	var application *Application
	for _, app := range account.GetServices() {
		if app.GetId() == applicationId {
			application = app
			break
		}
	}

	if application == nil {
		this.engine.RenderJsonError(resp, req, "error-application-id-not-found", http.StatusBadRequest)
		return
	}

	if len(application.GetAttributes()) == 0 {
		application.Attributes = []*Attribute{
			&attribute,
		}
	} else {
		match := false
		for i, attr := range application.GetAttributes() {
			if attr.GetKey() == attribute.GetKey() {
				application.Attributes[i] = &attribute
				match = true
				break
			}
		}
		if !match {
			application.Attributes = append(application.Attributes, &attribute)
		}
	}

	err = this.service.SaveAccount(account)
	if err != nil {
		this.engine.RenderJsonError(resp, req, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (this *EndPoint) ApiGetAccount(resp http.ResponseWriter, req *http.Request) {
	omni_http.SetCORSHeaders(resp)
	vars := mux.Vars(req)
	id := vars["id"]

	if id == "" {
		this.engine.RenderJsonError(resp, req, "error-missing-id", http.StatusBadRequest)
		return
	}

	account, err := this.service.GetAccount(id)

	switch {
	case err == ERROR_NOT_FOUND:
		this.engine.RenderJsonError(resp, req, "error-account-not-found", http.StatusNotFound)
		return
	case err != nil:
		this.engine.RenderJsonError(resp, req, err.Error(), http.StatusInternalServerError)
		return
	}

	err = this.engine.Marshal(req, account, resp)
	if err != nil {
		this.engine.RenderJsonError(resp, req, "malformed-account", http.StatusInternalServerError)
		return
	}
}

func (this *EndPoint) ApiDeleteAccount(resp http.ResponseWriter, req *http.Request) {
	omni_http.SetCORSHeaders(resp)
	vars := mux.Vars(req)
	id := vars["id"]

	if id == "" {
		this.engine.RenderJsonError(resp, req, "error-missing-id", http.StatusBadRequest)
		return
	}

	err := this.service.DeleteAccount(id)

	switch {
	case err == ERROR_NOT_FOUND:
		this.engine.RenderJsonError(resp, req, "error-account-not-found", http.StatusNotFound)
		return
	case err != nil:
		this.engine.RenderJsonError(resp, req, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (this *EndPoint) findAccount(email, phone string) (account *Account, err error) {
	switch {
	case email != "":
		account, err = this.service.FindAccountByEmail(email)
	case phone != "":
		account, err = this.service.FindAccountByPhone(phone)
	case email == "" && phone == "":
		err = ERROR_MISSING_INPUT
	}
	return
}
