package passport

import (
	"fmt"
	"github.com/golang/glog"
	api "github.com/qorio/api/passport"
	omni_auth "github.com/qorio/omni/auth"
	omni_common "github.com/qorio/omni/common"
	omni_rest "github.com/qorio/omni/rest"
	"math"
	"net/http"
	"strings"
	"time"
)

type EndPoint struct {
	settings Settings
	service  Service
	engine   omni_rest.Engine
	encrypt  omni_auth.EncryptionService
}

func NewApiEndPoint(settings Settings, auth omni_auth.Service, service Service,
	webhooks omni_rest.WebHooksService,
	encrypt omni_auth.EncryptionService) (ep *EndPoint, err error) {
	ep = &EndPoint{
		settings: settings,
		service:  service,
		engine:   omni_rest.NewEngine(&api.Methods, auth, webhooks),
		encrypt:  encrypt,
	}

	ep.engine.Bind(
		omni_rest.SetHandler(api.Methods[api.AuthUser], ep.ApiAuthenticate),
		omni_rest.SetHandler(api.Methods[api.AuthUserForService], ep.ApiAuthenticateForService),
	)

	ep.engine.Bind(
		omni_rest.SetAuthenticatedHandler(api.Methods[api.RegisterUser], ep.ApiRegisterUser),
	)

	ep.engine.Bind(
		omni_rest.SetHandler(api.Methods[api.FetchAccount], ep.ApiGetAccount),
		omni_rest.SetHandler(api.Methods[api.CreateOrUpdateAccount], ep.ApiSaveAccount),
		omni_rest.SetHandler(api.Methods[api.UpdateAccountPrimaryLogin], ep.ApiSaveAccountPrimary),
		omni_rest.SetHandler(api.Methods[api.AddOrUpdateAccountService], ep.ApiSaveAccountService),
		omni_rest.SetHandler(api.Methods[api.AddOrUpdateServiceAttribute], ep.ApiSaveAccountServiceAttribute),
		omni_rest.SetHandler(api.Methods[api.DeleteAccount], ep.ApiDeleteAccount),
	)

	return ep, nil
}

func (this *EndPoint) ServeHTTP(resp http.ResponseWriter, request *http.Request) {
	this.engine.ServeHTTP(resp, request)
}

func defaultResolveServiceId(req *http.Request) string {
	return req.URL.Host
}

func (this *EndPoint) resolve_service_id(requestedServiceId string, req *http.Request) string {
	if requestedServiceId == "" {
		if this.settings.ResolveServiceId != nil {
			return this.settings.ResolveServiceId(req)
		} else {
			return defaultResolveServiceId(req)
		}
	}
	return requestedServiceId
}

// Authenticates and returns a token as the response
func (this *EndPoint) ApiAuthenticate(resp http.ResponseWriter, req *http.Request) {
	this.auth(resp, req, func(ep *EndPoint, authRequest *api.AuthRequest) string {
		return authRequest.GetService()
	})
}

func (this *EndPoint) ApiAuthenticateForService(resp http.ResponseWriter, req *http.Request) {
	this.auth(resp, req, func(ep *EndPoint, authRequest *api.AuthRequest) string {
		return ep.engine.GetUrlParameter(req, "service")
	})
}

