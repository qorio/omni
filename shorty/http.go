package shorty

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"github.com/golang/glog"
	"github.com/gorilla/mux"
	omni_http "github.com/qorio/omni/http"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var (
	contextQueryParam = "__xrlc"
	appUrlSchemeParam = "__xrlp"
	shortCodeParam    = "__xrls"

	uuidCookieKey       = "uuid"
	lastViewedCookieKey = "last"
)

type ShortyAddRequest struct {
	Vanity   string        `json:"vanity"`
	LongUrl  string        `json:"longUrl"`
	Rules    []RoutingRule `json:"rules"`
	Origin   string        `json:"origin"`
	ApiToken string        `json:"token"` // user facing token that resolves to appKey
	Campaign string        `json:"campaign"`
}

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

var secureCookie *omni_http.SecureCookie
var regexFmt string = "[_A-Za-z0-9\\.\\-]{%d,}"

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
		api.router.HandleFunc("/api/v1/url", api.ApiAddHandler).Methods("POST").Name("add")

		// First attempt if the app starts up organically -- this gives the server an opportunity
		// to match by fingerprinting.  If fingerprinting cannot match a user then the response will tell
		// the SDK to use safari for first launch install reporting.
		api.router.HandleFunc("/api/v1/tryfp/{scheme}/{app_uuid}",
			api.ApiTryMatchInstallOnOrganicAppLaunch).Methods("POST").Name("app_install_try_fingerprint")

		api.router.HandleFunc("/api/v1/events/install/{scheme}/{app_uuid}",
			api.ApiReportInstallOnReferredAppLaunch).Methods("POST").Name("app_install_referred_launch")

		api.router.HandleFunc("/api/v1/events/openurl/{scheme}/{app_uuid}",
			api.ApiReportAppOpenUrl).Methods("POST").Name("app_ping")

		api.router.HandleFunc("/api/v1/events/missing/{scheme}/{id:"+regex+"}",
			api.ReportDeviceUrlSchemeHandlerMissing).Name("app_missing")

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

		api.router.HandleFunc("/h/{shortUrlId:"+regex+"}/{uuid}",
			api.HarvestCookiedUUIDHandler).Methods("GET").Name("harvest")

		// Intermediary points where information about the current context of the short link gets collected.
		api.router.HandleFunc("/c/{scheme}/{shortCode:"+regex+"}/{uuid}/{fetchUrl}",
			api.CollectContextHandler).Methods("GET").Name("collect_context")

		// Dynamically generated so that any html @fetchUrl loaded by above can reference it as src="../deeplink.js"
		api.router.HandleFunc("/c/{scheme}/{shortCode:"+regex+"}/{uuid}/deeplink.js",
			api.DeeplinkJavaScriptHandler).Methods("GET").Name("collect_context")

		return api, nil
	} else {
		return nil, err
	}
}

func (this *ShortyEndPoint) ServeHTTP(resp http.ResponseWriter, request *http.Request) {
	this.router.ServeHTTP(resp, request)
}

func (this *ShortyEndPoint) ApiAddHandler(resp http.ResponseWriter, req *http.Request) {
	omni_http.SetCORSHeaders(resp)

	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		renderJsonError(resp, req, err.Error(), http.StatusInternalServerError)
		return
	}

	var message ShortyAddRequest
	dec := json.NewDecoder(strings.NewReader(string(body)))
	for {
		if err := dec.Decode(&message); err == io.EOF {
			break
		} else if err != nil {
			renderJsonError(resp, req, err.Error(), http.StatusBadRequest)
			return
		}
	}

	if message.LongUrl == "" {
		renderJsonError(resp, req, "No URL to shorten", http.StatusBadRequest)
		return
	}

	// Set the starting values, and the api will validate the rules and return a saved reference.
	shortUrl := &ShortUrl{
		Origin: message.Origin,

		// TODO - add lookup of api token to valid apiKey.
		// A api token is used by client as a way to authenticate and identify the actual app.
		// This way, we can revoke the token and shut down a client.
		AppKey: message.ApiToken,

		// TODO - this is a key that references a future struct that encapsulates all the
		// rules around default routing (appstore, etc.).  This will simplify the api by not
		// requiring ios client to send in rules on android, for example.  The service should
		// check to see if there's valid campaign for the same app key. If yes, then merge the
		// routing rules.  If not, just let this value be a tag of some kind.
		CampaignKey: message.Campaign,
	}
	if message.Vanity != "" {
		shortUrl, err = this.service.VanityUrl(message.Vanity, message.LongUrl, message.Rules, *shortUrl)
	} else {
		shortUrl, err = this.service.ShortUrl(message.LongUrl, message.Rules, *shortUrl)
	}

	if err != nil {
		renderJsonError(resp, req, err.Error(), http.StatusBadRequest)
		return
	}

	if _, err := this.router.Get("redirect").URL("id", shortUrl.Id); err != nil {
		renderJsonError(resp, req, err.Error(), http.StatusBadRequest)
		return
	}

	buff, err := json.Marshal(shortUrl)
	if err != nil {
		renderJsonError(resp, req, "Malformed short url rule", http.StatusInternalServerError)
		return
	}
	resp.Write(buff)
}

