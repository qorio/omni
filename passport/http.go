package passport

import (
	"code.google.com/p/goprotobuf/proto"
	"errors"
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
	oauth2   OAuth2Service
	engine   omni_rest.Engine
}

var ServiceId = "passport"

func NewApiEndPoint(settings Settings,
	auth omni_auth.Service,
	service Service,
	oauth2 OAuth2Service,
	webhooks omni_rest.WebHooksService) (ep *EndPoint, err error) {
	ep = &EndPoint{
		settings: settings,
		service:  service,
		oauth2:   oauth2,
		engine:   omni_rest.NewEngine(&api.Methods, auth, webhooks),
	}

	ep.engine.Bind(
		omni_rest.SetHandler(api.Methods[api.AuthUser], ep.ApiAuthenticate),
		omni_rest.SetHandler(api.Methods[api.AuthUserForService], ep.ApiAuthenticateForService),
	)

	ep.engine.Bind(
		omni_rest.SetAuthenticatedHandler(ServiceId, api.Methods[api.RegisterUser], ep.ApiRegisterUser),
	)

	ep.engine.Bind(
		omni_rest.SetAuthenticatedHandler(ServiceId, api.Methods[api.FetchAccount], ep.ApiGetAccount),
		omni_rest.SetAuthenticatedHandler(ServiceId, api.Methods[api.CreateOrUpdateAccount], ep.ApiSaveAccount),
		omni_rest.SetAuthenticatedHandler(ServiceId, api.Methods[api.UpdateAccountPrimaryLogin], ep.ApiSaveAccountPrimary),
		omni_rest.SetAuthenticatedHandler(ServiceId, api.Methods[api.AddOrUpdateAccountService], ep.ApiSaveAccountService),
		omni_rest.SetAuthenticatedHandler(ServiceId, api.Methods[api.AddOrUpdateServiceAttribute], ep.ApiSaveAccountServiceAttribute),
		omni_rest.SetAuthenticatedHandler(ServiceId, api.Methods[api.DeleteAccount], ep.ApiDeleteAccount),
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
	this.auth(resp, req, func(ep *EndPoint, authRequest *api.Identity) string {
		return authRequest.GetService()
	})
}

func (this *EndPoint) ApiAuthenticateForService(resp http.ResponseWriter, req *http.Request) {
	this.auth(resp, req, func(ep *EndPoint, authRequest *api.Identity) string {
		return ep.engine.GetUrlParameter(req, "service")
	})
}

func (this *EndPoint) auth(resp http.ResponseWriter, req *http.Request,
	get_service_friendly_name func(*EndPoint, *api.Identity) string) {

	request := api.Methods[api.AuthUser].RequestBody().(api.Identity)
	err := this.engine.Unmarshal(req, &request)
	if err != nil {
		this.engine.HandleError(resp, req, err.Error(), http.StatusBadRequest)
		return
	}

	// a quick check
	if request.Password == nil && request.Oauth2AccessToken == nil {
		this.engine.HandleError(resp, req, "error-no-credentials", http.StatusUnauthorized)
		return
	}

	// Check credentials
	account, err := this.findAccountByIdentity(&request)
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
	if account == nil {
		this.engine.HandleError(resp, req, "error-lookup-account", http.StatusInternalServerError)
		return
	}

	switch {
	case request.Password != nil:
		if !Password(request.Password).MatchAccount(account) {
			this.engine.HandleError(resp, req, "error-bad-credentials", http.StatusUnauthorized)
			return
		}

	case request.Oauth2AccessToken != nil:

	}

	// Now we have passed the check.
	serviceFriendlyName := get_service_friendly_name(this, &request)
	requestedServiceId := this.resolve_service_id(get_service_friendly_name(this, &request), req)

	matches := 0
	matchAll := false

	if serviceFriendlyName == "" {
		// If the friendly name of the service is "", this means we are retrieving all the services
		// and their auth scopes.
		matchAll = true
	}

	token := this.engine.NewAuthToken()
	for _, service := range account.GetServices() {
		if matchAll || service.GetId() == requestedServiceId {
			matches++
			prefix := service.GetId() + "/"
			func() {
				// encode the token
				token.Add(prefix+"@id", service.GetAccountId()).
					Add(prefix+"@status", service.GetStatus()).
					Add(prefix+"@scopes", strings.Join(service.GetScopes(), ","))

				for _, attribute := range service.GetAttributes() {
					if attribute.GetEmbedInToken() {
						switch attribute.GetType() {
						case api.Attribute_STRING:
							token.Add(prefix+attribute.GetKey(), attribute.GetStringValue())
						case api.Attribute_NUMBER:
							token.Add(prefix+attribute.GetKey(), attribute.GetNumberValue())
						case api.Attribute_BOOL:
							token.Add(prefix+attribute.GetKey(), attribute.GetBoolValue())
						case api.Attribute_BLOB:
							token.Add(prefix+attribute.GetKey(), attribute.GetBlobValue())
						}
					}
				}
			}()

			if !matchAll {
				break
			}
		}
	}

	if matches == 0 {
		this.engine.HandleError(resp, req, "error-not-a-member", http.StatusUnauthorized)
		return
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

	login := api.Methods[api.RegisterUser].RequestBody().(api.Identity)
	err := this.engine.Unmarshal(req, &login)
	if err != nil {
		this.engine.HandleError(resp, req, err.Error(), http.StatusBadRequest)
		return
	}

	account, err := this.findAccountByIdentity(&login)
	if err != nil && err != ERROR_NOT_FOUND {
		this.engine.HandleError(resp, req, err.Error(), http.StatusInternalServerError)
		return
	}
	if account != nil {
		this.engine.HandleError(resp, req, "error-duplicate", http.StatusConflict)
		return
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

	if account.Primary != nil {
		// Store the hmac instead
		Password(account.Primary.Password).Hash().Update()
	}

	err = this.service.SaveAccount(account)
	if err != nil {
		this.engine.HandleError(resp, req, err.Error(), http.StatusInternalServerError)
		return
	}

	sanitize(account)

	// Use the service id to determine the necessary callback / webhook after account record has been created.
	this.engine.EventChannel() <- &omni_rest.EngineEvent{
		Service:       requestedServiceId,
		ServiceMethod: api.RegisterUser,
		Body:          struct{ Account *api.Account }{account},
	}

	err = this.engine.Marshal(req, account.Primary, resp)
	if err != nil {
		this.engine.HandleError(resp, req, "malformed-account", http.StatusInternalServerError)
		return
	}
}

func (this *EndPoint) ApiSaveAccount(context omni_auth.Context, resp http.ResponseWriter, req *http.Request) {
	account := api.Methods[api.CreateOrUpdateAccount].RequestBody().(api.Account)
	err := this.engine.Unmarshal(req, &account)
	if err != nil {
		this.engine.HandleError(resp, req, err.Error(), http.StatusBadRequest)
		return
	}

	if account.Primary != nil {
		native := account.Primary
		if native.GetPhone() == "" && native.GetEmail() == "" && native.GetUsername() == "" {
			this.engine.HandleError(resp, req, ERROR_MISSING_INPUT.Error(), http.StatusBadRequest)
			return
		}
	}

	hasLoginId := account.Primary.Id != nil
	hasAccountId := account.Id != nil

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
		existing, err := this.findAccountByIdentity(account.GetPrimary())
		if err != nil && err != ERROR_NOT_FOUND {
			this.engine.HandleError(resp, req, "error-lookup", http.StatusInternalServerError)
			return
		}
		if existing != nil {
			this.engine.HandleError(resp, req, "error-conflict", http.StatusConflict)
			return
		}

		// Ok - assign login id
		uuid := omni_common.NewUUID().String()
		account.GetPrimary().Id = &uuid

	case !hasLoginId && !hasAccountId:
		existing, err := this.findAccountByIdentity(account.GetPrimary())
		if err != nil && err != ERROR_NOT_FOUND {
			this.engine.HandleError(resp, req, "error-lookup", http.StatusInternalServerError)
			return
		}
		if existing != nil {
			this.engine.HandleError(resp, req, "error-conflict", http.StatusConflict)
			return
		}

		// Ok - assign new login id
		uuid := omni_common.NewUUID().String()
		account.GetPrimary().Id = &uuid
		// Ok - assign new account id
		uuid = omni_common.NewUUID().String()
		account.Id = &uuid
	}

	if account.Id == nil {
		account.Id = proto.String(omni_common.NewUUID().String())
	}
	err = this.service.SaveAccount(&account)
	if err != nil {
		this.engine.HandleError(resp, req, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (this *EndPoint) ApiSaveAccountPrimary(context omni_auth.Context, resp http.ResponseWriter, req *http.Request) {
	id := this.engine.GetUrlParameter(req, "id")

	login := api.Methods[api.UpdateAccountPrimaryLogin].RequestBody().(api.Identity)
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

	err = validate_login(&login)
	if err != nil {
		this.engine.HandleError(resp, req, err.Error(), http.StatusBadRequest)
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

func (this *EndPoint) ApiSaveAccountService(context omni_auth.Context, resp http.ResponseWriter, req *http.Request) {
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

func (this *EndPoint) ApiSaveAccountServiceAttribute(context omni_auth.Context, resp http.ResponseWriter, req *http.Request) {
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

func (this *EndPoint) ApiGetAccount(context omni_auth.Context, resp http.ResponseWriter, req *http.Request) {
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

	err = this.engine.Marshal(req, sanitize(account), resp)
	if err != nil {
		this.engine.HandleError(resp, req, "malformed-account", http.StatusInternalServerError)
		return
	}
}

func (this *EndPoint) ApiDeleteAccount(context omni_auth.Context, resp http.ResponseWriter, req *http.Request) {
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

func (this *EndPoint) findAccountByIdentity(login *api.Identity) (account *api.Account, err error) {
	// If id is provided then just use that
	if login.Id != nil {
		account, err = this.service.GetAccount(omni_common.UUIDFromString(login.GetId()))
		return
	}

	// Otherwise, match by login username/email/phone or by oauth access token
	switch {
	case login.Password != nil:
		account, err = this.findAccount(login.GetEmail(), login.GetPhone(), login.GetUsername())
		return
	case login.Oauth2AccessToken != nil:
		// If a provider user account id is present, then use that to do a look up and see
		// if we can locate an account.  If not, call the provider's api to locate some user / profile
		// object or to get the actual user id.

		oauth2_account_id := login.GetOauth2AccountId()

		if login.Oauth2AccountId == nil {
			// Use the provider's api to debug/ validate the token and get a user id back.
			// We need to know the app id
			if login.Oauth2AppId == nil {
				return nil, errors.New("oauth2-app-id-missing")
			} else {
				v, err := this.oauth2.ValidateToken(login.GetOauth2Provider(), login.GetOauth2AppId(), login.GetOauth2AccessToken())
				if err != nil {
					return nil, err
				} else {
					// verify that the app id matches
					if v.AppId != "" && v.AppId != login.GetOauth2AppId() {
						return nil, errors.New("oauth2-app-id-mismatch")
					}
					oauth2_account_id = v.AccountId
				}
			}
		}
		account, err = this.service.FindAccountByOAuth2(login.GetOauth2Provider(), oauth2_account_id)
		return
	}
	return
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