func (this *EndPoint) auth(resp http.ResponseWriter, req *http.Request, get_service func(*EndPoint, *api.AuthRequest) string) {
	request := api.Methods[api.AuthUser].RequestBody().(api.AuthRequest)
	err := this.engine.Unmarshal(req, &request)
	if err != nil {
		this.engine.HandleError(resp, req, err.Error(), http.StatusBadRequest)
		return
	}

	// Lookup account
	account, err := this.findAccount(request.GetEmail(), request.GetPhone(), request.GetUsername())

	switch {
	case err == ERROR_NOT_FOUND:
		this.engine.HandleError(resp, req, "error-account-not-found", http.StatusUnauthorized)
		return
	case err == ERROR_MISSING_INPUT:
		this.engine.HandleError(resp, req, err.Error(), http.StatusBadRequest)
		return

	case err != nil:
		this.engine.HandleError(resp, req, "error-lookup-account", http.StatusInternalServerError)
		return
	}

	// Check credentials
	encrypted, err := this.encrypt.Encrypt([]byte(request.GetPassword()))
	if err == nil {
		encryptedStr := fmt.Sprintf("%0x", encrypted)
		request.Password = &encryptedStr
	}

	if account.Primary.GetPassword() != request.GetPassword() {
		this.engine.HandleError(resp, req, "error-bad-credentials", http.StatusUnauthorized)
		return
	}

	requestedServiceId := this.resolve_service_id(get_service(this, &request), req)
	var service *api.Service
	for _, test := range account.GetServices() {
		if test.GetId() == requestedServiceId {
			service = test
			break
		}
	}

	if service == nil {
		this.engine.HandleError(resp, req, "error-not-a-member", http.StatusUnauthorized)
		return
	}

	// encode the token
	token := this.engine.NewAuthToken()
	token.Add("@id", service.GetId()).
		Add("@status", service.GetStatus()).
		Add("@accountId", service.GetAccountId()).
		Add("@permissions", strings.Join(service.GetPermissions(), ","))

	for _, attribute := range service.GetAttributes() {
		if attribute.GetEmbedSigninToken() {
			switch attribute.GetType() {
			case api.Attribute_STRING:
				token.Add(attribute.GetKey(), attribute.GetStringValue())
			case api.Attribute_NUMBER:
				token.Add(attribute.GetKey(), attribute.GetNumberValue())
			case api.Attribute_BOOL:
				token.Add(attribute.GetKey(), attribute.GetBoolValue())
			case api.Attribute_BLOB:
				token.Add(attribute.GetKey(), attribute.GetBlobValue())
			}
		}
	}
	tokenString, err := this.engine.SignedString(token)
	if err != nil {
		glog.Warningln("error-generating-auth-token", err)
		this.engine.HandleError(resp, req, "cannot-generate-auth-token", http.StatusInternalServerError)
		return
	}

	// Response
	authResponse := api.Methods[api.AuthUser].ResponseBody().(api.AuthResponse)
	authResponse.Token = &tokenString

	err = this.engine.Marshal(req, &authResponse, resp)
	if err != nil {
		this.engine.HandleError(resp, req, "malformed-response", http.StatusInternalServerError)
		return
	}
}

func (this *EndPoint) ApiRegisterUser(context omni_auth.Context, resp http.ResponseWriter, req *http.Request) {
	requestedServiceId := this.resolve_service_id(this.engine.GetUrlParameter(req, "service"), req)
	if requestedServiceId == "" {
		this.engine.HandleError(resp, req, "cannot-determine-service", http.StatusBadRequest)
		return
	}

	login := api.Methods[api.RegisterUser].RequestBody().(api.Login)
	err := this.engine.Unmarshal(req, &login)
	if err != nil {
		this.engine.HandleError(resp, req, err.Error(), http.StatusBadRequest)
		return
	}

	// Check to see if login already exists
	account, err := this.findAccount(login.GetEmail(), login.GetPhone(), login.GetUsername())
	if err != nil {
		this.engine.HandleError(resp, req, err.Error(), http.StatusInternalServerError)
		return
	}
	if account != nil {
		this.engine.HandleError(resp, req, "error-duplicate", http.StatusConflict)
		return
	}

	// Encrypt the password
	encrypted, err := this.encrypt.Encrypt([]byte(login.GetPassword()))
	if err == nil {
		encryptedStr := fmt.Sprintf("%0x", encrypted)
		login.Password = &encryptedStr
	}

	// Create the entire Account object
	uuid := omni_common.NewUUID().String()
	login.Id = &uuid

	ts := float64(time.Now().UnixNano()) / math.Pow10(9)
	account = &api.Account{
		Id:               &uuid,
		Primary:          &login,
		CreatedTimestamp: &ts,
	}
	err = this.service.SaveAccount(account)
	if err != nil {
		this.engine.HandleError(resp, req, err.Error(), http.StatusInternalServerError)
		return
	}

	// Clear the password before we send it on to other systems
	account.Primary.Password = nil

	// Use the service id to determine the necessary callback / webhook after account record has been created.
	this.engine.EventChannel() <- &omni_rest.EngineEvent{
		ServiceMethod: api.RegisterUser,
		Body:          struct{ Account *api.Account }{account},
	}

}

