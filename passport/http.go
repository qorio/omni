package passport

import (
	"code.google.com/p/goprotobuf/proto"
	"encoding/json"
	"fmt"
	"github.com/golang/glog"
	"github.com/gorilla/mux"
	omni_auth "github.com/qorio/omni/auth"
	omni_common "github.com/qorio/omni/common"
	omni_http "github.com/qorio/omni/http"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
)

type EndPoint struct {
	settings Settings
	router   *mux.Router
	auth     *omni_auth.Service
	service  Service
}

func defaultResolveApplicationId(req *http.Request) string {
	return req.URL.Host
}

func NewApiEndPoint(settings Settings, auth *omni_auth.Service, service Service) (api *EndPoint, err error) {
	api = &EndPoint{
		settings: settings,
		router:   mux.NewRouter(),
		auth:     auth,
		service:  service,
	}

	// Authentication endpoint
	api.router.HandleFunc("/api/v1/auth", api.ApiAuthenticate).
		Methods("POST").Name("auth")

	// Account management endpoints
	api.router.HandleFunc("/api/v1/account", api.ApiSaveAccount).
		Methods("POST").Name("account-save")
	api.router.HandleFunc("/api/v1/account/{id}/primary", api.ApiSaveAccountPrimary).
		Methods("POST").Name("account-login-update")
	api.router.HandleFunc("/api/v1/account/{id}/services", api.ApiSaveAccountService).
		Methods("POST").Name("account-services-update")
	api.router.HandleFunc("/api/v1/account/{id}/service/{applicationId}/attributes", api.ApiSaveAccountServiceAttribute).
		Methods("POST").Name("account-services-attribute-update")
	api.router.HandleFunc("/api/v1/account/{id}", api.ApiGetAccount).
		Methods("GET").Name("account-get")
	api.router.HandleFunc("/api/v1/account/{id}", api.ApiDeleteAccount).
		Methods("DELETE").Name("account-delete")

	return api, nil
}

func (this *EndPoint) ServeHTTP(resp http.ResponseWriter, request *http.Request) {
	this.router.ServeHTTP(resp, request)
}

// Authenticates and returns a token as the response
func (this *EndPoint) ApiAuthenticate(resp http.ResponseWriter, req *http.Request) {
	omni_http.SetCORSHeaders(resp)

	request := &AuthRequest{}
	err := unmarshal(req.Header.Get("Content-Type"), req.Body, request)
	if err != nil {
		renderJsonError(resp, req, err.Error(), http.StatusBadRequest)
		return
	}

	// do the lookup here...
	account, err := this.findAccount(request.GetEmail(), request.GetPhone())

	switch {
	case err == ERROR_NOT_FOUND:
		renderJsonError(resp, req, "error-account-not-found", http.StatusUnauthorized)
		return
	case err == ERROR_MISSING_INPUT:
		renderJsonError(resp, req, err.Error(), http.StatusBadRequest)
		return

	case err != nil:
		renderJsonError(resp, req, "error-lookup-account", http.StatusInternalServerError)
		return
	case err == nil && account.Primary.GetPassword() != request.GetPassword():
		renderJsonError(resp, req, "error-bad-credentials", http.StatusUnauthorized)
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
		renderJsonError(resp, req, "error-not-a-member", http.StatusUnauthorized)
		return
	}

	// encode the token
	token := this.auth.NewToken()
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
	tokenString, err := this.auth.SignedString(token)
	if err != nil {
		glog.Warningln("error-generating-auth-token", err)
		renderJsonError(resp, req, "cannot-generate-auth-token", http.StatusInternalServerError)
		return
	}

	// Response
	authResponse := &AuthResponse{
		Token: &tokenString,
	}

	buff, err := marshal(req.Header.Get("Content-Type"), authResponse)
	if err != nil {
		renderJsonError(resp, req, "malformed-response", http.StatusInternalServerError)
		return
	}
	resp.Write(buff)
}

