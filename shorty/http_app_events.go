package shorty

import (
	"encoding/json"
	"flag"
	"github.com/golang/glog"
	"github.com/gorilla/mux"
	omni_http "github.com/qorio/omni/http"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
	"time"
)

var (
	fingerPrintExpirationMinutes = flag.Int64("fingerprint_expiration_minutes", 2, "Minutes TTL matching by fingerprint")
	fingerPrintMinMatchingScore  = flag.Float64("fingerprint_min_score", 0.8, "Minimum score to match by fingerprint")
)

// /api/v1/events/try/{scheme}/{app_uuid}
// This handler is hit first when the app starts up organically.  So in this case, nothing is known,
// other than the scheme and app_uuid.  So we try to see if there's an existing decode record by the
// same ip address, postal code, region, and country within a specified timeframe.  If there is a match
// then this concludes the install reporting.  Otherwise, we tell the SDK to try again, this time by
// doing the switch through Safari via the public /i/scheme/app_uuid end point opening that URL instead.
func (this *ShortyEndPoint) ApiTryMatchInstallOnOrganicAppLaunch(resp http.ResponseWriter, req *http.Request) {
	omni_http.SetCORSHeaders(resp)

	vars := mux.Vars(req)

	app := vars["scheme"]
	if app == "" {
		renderError(resp, req, "No app customer url scheme", http.StatusBadRequest)
		return
	}

	appContext := vars["app_uuid"]
	if appContext == "" {
		renderError(resp, req, "No uuid", http.StatusBadRequest)
		return
	}

	// NO REQUEST BODY

	// That's all now we need to get the ip address etc.
	origin, _ := this.requestParser.Parse(req)
	fingerprint := omni_http.FingerPrint(origin)
	score, visit, _ := this.service.MatchFingerPrint(fingerprint)

	glog.Infoln("Matching fingerprint: score=", score, "visit=", visit)

	// TOOD - make the min score configurable
	// Also make sure the last visit was no more than 5 minutes ago
	if score > *fingerPrintMinMatchingScore && (time.Now().Unix()-visit.Timestamp) < *fingerPrintExpirationMinutes*60 {

		// Good enough - tell the SDK to go on.  No need to try to report conversion
		appOpen := &AppOpen{
			SourceContext:     visit.Context,
			SourceApplication: visit.Referrer,
			ShortCode:         visit.ShortCode,
			Deeplink:          visit.Deeplink,
		}
		if err := this.handleInstall(UrlScheme(app), UUID(appContext), appOpen, req, "fingerprint"); err != nil {
			renderJsonError(resp, req, err.Error(), http.StatusInternalServerError)
			return
		}
		if err := this.handleAppOpen(UrlScheme(app), UUID(appContext), appOpen, req); err != nil {
			renderJsonError(resp, req, err.Error(), http.StatusInternalServerError)
			return
		}

		buff, err := json.Marshal(visit)
		if err == nil {
			resp.Write(buff)
		} else {
			renderJsonError(resp, req, err.Error(), http.StatusInternalServerError)
			return
		}

	} else {
		// No good.  Tell SDK to try to use the switch method
		resp.WriteHeader(http.StatusNotAcceptable) // 406
	}

}