// cookied = if the user uuid is cookied
func processCookies(cookies omni_http.Cookies, shortCode string) (visits int, cookied bool, last, uuid string) {
	cookies.Get(lastViewedCookieKey, &last)

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
	cookieError = cookies.Set(lastViewedCookieKey, sc)
	cookieError = cookies.Set(sc, visits)

	return
}

func (this *ShortyEndPoint) RedirectHandler(resp http.ResponseWriter, req *http.Request) {
	req.ParseForm()
	vars := mux.Vars(req)
	shortUrl, err := this.service.Find(vars["id"])

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

	omni_http.SetNoCachingHeaders(resp)
	cookies := omni_http.NewCookieHandler(secureCookie, resp, req)

	visits, cookied, last, userId := processCookies(cookies, shortUrl.Id)

	var matchedRule *RoutingRule
	var matchedRuleIndex int = -1

	// If there are platform-dependent routing
	if len(shortUrl.Rules) > 0 {
		userAgent := omni_http.ParseUserAgent(req)
		origin, _ := this.requestParser.Parse(req)

		for i, rule := range shortUrl.Rules {
			if match := rule.Match(this.service, userAgent, origin, cookies); match {

				matchedRule = &rule
				matchedRuleIndex = i

				switch {

				case rule.HarvestCookiedUUID:

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
					fetchUrl := url.QueryEscape(rule.ContentSourceUrl)
					appUrlScheme := url.QueryEscape(rule.AppUrlScheme)
					destination = fmt.Sprintf("/h/%s/%s?c=%s&s=%s", shortUrl.Id, userId, fetchUrl, appUrlScheme)

				case rule.SendToInterstitial:

					// In this case, redirect to a special page that will collect the uuid, shortCode, etc.
					renderInline = false
					fetchUrl := url.QueryEscape(rule.ContentSourceUrl)
					appUrlScheme := url.QueryEscape(rule.AppUrlScheme)
					destination = fmt.Sprintf("/c/%s/%s/%s/%s", appUrlScheme, shortUrl.Id, userId, fetchUrl)

				case rule.ContentSourceUrl != "":
					renderInline = true
					destination = omni_http.FetchFromUrl(userAgent.Header, rule.ContentSourceUrl)

				case rule.AppUrlScheme != "":
					renderInline = false
					if !rule.NoAppStoreRedirect {
						destination = rule.AppStoreUrl
					} else {
						destination = rule.Destination
					}

				default:
					renderInline = false
					destination = rule.Destination
				}
				break
			} // if match
		} // foreach rule
	} // if there are rules

	// support for /shortCode?404= when app is missing
	if _, has := req.Form["404"]; has && matchedRule != nil {
		// here we get an event that the app is missing...
		count, _ := this.service.DeleteInstall(userId, matchedRule.AppUrlScheme)
		glog.Infoln("APP MISSING:", userId, matchedRule.AppUrlScheme, "found=", count)

		// do another redirect
		if next, err := this.router.Get("redirect").URL("id", shortUrl.Id); err == nil {
			http.Redirect(resp, req, next.String(), http.StatusMovedPermanently)
			return
		}
	}

	if renderInline {
		resp.Write([]byte(destination))
	} else {
		destination = injectContext(destination, matchedRule, shortUrl, userId)
		http.Redirect(resp, req, destination, http.StatusMovedPermanently)
	}

	// Record stats asynchronously
	timestamp := time.Now().Unix()

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
		this.service.SaveMatchableVisit(&MatchableVisit{
			Fingerprint: fingerprint,
			UUID:        userId,
			ShortCode:   shortUrl.Id,
			Deeplink:    destination,
			Timestamp:   timestamp,
			Referrer:    origin.Referrer,
		})

		this.service.PublishDecode(&DecodeEvent{
			RequestOrigin:    origin,
			Destination:      destination,
			ShortyUUID:       userId,
			Origin:           shortUrl.Origin,
			AppKey:           shortUrl.AppKey,
			CampaignKey:      shortUrl.CampaignKey,
			MatchedRuleIndex: matchedRuleIndex,
		})
		shortUrl.Record(origin, visits > 1)
	}()
}