func (this *EndPoint) ApiSaveAccount(resp http.ResponseWriter, req *http.Request) {
	account := api.Methods[api.CreateOrUpdateAccount].RequestBody().(api.Account)
	err := this.engine.Unmarshal(req, &account)
	if err != nil {
		this.engine.HandleError(resp, req, err.Error(), http.StatusBadRequest)
		return
	}

	if account.GetPrimary().GetPhone() == "" && account.GetPrimary().GetEmail() == "" {
		this.engine.HandleError(resp, req, ERROR_MISSING_INPUT.Error(), http.StatusBadRequest)
		return
	}

	hasLoginId := account.GetPrimary().GetId() != ""
	hasAccountId := account.GetId() != ""

	switch {
	case hasLoginId && hasAccountId:
		// simple update case

	case hasLoginId && !hasAccountId:
		// not allowed -- should not start a new one.
		this.engine.HandleError(resp, req, "cannot-transfer-login-to-new-account",
			http.StatusBadRequest)
		return

	case !hasLoginId && hasAccountId:
		// this is changing primary login of the account
		// check availability of phone/email
		existing, _ := this.findAccount(account.GetPrimary().GetEmail(),
			account.GetPrimary().GetPhone(), account.GetPrimary().GetUsername())
		if existing != nil {
			this.engine.HandleError(resp, req, "error-duplicate", http.StatusConflict)
			return
		}

		// Ok - assign login id
		uuid := omni_common.NewUUID().String()
		account.GetPrimary().Id = &uuid

	case !hasLoginId && !hasAccountId:
		// this is new login and account
		// check availability of phone/email
		existing, _ := this.findAccount(account.GetPrimary().GetEmail(),
			account.GetPrimary().GetPhone(), account.GetPrimary().GetUsername())
		if existing != nil {
			this.engine.HandleError(resp, req, "error-duplicate", http.StatusConflict)
			return
		}

		// Ok - assign new login id
		uuid := omni_common.NewUUID().String()
		account.GetPrimary().Id = &uuid
		// Ok - assign new account id
		uuid = omni_common.NewUUID().String()
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

		uuid := omni_common.NewUUID().String()
		account.Id = &uuid
	}
	err = this.service.SaveAccount(&account)
	if err != nil {
		this.engine.HandleError(resp, req, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (this *EndPoint) ApiSaveAccountPrimary(resp http.ResponseWriter, req *http.Request) {
	id := this.engine.GetUrlParameter(req, "id")

	login := api.Methods[api.UpdateAccountPrimaryLogin].RequestBody().(api.Login)
	err := this.engine.Unmarshal(req, &login)
	if err != nil {
		this.engine.HandleError(resp, req, err.Error(), http.StatusBadRequest)
		return
	}

	if id == "" {
		this.engine.HandleError(resp, req, "error-missing-id", http.StatusBadRequest)
		return
	}

	account, err := this.service.GetAccount(omni_common.UUIDFromString(id))

	switch {
	case err == ERROR_NOT_FOUND:
		this.engine.HandleError(resp, req, "error-account-not-found", http.StatusNotFound)
		return
	case err != nil:
		this.engine.HandleError(resp, req, err.Error(), http.StatusInternalServerError)
		return
	}

	switch {
	case login.GetPhone() == "" && login.GetEmail() == "":
		this.engine.HandleError(resp, req, "error-missing-email-or-phone", http.StatusBadRequest)
		return
	case login.GetPassword() == "":
		this.engine.HandleError(resp, req, "error-missing-password", http.StatusBadRequest)
		return
	}

	if login.GetId() == "" {
		uuid := omni_common.NewUUID().String()
		login.Id = &uuid
	}

	// update the primary
	account.Primary = &login

	err = this.service.SaveAccount(account)
	if err != nil {
		this.engine.HandleError(resp, req, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (this *EndPoint) ApiSaveAccountService(resp http.ResponseWriter, req *http.Request) {
	id := this.engine.GetUrlParameter(req, "id")

	service := api.Methods[api.AddOrUpdateAccountService].RequestBody().(api.Service)
	err := this.engine.Unmarshal(req, &service)
	if err != nil {
		this.engine.HandleError(resp, req, err.Error(), http.StatusBadRequest)
		return
	}

	if id == "" {
		this.engine.HandleError(resp, req, "error-missing-id", http.StatusBadRequest)
		return
	}

	account, err := this.service.GetAccount(omni_common.UUIDFromString(id))

	switch {
	case err == ERROR_NOT_FOUND:
		this.engine.HandleError(resp, req, "error-account-not-found", http.StatusNotFound)
		return
	case err != nil:
		this.engine.HandleError(resp, req, err.Error(), http.StatusInternalServerError)
		return
	}

	// find the service by id and replace it
	if len(account.GetServices()) == 0 {
		account.Services = []*api.Service{
			&service,
		}
	} else {
		match := false
		for i, app := range account.GetServices() {
			if app.GetId() == service.GetId() {
				account.Services[i] = &service
				match = true
				break
			}
		}
		if !match {
			account.Services = append(account.Services, &service)
		}
	}

	err = this.service.SaveAccount(account)
	if err != nil {
		this.engine.HandleError(resp, req, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (this *EndPoint) ApiSaveAccountServiceAttribute(resp http.ResponseWriter, req *http.Request) {
	id := this.engine.GetUrlParameter(req, "id")
	serviceId := this.engine.GetUrlParameter(req, "service")

	attribute := api.Methods[api.AddOrUpdateServiceAttribute].RequestBody().(api.Attribute)
	err := this.engine.Unmarshal(req, &attribute)
	if err != nil {
		this.engine.HandleError(resp, req, err.Error(), http.StatusBadRequest)
		return
	}

	if id == "" {
		this.engine.HandleError(resp, req, "error-missing-id", http.StatusBadRequest)
		return
	}

	account, err := this.service.GetAccount(omni_common.UUIDFromString(id))

	switch {
	case err == ERROR_NOT_FOUND:
		this.engine.HandleError(resp, req, "error-account-not-found", http.StatusNotFound)
		return
	case err != nil:
		this.engine.HandleError(resp, req, err.Error(), http.StatusInternalServerError)
		return
	}

	// find the service by id and update its attributes
	var service *api.Service
	for _, app := range account.GetServices() {
		if app.GetId() == serviceId {
			service = app
			break
		}
	}

	if service == nil {
		this.engine.HandleError(resp, req, "error-service-id-not-found", http.StatusBadRequest)
		return
	}

	if len(service.GetAttributes()) == 0 {
		service.Attributes = []*api.Attribute{
			&attribute,
		}
	} else {
		match := false
		for i, attr := range service.GetAttributes() {
			if attr.GetKey() == attribute.GetKey() {
				service.Attributes[i] = &attribute
				match = true
				break
			}
		}
		if !match {
			service.Attributes = append(service.Attributes, &attribute)
		}
	}

	err = this.service.SaveAccount(account)
	if err != nil {
		this.engine.HandleError(resp, req, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (this *EndPoint) ApiGetAccount(resp http.ResponseWriter, req *http.Request) {
	id := this.engine.GetUrlParameter(req, "id")

	if id == "" {
		this.engine.HandleError(resp, req, "error-missing-id", http.StatusBadRequest)
		return
	}

	account, err := this.service.GetAccount(omni_common.UUIDFromString(id))

	switch {
	case err == ERROR_NOT_FOUND:
		this.engine.HandleError(resp, req, "error-account-not-found", http.StatusNotFound)
		return
	case err != nil:
		this.engine.HandleError(resp, req, err.Error(), http.StatusInternalServerError)
		return
	}

	err = this.engine.Marshal(req, account, resp)
	if err != nil {
		this.engine.HandleError(resp, req, "malformed-account", http.StatusInternalServerError)
		return
	}
}

func (this *EndPoint) ApiDeleteAccount(resp http.ResponseWriter, req *http.Request) {
	id := this.engine.GetUrlParameter(req, "id")

	if id == "" {
		this.engine.HandleError(resp, req, "error-missing-id", http.StatusBadRequest)
		return
	}

	err := this.service.DeleteAccount(omni_common.UUIDFromString(id))

	switch {
	case err == ERROR_NOT_FOUND:
		this.engine.HandleError(resp, req, "error-account-not-found", http.StatusNotFound)
		return
	case err != nil:
		this.engine.HandleError(resp, req, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (this *EndPoint) findAccount(email, phone, username string) (account *api.Account, err error) {
	switch {
	case email != "":
		account, err = this.service.FindAccountByEmail(email)
	case phone != "":
		account, err = this.service.FindAccountByPhone(phone)
	case username != "":
		account, err = this.service.FindAccountByUsername(phone)
	case email == "" && phone == "" && username == "":
		err = ERROR_MISSING_INPUT
	}
	return
}
