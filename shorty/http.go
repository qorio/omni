package shorty

import (
	"crypto/rand"
	"fmt"
	"github.com/golang/glog"
	"github.com/gorilla/mux"
	"github.com/qorio/omni/auth"
	omni_http "github.com/qorio/omni/http"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var secureCookie *omni_http.SecureCookie

var (
	contextQueryParam = "__xrlc"
	appUrlSchemeParam = "__xrlp"
	shortCodeParam    = "__xrls"
	noAppInstallParam = "__xrl_noapp"

	uuidCookieKey       = "uuid"
	lastViewedCookieKey = "last"

	regexFmt string = "[_A-Za-z0-9\\.\\-]{%d,}"
)

type ShortyEndPointSettings struct {
	Redirect404     string
	GeoIpDbFilePath string
}

type ShortyEndPoint struct {
	settings      ShortyEndPointSettings
	router        *mux.Router
	requestParser *omni_http.RequestParser
	service       Shorty
}

func init() {
	var err error
	secureCookie, err = omni_http.NewSecureCookie([]byte(""), nil)
	if err != nil {
		glog.Warningln("Cannot initialize secure cookie!")
		panic(err)
	}
}

func NewApiEndPoint(settings ShortyEndPointSettings, service Shorty) (api *ShortyEndPoint, err error) {
	if requestParser, err := omni_http.NewRequestParser(settings.GeoIpDbFilePath); err == nil {
		api = &ShortyEndPoint{
			settings:      settings,
			router:        mux.NewRouter(),
			requestParser: requestParser,
			service:       service,
		}
		regex := fmt.Sprintf(regexFmt, service.UrlLength())
		api.router.HandleFunc("/{id:"+regex+"}", api.RedirectHandler).Methods("GET").Name("redirect")

		// Campaign based
		api.router.HandleFunc("/api/v1/campaign", auth.RequiresAuth(api.ApiAddCampaignHandler)).
			Methods("POST").Name("add_campaign")
		api.router.HandleFunc("/api/v1/campaign/{campaignId}", auth.RequiresAuth(api.ApiGetCampaignHandler)).
			Methods("GET").Name("get_campaign")
		api.router.HandleFunc("/api/v1/campaign/{campaignId}", auth.RequiresAuth(api.ApiUpdateCampaignHandler)).
			Methods("POST").Name("update_campaign")
		api.router.HandleFunc("/api/v1/campaign/{campaignId}/url", auth.RequiresAuth(api.ApiAddCampaignUrlHandler)).
			Methods("POST").Name("add_campaign_url")

		// Stand-alone
		api.router.HandleFunc("/api/v1/url", auth.RequiresAuth(api.ApiAddUrlHandler)).
			Methods("POST").Name("add")

		// First attempt if the app starts up organically -- this gives the server an opportunity
		// to match by fingerprinting.  If fingerprinting cannot match a user then the response will tell
		// the SDK to use safari for first launch install reporting.
		api.router.HandleFunc("/api/v1/tryfp/{scheme}/{app_uuid}",
			api.ApiTryMatchInstallOnOrganicAppLaunch).Methods("POST").Name("app_install_try_fingerprint")

		api.router.HandleFunc("/api/v1/events/install/{scheme}/{app_uuid}",
			api.ApiReportInstallOnReferredAppLaunch).Methods("POST").Name("app_install_referred_launch")

		api.router.HandleFunc("/api/v1/events/openurl/{scheme}/{app_uuid}",
			api.ApiReportAppOpenUrl).Methods("POST").Name("app_ping")

		return api, nil
	} else {
		return nil, err
	}
}

func NewRedirector(settings ShortyEndPointSettings, service Shorty) (api *ShortyEndPoint, err error) {
	if requestParser, err := omni_http.NewRequestParser(settings.GeoIpDbFilePath); err == nil {
		api = &ShortyEndPoint{
			settings:      settings,
			router:        mux.NewRouter(),
			requestParser: requestParser,
			service:       service,
		}

		regex := fmt.Sprintf(regexFmt, service.UrlLength())
		api.router.HandleFunc("/{id:"+regex+"}", api.RedirectHandler).Methods("GET").Name("redirect")

		// Install reporting when no context is known (organic install, first launch)
		api.router.HandleFunc("/i/{scheme}/{app_uuid}",
			api.ReportInstallOnOrganicAppLaunch).Methods("GET").Name("app_install_on_direct_launch")

		api.router.HandleFunc("/m/{scheme}/{uuid}/{shortCode:"+regex+"}/",
			api.CheckAppInstallInterstitialHandler).Methods("GET").Name("app_install_interstitial")

		// Dynamically generated so that any html @fetchUrl loaded by above can reference it as src="../deeplink.js"
		api.router.HandleFunc("/m/{scheme}/{uuid}/{shortCode:"+regex+"}/deeplink.js",
			api.CheckAppInstallInterstitialJSHandler).Methods("GET").Name("app_install_interstitial_js")

		return api, nil
	} else {
		return nil, err
	}
}

func (this *ShortyEndPoint) ServeHTTP(resp http.ResponseWriter, request *http.Request) {
	this.router.ServeHTTP(resp, request)
}

func (this *ShortyEndPoint) RedirectHandler(resp http.ResponseWriter, req *http.Request) {
	req.ParseForm()
	vars := mux.Vars(req)
	shortUrl, err := this.service.FindUrl(vars["id"])

	if err != nil {
		renderError(resp, req, err.Error(), http.StatusInternalServerError)
		return
	} else if shortUrl == nil {
		if this.settings.Redirect404 != "" {
			originalUrl, err := this.router.Get("redirect").URL("id", vars["id"])
			if err != nil {
				renderError(resp, req, err.Error(), http.StatusInternalServerError)
				return
			}
			url404 := strings.Replace(this.settings.Redirect404,
				"$origURL", url.QueryEscape(fmt.Sprintf("http://%s%s", req.Host, originalUrl.String())), 1)
			http.Redirect(resp, req, url404, http.StatusTemporaryRedirect)
			return
		}
		renderError(resp, req, "No URL was found with that shorty code", http.StatusNotFound)
		return
	}

	var destination string = shortUrl.Destination
	var renderInline bool = false
	var appOpen *AppOpen
	var found bool

	omni_http.SetNoCachingHeaders(resp)
	cookies := omni_http.NewCookieHandler(secureCookie, resp, req)

	visits, cookied, last, userId := processCookies(cookies, shortUrl.Id)

	var matchedRule *RoutingRule
	matchedRuleId, deeplink := "no-match", "?"

	userAgent := omni_http.ParseUserAgent(req)
	origin, _ := this.requestParser.Parse(req)

	matchedRule, notFound := shortUrl.MatchRule(this.service, userAgent, origin, cookies)
	if notFound != nil || matchedRule == nil {
		http.Redirect(resp, req, shortUrl.Destination, http.StatusMovedPermanently)
		goto event
	}

	glog.Infoln("REDIRECT: matched rule:", matchedRule.Id)
	matchedRuleId = matchedRule.Id
	deeplink = matchedRule.Destination

	// check if there's been an appOpen
	appOpen, found, _ = this.service.FindAppOpen(UrlScheme(matchedRule.AppUrlScheme), UUID(userId))
	if !found || float64(time.Now().Unix()-appOpen.Timestamp) >= matchedRule.AppOpenTTLDays*24.*60.*60. {

		glog.V(10).Infoln("REDIRECT: no app-open in days:", matchedRule.AppOpenTTLDays)

		switch {

		case matchedRule.SendToInterstitial.isTrue():

			// Special handling of platforms where apps using webviews don't shared
			// cookies -- aka sandboxed uuids -- (e.g. iOS):
			// If the option to harvest the uuid cookie in this context is set to true
			// then do a redirect to a special harvest url where the uuid in the cookie
			// can be collected.  The harvest url will directly render content using the
			// rule's content properties e.g. rule.InlineContent or rule.FetchFromUrl.
			// This will let the calling browser think it's got the final content.
			// The user is then prompted to follow that harvest url in another browser
			// (eg. mobile Safari).  When the user uses another browser to go to that
			// url (which contains the uuid in this context), the handler of that url
			// will have a uuid-uuid pair.
			// For example, if on iOS, the Facebook app accesses the shortlink and arrives
			// here, we will have the uuid generated (userId).  We then redirect to a harvest
			// url //harvest/<shortCode>/<userId> and render a static html page instructing
			// the user to open the harvest url in Safari.  When the user opens the harvest
			// link in Safari, a different userId is generated (because the original userId
			// in the FB app webview is not shared).  Because the harvest url has the userId
			// in the FB app context, an association between the uuid-safari and uuid-fbapp
			// can be created.

			// If we need to harvest the cookied uuid - then
			// just redirect to the special landing page url where the cookied uuid (userId)
			// can be harvested.
			renderInline = false
			fetchUrl := url.QueryEscape(matchedRule.ContentSourceUrl)
			appUrlScheme := url.QueryEscape(matchedRule.AppUrlScheme)
			destination = fmt.Sprintf("/m/%s/%s/%s/?f=%s", appUrlScheme, userId, shortUrl.Id, fetchUrl)

		case matchedRule.ContentSourceUrl != "":

			renderInline = true
			destination = omni_http.FetchFromUrl(userAgent.Header, matchedRule.ContentSourceUrl)

		default:
			renderInline = false
			destination = matchedRule.AppStoreUrl
		}

	} else {

		glog.V(10).Infoln("REDIRECT- found app-open. redirecting", matchedRule.Destination)

		renderInline = false
		destination = matchedRule.Destination
	}

	if renderInline {
		resp.Write([]byte(destination))
	} else {
		destination = injectContext(destination, matchedRule, shortUrl, userId)
		http.Redirect(resp, req, destination, http.StatusMovedPermanently)
	}

event:
	// Record stats asynchronously
	timestamp := timestamp()

	go func() {
		origin, geoParseErr := this.requestParser.Parse(req)
		origin.Cookied = cookied
		origin.Visits = visits
		origin.LastVisit = last
		origin.Destination = destination
		origin.ShortCode = shortUrl.Id
		if geoParseErr != nil {
			glog.Warningln("can-not-determine-location", geoParseErr)
		}
		glog.Infoln(
			"uuid:", userId, "url:", shortUrl.Id, "send-to:", destination,
			"ip:", origin.Ip, "mobile:", origin.UserAgent.Mobile,
			"platform:", origin.UserAgent.Platform, "os:", origin.UserAgent.OS, "make:", origin.UserAgent.Make,
			"browser:", origin.UserAgent.Browser, "version:", origin.UserAgent.BrowserVersion,
			"location:", *origin.Location,
			"useragent:", origin.UserAgent.Header,
			"cookied", cookied)

		// Save a fingerprint
		fingerprint := omni_http.FingerPrint(origin)
		fv := &FingerprintedVisit{
			Fingerprint:   fingerprint,
			Context:       UUID(userId),
			ShortCode:     shortUrl.Id,
			Visit:         destination,
			Deeplink:      deeplink,
			Timestamp:     timestamp,
			Referrer:      origin.Referrer,
			MatchedRuleId: matchedRuleId,
			UserAgent:     origin.UserAgent.Header,
		}
		this.service.SaveFingerprintedVisit(fv)
		glog.V(20).Infoln("saved visit", "fingerprint=", fingerprint, "visit=", fv)

		this.service.PublishDecode(&DecodeEvent{
			RequestOrigin: origin,
			Destination:   destination,
			Context:       UUID(userId),
			Origin:        shortUrl.Origin,
			AppKey:        shortUrl.AppKey,
			CampaignKey:   shortUrl.CampaignKey,
			MatchedRuleId: matchedRuleId,
		})
		shortUrl.Record(origin, visits > 1)
	}()

	return
}

func injectContext(dest string, matchedRule *RoutingRule, shortUrl *ShortUrl, userId string) string {
	destination := dest
	if matchedRule != nil {
		parsedUrl, pErr := url.Parse(destination)
		// If the schemes match, then it's a deeplink.  Add additional params if the destination is
		// a deeplink or a http url that is mapped in Android's intent filter (where an app can also handle
		// url with http as scheme.
		if pErr == nil && (parsedUrl.Scheme == matchedRule.AppUrlScheme || matchedRule.IsAndroidIntentFilter.isTrue()) {
			destination = addQueryParam(destination, contextQueryParam, userId)
			destination = addQueryParam(destination, appUrlSchemeParam, matchedRule.AppUrlScheme)
			destination = addQueryParam(destination, shortCodeParam, shortUrl.Id)
		}
	}
	return destination
}

func addQueryParam(url, key, value string) string {
	if strings.ContainsRune(url, '?') {
		return url + "&" + key + "=" + value
	} else {
		return url + "?" + key + "=" + value
	}
}

func renderJsonError(resp http.ResponseWriter, req *http.Request, message string, code int) (err error) {
	resp.WriteHeader(code)
	resp.Write([]byte(fmt.Sprintf("{\"error\":\"%s\"}", message)))
	return
}

func renderError(resp http.ResponseWriter, req *http.Request, message string, code int) (err error) {
	resp.WriteHeader(code)
	resp.Write([]byte(fmt.Sprintf("<html><body>Error: %s </body></html>", message)))
	return
}

// newUUID generates a random UUID according to RFC 4122
func newUUID() (string, error) {
	uuid := make([]byte, 16)
	n, err := io.ReadFull(rand.Reader, uuid)
	if n != len(uuid) || err != nil {
		return "", err
	}
	// variant bits; see section 4.1.1
	uuid[8] = uuid[8]&^0xc0 | 0x80
	// version 4 (pseudo-random); see section 4.1.3
	uuid[6] = uuid[6]&^0xf0 | 0x40
	return fmt.Sprintf("%x-%x-%x-%x-%x", uuid[0:4], uuid[4:6], uuid[6:8], uuid[8:10], uuid[10:]), nil
}

// cookied = if the user uuid is cookied
func processCookies(cookies omni_http.Cookies, shortCode string) (visits int, cookied bool, last, uuid string) {
	last, _ = cookies.GetPlainString(lastViewedCookieKey)

	sc := shortCode
	if sc == "" {
		sc = last
	}
	cookies.Get(sc, &visits)

	var cookieError error

	uuid, _ = cookies.GetPlainString(uuidCookieKey)
	if uuid == "" {
		if uuid, _ = newUUID(); uuid != "" {
			cookieError = cookies.SetPlainString(uuidCookieKey, uuid)
			cookied = cookieError == nil
		}
	}

	visits++
	cookieError = cookies.SetPlainString(lastViewedCookieKey, sc)
	cookieError = cookies.Set(sc, visits)

	return
}