func (this *EndPoint) ApiSaveAccount(resp http.ResponseWriter, req *http.Request) {
	omni_http.SetCORSHeaders(resp)

	account := &Account{}
	err := unmarshal(req.Header.Get("Content-Type"), req.Body, account)
	if err != nil {
		renderJsonError(resp, req, err.Error(), http.StatusBadRequest)
		return
	}

	if account.GetPrimary().GetPhone() == "" && account.GetPrimary().GetEmail() == "" {
		renderJsonError(resp, req, ERROR_MISSING_INPUT.Error(), http.StatusBadRequest)
		return
	}

	hasLoginId := account.GetPrimary().GetId() != ""
	hasAccountId := account.GetId() != ""

	switch {
	case hasLoginId && hasAccountId:
		// simple update case

	case hasLoginId && !hasAccountId:
		// not allowed -- should not start a new one.
		renderJsonError(resp, req, "cannot-transfer-login-to-new-account",
			http.StatusBadRequest)
		return

	case !hasLoginId && hasAccountId:
		// this is changing primary login of the account
		// check availability of phone/email
		existing, _ := this.findAccount(account.GetPrimary().GetEmail(),
			account.GetPrimary().GetPhone())
		if existing != nil {
			renderJsonError(resp, req, "error-duplicate", http.StatusConflict)
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
			renderJsonError(resp, req, "error-duplicate", http.StatusConflict)
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
	err = this.service.SaveAccount(account)
	if err != nil {
		renderJsonError(resp, req, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (this *EndPoint) ApiSaveAccountPrimary(resp http.ResponseWriter, req *http.Request) {
	omni_http.SetCORSHeaders(resp)
	vars := mux.Vars(req)
	id := vars["id"]

	login := &Login{}
	err := unmarshal(req.Header.Get("Content-Type"), req.Body, login)
	if err != nil {
		renderJsonError(resp, req, err.Error(), http.StatusBadRequest)
		return
	}

	if id == "" {
		renderJsonError(resp, req, "error-missing-id", http.StatusBadRequest)
		return
	}

	account, err := this.service.GetAccount(id)

	switch {
	case err == ERROR_NOT_FOUND:
		renderJsonError(resp, req, "error-account-not-found", http.StatusNotFound)
		return
	case err != nil:
		renderJsonError(resp, req, err.Error(), http.StatusInternalServerError)
		return
	}

	switch {
	case login.GetPhone() == "" && login.GetEmail() == "":
		renderJsonError(resp, req, "error-missing-email-or-phone", http.StatusBadRequest)
		return
	case login.GetPassword() == "":
		renderJsonError(resp, req, "error-missing-password", http.StatusBadRequest)
		return
	}

	if login.GetId() == "" {
		uuid, _ := omni_common.NewUUID()
		login.Id = &uuid
	}

	// update the primary
	account.Primary = login

	err = this.service.SaveAccount(account)
	if err != nil {
		renderJsonError(resp, req, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (this *EndPoint) ApiSaveAccountService(resp http.ResponseWriter, req *http.Request) {
	omni_http.SetCORSHeaders(resp)
	vars := mux.Vars(req)
	id := vars["id"]

	application := &Application{}
	err := unmarshal(req.Header.Get("Content-Type"), req.Body, application)
	if err != nil {
		renderJsonError(resp, req, err.Error(), http.StatusBadRequest)
		return
	}

	if id == "" {
		renderJsonError(resp, req, "error-missing-id", http.StatusBadRequest)
		return
	}

	account, err := this.service.GetAccount(id)

	switch {
	case err == ERROR_NOT_FOUND:
		renderJsonError(resp, req, "error-account-not-found", http.StatusNotFound)
		return
	case err != nil:
		renderJsonError(resp, req, err.Error(), http.StatusInternalServerError)
		return
	}

	// find the application by id and replace it
	if len(account.GetServices()) == 0 {
		account.Services = []*Application{
			application,
		}
	} else {
		match := false
		for i, app := range account.GetServices() {
			if app.GetId() == application.GetId() {
				account.Services[i] = application
				match = true
				break
			}
		}
		if !match {
			account.Services = append(account.Services, application)
		}
	}

	err = this.service.SaveAccount(account)
	if err != nil {
		renderJsonError(resp, req, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (this *EndPoint) ApiSaveAccountServiceAttribute(resp http.ResponseWriter, req *http.Request) {
	omni_http.SetCORSHeaders(resp)
	vars := mux.Vars(req)
	id := vars["id"]
	applicationId := vars["applicationId"]

	attribute := &Attribute{}
	err := unmarshal(req.Header.Get("Content-Type"), req.Body, attribute)
	if err != nil {
		renderJsonError(resp, req, err.Error(), http.StatusBadRequest)
		return
	}

	if id == "" {
		renderJsonError(resp, req, "error-missing-id", http.StatusBadRequest)
		return
	}

	account, err := this.service.GetAccount(id)

	switch {
	case err == ERROR_NOT_FOUND:
		renderJsonError(resp, req, "error-account-not-found", http.StatusNotFound)
		return
	case err != nil:
		renderJsonError(resp, req, err.Error(), http.StatusInternalServerError)
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
		renderJsonError(resp, req, "error-application-id-not-found", http.StatusBadRequest)
		return
	}

	if len(application.GetAttributes()) == 0 {
		application.Attributes = []*Attribute{
			attribute,
		}
	} else {
		match := false
		for i, attr := range application.GetAttributes() {
			if attr.GetKey() == attribute.GetKey() {
				application.Attributes[i] = attribute
				match = true
				break
			}
		}
		if !match {
			application.Attributes = append(application.Attributes, attribute)
		}
	}

	err = this.service.SaveAccount(account)
	if err != nil {
		renderJsonError(resp, req, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (this *EndPoint) ApiGetAccount(resp http.ResponseWriter, req *http.Request) {
	omni_http.SetCORSHeaders(resp)
	vars := mux.Vars(req)
	id := vars["id"]

	if id == "" {
		renderJsonError(resp, req, "error-missing-id", http.StatusBadRequest)
		return
	}

	account, err := this.service.GetAccount(id)

	switch {
	case err == ERROR_NOT_FOUND:
		renderJsonError(resp, req, "error-account-not-found", http.StatusNotFound)
		return
	case err != nil:
		renderJsonError(resp, req, err.Error(), http.StatusInternalServerError)
		return
	}

	buff, err := marshal(req.Header.Get("Content-Type"), account)
	if err != nil {
		renderJsonError(resp, req, "malformed-account", http.StatusInternalServerError)
		return
	}
	resp.Write(buff)
}

func (this *EndPoint) ApiDeleteAccount(resp http.ResponseWriter, req *http.Request) {
	omni_http.SetCORSHeaders(resp)
	vars := mux.Vars(req)
	id := vars["id"]

	if id == "" {
		renderJsonError(resp, req, "error-missing-id", http.StatusBadRequest)
		return
	}

	err := this.service.DeleteAccount(id)

	switch {
	case err == ERROR_NOT_FOUND:
		renderJsonError(resp, req, "error-account-not-found", http.StatusNotFound)
		return
	case err != nil:
		renderJsonError(resp, req, err.Error(), http.StatusInternalServerError)
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

func unmarshal(contentType string, body io.ReadCloser, typed proto.Message) (err error) {
	switch {
	case contentType == "application/json" || contentType == "":
		dec := json.NewDecoder(body)
		return dec.Decode(typed)
	case contentType == "application/protobuf":
		buff, err := ioutil.ReadAll(body)
		if err != nil {
			return err
		}
		return proto.Unmarshal(buff, typed)
	default:
		return ERROR_UNKNOWN_CONTENT_TYPE
	}
}

func marshal(contentType string, typed proto.Message) (buff []byte, err error) {
	switch {
	case contentType == "application/json" || contentType == "":
		buff, err = json.Marshal(typed)
	case contentType == "application/protobuf":
		buff, err = proto.Marshal(typed)
	default:
		err = ERROR_UNKNOWN_CONTENT_TYPE
	}
	return
}

func renderJsonError(resp http.ResponseWriter, req *http.Request, message string, code int) (err error) {
	resp.WriteHeader(code)
	resp.Write([]byte(fmt.Sprintf("{\"error\":\"%s\"}", message)))
	return
}