// /i/{scheme}/{app_uuid}
//
// This is the case where the app starts up without any context or un-referred (not called by
// another application via a deeplink.  So there are no context uuid or short code.  Instead
// the app reports install by performing a GET via a browser like Safari.
func (this *ShortyEndPoint) ReportInstallOnOrganicAppLaunch(resp http.ResponseWriter, req *http.Request) {
	omni_http.SetNoCachingHeaders(resp)

	vars := mux.Vars(req)
	cookies := omni_http.NewCookieHandler(secureCookie, resp, req)

	app := vars["scheme"]
	if app == "" {
		renderError(resp, req, "No app customer url scheme", http.StatusBadRequest)
		return
	}

	appContext := vars["app_uuid"]
	if appContext == "" {
		renderError(resp, req, "No uuid", http.StatusBadRequest)
		return
	}

	// read the cookies that have been set before when user clicked a short link
	// this allows us to send a redirect as appropriate; otherwise, send a app url with 404
	// a unique user identifier -- generated by us and the lastViewed short code
	lastViewed, userId := "", ""
	userId, _ = cookies.GetPlainString(uuidCookieKey)
	cookies.Get(lastViewedCookieKey, &lastViewed)

	// The lastViewed may not be the shortcode, but the interstitial
	sc := lastViewed
	parts := strings.Split(lastViewed, "/")
	if len(parts) >= 2 && parts[0] == "m" {
		sc = parts[1]
	}
	// Construct a AppOpen object using what is read from the http headers / cookies
	appOpen := &AppOpen{
		SourceContext:     UUID(userId),
		ShortCode:         sc,
		Deeplink:          ".", // The app opened itself without referrer
		SourceApplication: "ORGANIC",
	}

	this.handleInstall(UrlScheme(app), UUID(appContext), appOpen, req, "browser-switch")
	this.handleAppOpen(UrlScheme(app), UUID(appContext), appOpen, req)

	switch {

	case appOpen.ShortCode != "":
		http.Redirect(resp, req, "/"+appOpen.ShortCode, http.StatusMovedPermanently)
		return

	default:
		// Ok to add extra param -- this handler is called only from SDK.
		destination := app + "://404"
		destination = addQueryParam(destination, contextQueryParam, userId)
		http.Redirect(resp, req, destination, http.StatusMovedPermanently)
		return
	}
}

