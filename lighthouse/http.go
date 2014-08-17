package lighthouse

import (
	"github.com/golang/glog"
	api "github.com/qorio/api/lighthouse"
	omni_auth "github.com/qorio/omni/auth"
	omni_rest "github.com/qorio/omni/rest"
	"net/http"
)

type EndPoint struct {
	settings Settings
	service  Service
	engine   omni_rest.Engine
}

func defaultResolveApplicationId(req *http.Request) string {
	return req.URL.Host
}

func NewApiEndPoint(settings Settings, auth omni_auth.Service, service Service) (ep *EndPoint, err error) {
	ep = &EndPoint{
		settings: settings,
		service:  service,
		engine:   omni_rest.NewEngine(&api.Methods, auth, nil),
	}

	ep.engine.Bind(
		omni_rest.SetAuthenticatedHandler(api.Methods[api.AddOrUpdateBeacon], ep.ApiUpsertBeacon),
	)
	return ep, nil
}

func (this *EndPoint) ServeHTTP(resp http.ResponseWriter, request *http.Request) {
	this.engine.ServeHTTP(resp, request)
}

func (this *EndPoint) ApiUpsertBeacon(context omni_auth.Context, resp http.ResponseWriter, req *http.Request) {
	glog.Infoln("ApiUpsertBeacon")
}