func (this *ShortyEndPoint) HarvestCookiedUUIDHandler(resp http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	uuid := vars["uuid"]
	shortUrl, err := this.service.Find(vars["shortCode"])

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

	var content string = ""

	req.ParseForm()

	omni_http.SetNoCachingHeaders(resp)
	cookies := omni_http.NewCookieHandler(secureCookie, resp, req)

	// visits, cookied, last, userId := processCookies(cookies, shortUrl)
	visits, cookied, last, userId := processCookies(cookies, shortUrl.Id)

	userAgent := omni_http.ParseUserAgent(req)
	//origin, _ := this.requestParser.Parse(req)

	// Here we check if the two uuids are different.  One uuid is in the url of this request.  This is the uuid
	// from some context (e.g. from FB webview on iOS).  Another uuid is one in the cookie -- either we assigned
	// or read from the client context.  The current context may not be the same as the context of the uuid in
	// the url.  This is because the user could be visiting the same link from another browser (eg. on Safari)
	// after being prompted.
	// If the two uuids do not match -- then we know the contexts are different.  The user is visiting from
	// some context other than the one with the original link.  So in this case, we can do a redirect back to
	// the short link that the user was looking at that got them to see the harvest url in the first place.
	// Otherwise, show the static content which may tell them to try again in a different browser/context.

	appUrlScheme := ""

	if uuid != userId {

		// We got the user to come here via a different context (browser) than the one that created
		// this url in the first place.  So link the two ids together and redirect back to the short url.

		if appUrlSchemeParam, exists := req.Form["s"]; exists {
			appUrlScheme = appUrlSchemeParam[0]
			this.service.Link(uuid, userId, appUrlSchemeParam[0], shortUrl.Id)
			// Here we also assume that the user will install the app at some point.
			// Go ahead and assume that and let other mechanisms to invalidate this.
			this.service.TrackInstall(uuid, appUrlSchemeParam[0])
		}
		if next, err := this.router.Get("redirect").URL("id", shortUrl.Id); err != nil {
			renderJsonError(resp, req, err.Error(), http.StatusBadRequest)
			return
		} else {
			http.Redirect(resp, req, next.String(), http.StatusMovedPermanently)
			goto linkevent
		}
	} else {

		// The user is still coming from the same browser context because the two ids are the same.
		// In this case we just show the static content.

		// we expect the fetch url to be included in the 'c' parameter
		if fetchFromUrl, exists := req.Form["c"]; exists {
			content = omni_http.FetchFromUrl(userAgent.Header, fetchFromUrl[0])
		}
		resp.Write([]byte(content))

		return
	}

linkevent:

	go func() {
		origin, geoParseErr := this.requestParser.Parse(req)
		origin.Cookied = cookied
		origin.Visits = visits
		origin.LastVisit = last
		origin.Destination = content

		installOrigin, installAppKey, installCampaignKey := "NONE", appUrlScheme, "DIRECT"
		if shortUrl != nil {
			origin.ShortCode = shortUrl.Id
			installOrigin = shortUrl.Origin
			installAppKey = shortUrl.AppKey
			installCampaignKey = shortUrl.CampaignKey
		}

		if geoParseErr != nil {
			glog.Warningln("can-not-determine-location", geoParseErr)
		}
		glog.Infoln("LINK-UUID",
			"uuid:", userId, "url:", shortUrl.Id, "send-to:", content,
			"ip:", origin.Ip, "mobile:", origin.UserAgent.Mobile,
			"platform:", origin.UserAgent.Platform, "os:", origin.UserAgent.OS, "make:", origin.UserAgent.Make,
			"browser:", origin.UserAgent.Browser, "version:", origin.UserAgent.BrowserVersion,
			"location:", *origin.Location,
			"useragent:", origin.UserAgent.Header,
			"cookied", cookied)

		this.service.PublishLink(&LinkEvent{
			RequestOrigin: origin,
			ShortyUUID_A:  uuid,
			ShortyUUID_B:  userId,
			Origin:        installOrigin,
			AppKey:        installAppKey,
			CampaignKey:   installCampaignKey,
		})

	}()
}