// /api/v1/events/install/{scheme}/{app_uuid}
// Similar to the handler above, this is invoked by SDK client via REST.  This is when the app is launched
// by a deeplink short url, such that the app's referring context are known.  The destination is sent back
// to the client as the api response and not by http redirect.
func (this *ShortyEndPoint) ApiReportInstallOnReferredAppLaunch(resp http.ResponseWriter, req *http.Request) {
	omni_http.SetCORSHeaders(resp)

	vars := mux.Vars(req)

	app := vars["scheme"]
	if app == "" {
		renderError(resp, req, "No app customer url scheme", http.StatusBadRequest)
		return
	}

	appContext := vars["app_uuid"]
	if appContext == "" {
		renderError(resp, req, "No uuid", http.StatusBadRequest)
		return
	}

	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		renderJsonError(resp, req, err.Error(), http.StatusInternalServerError)
		return
	}

	appOpen := &AppOpen{}
	dec := json.NewDecoder(strings.NewReader(string(body)))
	for {
		if err := dec.Decode(appOpen); err == io.EOF {
			break
		} else if err != nil {
			renderJsonError(resp, req, err.Error(), http.StatusBadRequest)
			return
		}
	}

	if err := this.handleInstall(UrlScheme(app), UUID(appContext), appOpen, req, "referred-app-open"); err != nil {
		renderJsonError(resp, req, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := this.handleAppOpen(UrlScheme(app), UUID(appContext), appOpen, req); err != nil {
		renderJsonError(resp, req, err.Error(), http.StatusInternalServerError)
		return
	}
}

// /api/v1/events/openurl/{scheme}/{app_uuid}
// The payload is a single AppOpen object
// The client POST this to the server when the app opens a deeplink url.
func (this *ShortyEndPoint) ApiReportAppOpenUrl(resp http.ResponseWriter, req *http.Request) {
	omni_http.SetCORSHeaders(resp)

	vars := mux.Vars(req)

	app := vars["scheme"]
	if app == "" {
		renderError(resp, req, "No app customer url scheme", http.StatusBadRequest)
		return
	}

	appContext := vars["app_uuid"]
	if appContext == "" {
		renderError(resp, req, "No uuid", http.StatusBadRequest)
		return
	}

	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		renderJsonError(resp, req, err.Error(), http.StatusInternalServerError)
		return
	}

	appOpen := &AppOpen{}
	dec := json.NewDecoder(strings.NewReader(string(body)))
	for {
		if err := dec.Decode(appOpen); err == io.EOF {
			break
		} else if err != nil {
			renderJsonError(resp, req, err.Error(), http.StatusBadRequest)
			return
		}
	}

	if err := this.handleAppOpen(UrlScheme(app), UUID(appContext), appOpen, req); err != nil {
		renderJsonError(resp, req, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (this *ShortyEndPoint) handleAppOpen(app UrlScheme, appContext UUID, appOpen *AppOpen, req *http.Request) error {
	this.service.Link(app, appOpen.SourceContext, appContext, appOpen.ShortCode)
	appOpen.Timestamp = timestamp()
	appOpen.AppContext = appContext
	this.service.TrackAppOpen(app, appContext, appOpen)

	shortUrl, err := this.service.FindUrl(appOpen.ShortCode)
	if err != nil {
		return err
	}

	if shortUrl == nil {
		// Problem - we can't do attribution
		glog.Warningln("cannot-determine-short-code")
	}

	go func() {

		origin, geoParseErr := this.requestParser.Parse(req)
		origin.Destination = appOpen.Deeplink

		installOrigin, installAppKey, installCampaignKey := "NONE", UUID(app), UUID("ORGANIC")

		if shortUrl != nil {
			origin.ShortCode = shortUrl.Id
			installOrigin = shortUrl.Origin
			installAppKey = shortUrl.AppKey
			installCampaignKey = shortUrl.CampaignKey
		}

		if geoParseErr != nil {
			glog.Warningln("can-not-determine-location", geoParseErr)
		}

		this.service.PublishAppOpen(&AppOpenEvent{
			RequestOrigin:     origin,
			Destination:       appOpen.Deeplink,
			App:               app,
			AppContext:        appContext,
			SourceContext:     appOpen.SourceContext,
			SourceApplication: appOpen.SourceApplication,
			Origin:            installOrigin,
			AppKey:            installAppKey,
			CampaignKey:       installCampaignKey,
		})

		if found, _ := this.service.FindLink(UUID(appContext), appOpen.SourceContext); !found {
			this.service.PublishLink(&LinkEvent{
				RequestOrigin: origin,
				Context1:      appContext,
				Context2:      appOpen.SourceContext,
				Origin:        installOrigin,
				AppKey:        installAppKey,
				CampaignKey:   installCampaignKey,
			})
		}
	}()
	return nil
}

func (this *ShortyEndPoint) handleInstall(app UrlScheme, appContext UUID, appOpen *AppOpen, req *http.Request, reportingMethod string) error {

	if appOpen.SourceContext != "" {
		this.service.Link(app, appOpen.SourceContext, appContext, appOpen.ShortCode)
	}

	this.service.TrackInstall(app, appContext)

	shortUrl, err := this.service.FindUrl(appOpen.ShortCode)
	if err != nil {
		return err
	}

	if shortUrl == nil {
		// Problem - we can't do attribution
		glog.Warningln("cannot-determine-short-code", appOpen)
	}

	go func() {

		origin, geoParseErr := this.requestParser.Parse(req)
		origin.Destination = appOpen.Deeplink

		installOrigin, installAppKey, installCampaignKey := "NONE", UUID(app), UUID("ORGANIC")

		if shortUrl != nil {
			origin.ShortCode = shortUrl.Id
			installOrigin = shortUrl.Origin
			installAppKey = shortUrl.AppKey
			installCampaignKey = shortUrl.CampaignKey
		}

		if geoParseErr != nil {
			glog.Warningln("can-not-determine-location", geoParseErr)
		}

		this.service.PublishInstall(&InstallEvent{
			RequestOrigin:     origin,
			Destination:       appOpen.Deeplink,
			App:               app,
			AppContext:        appContext,
			SourceContext:     appOpen.SourceContext,
			SourceApplication: appOpen.SourceApplication,
			Origin:            installOrigin,
			AppKey:            installAppKey,
			CampaignKey:       installCampaignKey,
			ReportingMethod:   reportingMethod,
		})

		if appOpen.SourceContext != "" {
			if found, _ := this.service.FindLink(appContext, appOpen.SourceContext); !found {
				this.service.PublishLink(&LinkEvent{
					RequestOrigin: origin,
					Context1:      appContext,
					Context2:      appOpen.SourceContext,
					Origin:        installOrigin,
					AppKey:        installAppKey,
					CampaignKey:   installCampaignKey,
				})
			}
		}

	}()
	return nil
}