func (this *ShortyEndPoint) CollectContextHandler(resp http.ResponseWriter, req *http.Request) {
	// Everything should be in the url
	vars := mux.Vars(req)
	uuid := vars["uuid"]
	shortCode := vars["shortCode"]
	appUrlScheme := vars["scheme"]
	fetchUrl := vars["fetchUrl"]

	omni_http.SetNoCachingHeaders(resp)
	cookies := omni_http.NewCookieHandler(secureCookie, resp, req)

	// visits, cookied, last, userId := processCookies(cookies, shortUrl)
	_, _, _, userId := processCookies(cookies, shortCode)
	userAgent := omni_http.ParseUserAgent(req)

	// Here we check if the two uuids are different.  One uuid is in the url of this request.  This is the uuid
	// from some context (e.g. from FB webview on iOS).  Another uuid is one in the cookie -- either we assigned
	// or read from the client context.  The current context may not be the same as the context of the uuid in
	// the url.  This is because the user could be visiting the same link from another browser (eg. on Safari)
	// after being prompted.
	// If the two uuids do not match -- then we know the contexts are different.  The user is visiting from
	// some context other than the one with the original link.  So in this case, we can do a redirect back to
	// the short link that the user was looking at that got them to see the harvest url in the first place.
	// Otherwise, show the static content which may tell them to try again in a different browser/context.

	if uuid != userId {

		this.service.Link(uuid, userId, appUrlScheme, shortCode)

		if next, err := this.router.Get("redirect").URL("id", shortCode); err != nil {
			renderJsonError(resp, req, err.Error(), http.StatusBadRequest)
		} else {
			http.Redirect(resp, req, next.String(), http.StatusMovedPermanently)
		}
	} else {

		// The user is still coming from the same browser context because the two ids are the same.
		// In this case we just show the static content.
		content := omni_http.FetchFromUrl(userAgent.Header, fetchUrl)
		resp.Write([]byte(content))
	}
	return
}

func (this *ShortyEndPoint) DeeplinkJavaScriptHandler(resp http.ResponseWriter, req *http.Request) {
	// Everything should be in the url
	vars := mux.Vars(req)
	uuid := vars["uuid"]
	shortCode := vars["shortCode"]
	appUrlScheme := vars["scheme"]
	fetchUrl := vars["fetchUrl"]

	omni_http.SetNoCachingHeaders(resp)
	cookies := omni_http.NewCookieHandler(secureCookie, resp, req)

	// visits, cookied, last, userId := processCookies(cookies, shortUrl)
	_, _, _, userId := processCookies(cookies, shortCode)
	userAgent := omni_http.ParseUserAgent(req)

	if uuid != userId {
		// Collect
		this.service.Link(uuid, userId, appUrlScheme, shortCode)
	}

	content := omni_http.FetchFromUrl(userAgent.Header, fetchUrl)
	resp.Write([]byte(content))
	return
}

func (this *ShortyEndPoint) ReportDeviceUrlSchemeHandlerMissing(resp http.ResponseWriter, req *http.Request) {
	req.ParseForm()
	vars := mux.Vars(req)
	//userAgent := omni_http.ParseUserAgent(req)
	origin, _ := this.requestParser.Parse(req)

	appUrlScheme := vars["scheme"]
	shortUrl, err := this.service.Find(vars["id"])
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

	omni_http.SetNoCachingHeaders(resp)
	cookies := omni_http.NewCookieHandler(secureCookie, resp, req)
	_, _, _, userId := processCookies(cookies, shortUrl.Id)

	glog.Infoln(">>>> APP MISSING", appUrlScheme, shortUrl.Id)
	// here we get an event that the app is missing...
	count, _ := this.service.DeleteInstall(userId, appUrlScheme)
	glog.Infoln("APP MISSING:", userId, appUrlScheme, "found=", count)

	// do another redirect
	if next, err := this.router.Get("redirect").URL("id", shortUrl.Id); err == nil {
		cookies.Set("Referer", origin.Referrer)
		http.Redirect(resp, req, next.String(), http.StatusMovedPermanently)
		return
	}
}

func (this *ShortyEndPoint) match(shortCode string, resp http.ResponseWriter, req *http.Request) (shortUrl *ShortUrl, match *RoutingRule, err error) {
	shortUrl, err = this.service.Find(shortCode)
	if err != nil {
		return
	}

	if shortUrl != nil && len(shortUrl.Rules) > 0 {
		userAgent := omni_http.ParseUserAgent(req)
		cookies := omni_http.NewCookieHandler(secureCookie, resp, req)
		origin, _ := this.requestParser.Parse(req)
		for _, rule := range shortUrl.Rules {
			if matched := rule.Match(this.service, userAgent, origin, cookies); matched {
				match = &rule
				return
			}
		}
	}
	return
}

func injectContext(dest string, matchedRule *RoutingRule, shortUrl *ShortUrl, userId string) string {
	destination := dest
	if matchedRule != nil {
		parsedUrl, pErr := url.Parse(destination)
		// If the schemes match, then it's a deeplink.  Add additional params if the destination is
		// a deeplink or a http url that is mapped in Android's intent filter (where an app can also handle
		// url with http as scheme.
		if pErr == nil && (parsedUrl.Scheme == matchedRule.AppUrlScheme || matchedRule.IsAndroidIntentFilter) {
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
	/*
		body, err := render(req, "layout", "error", map[string]string{"Error": message})
		if err != nil {
			http.Error(resp, err.Error(), http.StatusInternalServerError)
			return
		}

		resp.WriteHeader(code)
		resp.Write(body)
	*/
	return nil
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
