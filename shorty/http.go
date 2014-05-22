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
	"math"
	"net/http"
	"net/url"
	"strings"
	"time"
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
		api.router.HandleFunc("/api/v1/stats/{id:"+regex+"}", api.StatsHandler).Methods("GET").Name("stats")
		api.router.HandleFunc("/api/v1/events/install/{scheme}/{app_uuid}",
			api.ReportInstallHandler).Methods("GET").Name("app_install")
		api.router.HandleFunc("/api/v1/events/missing/{scheme}/{id:"+regex+"}",
			api.ReportDeviceUrlSchemeHandlerMissing).Name("app_missing")
		api.router.HandleFunc("/api/v1/events/ping/{scheme}/{id:"+regex+"}/{uuid}/{app_uuid}",
			api.ReportDeviceUrlSchemeHandlerPing).Methods("POST").Name("app_ping")

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
		api.router.HandleFunc("/h/{shortUrlId:"+regex+"}/{uuid}",
			api.HarvestCookiedUUIDHandler).Methods("GET").Name("harvest")

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
func processCookies(cookies omni_http.Cookies, shortUrl *ShortUrl) (visits int, cookied bool, last, uuid string) {
	cookies.Get("last", &last)
	cookies.Get(shortUrl.Id, &visits)

	var cookieError error

	uuid, _ = cookies.GetPlainString("uuid")
	if uuid == "" {
		if uuid, _ = newUUID(); uuid != "" {
			cookieError = cookies.SetPlainString("uuid", uuid)
			cookied = cookieError == nil
		}
	}

	visits++
	cookieError = cookies.Set("last", shortUrl.Id)
	cookieError = cookies.Set(shortUrl.Id, visits)

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

	visits, cookied, last, userId := processCookies(cookies, shortUrl)

	var rule RoutingRule
	// If there are platform-dependent routing
	if len(shortUrl.Rules) > 0 {
		userAgent := omni_http.ParseUserAgent(req)
		origin, _ := this.requestParser.Parse(req)

		for _, rule = range shortUrl.Rules {
			if match := rule.Match(this.service, userAgent, origin, cookies); match {

				destination = rule.Destination // default

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
				if rule.HarvestCookiedUUID {

					// If we need to harvest the cookied uuid - then
					// just redirect to the special landing page url where the cookied uuid (userId)
					// can be harvested.
					renderInline = false
					fetchUrl := url.QueryEscape(rule.FetchFromUrl)
					appUrlScheme := url.QueryEscape(rule.AppUrlScheme)
					destination = fmt.Sprintf("/h/%s/%s?c=%s&s=%s", shortUrl.Id, userId, fetchUrl, appUrlScheme)

				} else {

					switch {

					case rule.FetchFromUrl != "":
						renderInline = true
						destination = omni_http.FetchFromUrl(userAgent.Header, rule.FetchFromUrl)

					case rule.AppUrlScheme != "":
						renderInline = false
						_, found, _ := this.service.FindInstall(userId, rule.AppUrlScheme)
						if !found {
							destination = rule.AppStoreUrl
						}

					default:
						renderInline = false
						destination = rule.Destination
					}
				}
				break
			} // if match
		} // foreach rule
	} // if there are rules

	// support for /shortCode?404= when app is missing
	if _, has := req.Form["404"]; has {
		// here we get an event that the app is missing...
		count, _ := this.service.DeleteInstall(userId, rule.AppUrlScheme)
		glog.Infoln("APP MISSING:", userId, rule.AppUrlScheme, "found=", count)

		// do another redirect
		if next, err := this.router.Get("redirect").URL("id", shortUrl.Id); err == nil {
			http.Redirect(resp, req, next.String(), http.StatusMovedPermanently)
			return
		}
	}

	if renderInline {
		resp.Write([]byte(destination))
	} else {
		http.Redirect(resp, req, destination, http.StatusMovedPermanently)
	}

	// Record stats asynchronously
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

		this.service.PublishDecode(&DecodeEvent{
			RequestOrigin: origin,
			Destination:   destination,
			ShortyUUID:    userId,
			Origin:        shortUrl.Origin,
			AppKey:        shortUrl.AppKey,
			CampaignKey:   shortUrl.CampaignKey,
		})
		shortUrl.Record(origin, visits > 1)
	}()
}

func (this *ShortyEndPoint) HarvestCookiedUUIDHandler(resp http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	uuid := vars["uuid"]
	shortUrl, err := this.service.Find(vars["shortUrlId"])

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
	visits, cookied, last, userId := processCookies(cookies, shortUrl)

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
			this.service.TrackInstall(uuid, appUrlSchemeParam[0], shortUrl.InstallTTLSeconds)
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
	_, _, _, userId := processCookies(cookies, shortUrl)

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

func (this *ShortyEndPoint) ReportDeviceUrlSchemeHandlerPing(resp http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)

	appUrlScheme := vars["scheme"]
	appUuid := vars["app_uuid"]
	uuid := vars["uuid"]
	shortCode := vars["id"]

	glog.Infoln("App ping:", shortCode, appUrlScheme, appUuid, "uuid=", uuid)

	this.service.TrackInstall(uuid, appUrlScheme, 0) // TOOD - fix the TTL
}

func (this *ShortyEndPoint) ReportInstallHandler(resp http.ResponseWriter, req *http.Request) {
	omni_http.SetNoCachingHeaders(resp)

	vars := mux.Vars(req)
	cookies := omni_http.NewCookieHandler(secureCookie, resp, req)

	// Two parameters
	// 1. app custom url scheme -> this allows us to key by mobile app per user
	// 2. some uuid for the app -> this tracks a user. on ios, idfa uuid is used.

	appUrlScheme := vars["scheme"]
	if appUrlScheme == "" {
		renderError(resp, req, "No app customer url scheme", http.StatusBadRequest)
		return
	}

	appUuid := vars["app_uuid"]
	if appUuid == "" {
		renderError(resp, req, "No uuid", http.StatusBadRequest)
		return
	}

	// read the cookies that have been set before when user clicked a short link
	// this allows us to send a redirect as appropriate; otherwise, send a app url with 404

	// a unique user identifier -- generated by us and the lastViewed short code
	userId, lastViewed := "", ""

	cookies.Get("uuid", &userId)
	cookies.Get("last", &lastViewed)

	var shortUrl *ShortUrl
	var err error
	var expiration int64 = 0
	shortCode := ""

	if userId == "" && lastViewed == "" {
		// This is an organic install - install not driven by short link

		// TODO - Send a Thank you page, where onLoad, a deeplink to return to the app is triggered.
		// The url / content for the thank you page must be associated to a scheme and platform
		// but not tied to a particular shortlink.
	}

	var destination string = appUrlScheme + "://404"
	if lastViewed == "" {
		destination = addQueryParam(destination, "cookie", userId)
		http.Redirect(resp, req, destination, http.StatusMovedPermanently)
		goto stat
	} else {
		shortUrl, err = this.service.Find(lastViewed)
		if err != nil {
			renderError(resp, req, err.Error(), http.StatusInternalServerError)
			return
		} else if shortUrl == nil {
			destination = addQueryParam(destination, "cookie", userId)
			http.Redirect(resp, req, destination, http.StatusMovedPermanently)
			goto stat
		} else {
			shortCode = shortUrl.Id
			if shortUrl.InstallTTLSeconds > 0 {
				expiration = time.Now().Unix() + shortUrl.InstallTTLSeconds
			}
		}
	}

	// If there are platform-dependent routing
	if len(shortUrl.Rules) > 0 {
		userAgent := omni_http.ParseUserAgent(req)
		origin, _ := this.requestParser.Parse(req)
		for _, rule := range shortUrl.Rules {
			if match := rule.Match(this.service, userAgent, origin, cookies); match {
				destination = rule.Destination
				break
			}
		}
	}

	destination = addQueryParam(destination, "cookie", userId)
	http.Redirect(resp, req, destination, http.StatusMovedPermanently)

stat:
	this.service.Link(appUuid, userId, appUrlScheme, shortCode)
	this.service.TrackInstall(userId, appUrlScheme, expiration)

	go func() {
		origin, geoParseErr := this.requestParser.Parse(req)
		origin.Destination = destination

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
		glog.Infoln("send-to:", destination,
			"ip:", origin.Ip, "mobile:", origin.UserAgent.Mobile,
			"platform:", origin.UserAgent.Platform, "os:", origin.UserAgent.OS, "make:", origin.UserAgent.Make,
			"browser:", origin.UserAgent.Browser, "version:", origin.UserAgent.BrowserVersion,
			"location:", *origin.Location,
			"useragent:", origin.UserAgent.Header)

		this.service.PublishInstall(&InstallEvent{
			RequestOrigin: origin,
			Destination:   destination,
			AppUrlScheme:  appUrlScheme,
			AppUUID:       appUuid,
			ShortyUUID:    userId,
			Origin:        installOrigin,
			AppKey:        installAppKey,
			CampaignKey:   installCampaignKey,
		})
	}()
}

type StatsSummary struct {
	Id      string      `json:"id"`
	Created string      `json:"when"`
	Hits    int         `json:"hits"`
	Uniques int         `json:"uniques"`
	Summary OriginStats `json:"summary"`
	Config  ShortUrl    `json:"config"`
}

func (this *ShortyEndPoint) StatsHandler(resp http.ResponseWriter, req *http.Request) {
	omni_http.SetCORSHeaders(resp)

	vars := mux.Vars(req)
	shortyUrl, err := this.service.Find(vars["id"])
	if err != nil {
		renderError(resp, req, err.Error(), http.StatusInternalServerError)
		return
	} else if shortyUrl == nil {
		renderError(resp, req, "No URL was found with short code", http.StatusNotFound)
		return
	}

	hits, err := shortyUrl.Hits()
	if err != nil {
		renderError(resp, req, err.Error(), http.StatusInternalServerError)
		return
	}

	uniques, err := shortyUrl.Uniques()
	if err != nil {
		renderError(resp, req, err.Error(), http.StatusInternalServerError)
		return
	}

	originStats, err := shortyUrl.Sources(true)
	if err != nil {
		renderError(resp, req, err.Error(), http.StatusInternalServerError)
		return
	}

	summary := StatsSummary{
		Id:      shortyUrl.Id,
		Created: relativeTime(time.Now().Sub(shortyUrl.Created)),
		Hits:    hits,
		Uniques: uniques,
		Summary: originStats,
		Config:  *shortyUrl,
	}

	buff, err := json.Marshal(summary)
	if err != nil {
		renderJsonError(resp, req, "Malformed summary", http.StatusInternalServerError)
		return
	}
	resp.Write(buff)
}

func relativeTime(duration time.Duration) string {
	hours := int64(math.Abs(duration.Hours()))
	minutes := int64(math.Abs(duration.Minutes()))
	when := ""
	switch {
	case hours >= (365 * 24):
		when = "Over an year ago"
	case hours > (30 * 24):
		when = fmt.Sprintf("%d months ago", int64(hours/(30*24)))
	case hours == (30 * 24):
		when = "a month ago"
	case hours > 24:
		when = fmt.Sprintf("%d days ago", int64(hours/24))
	case hours == 24:
		when = "yesterday"
	case hours >= 2:
		when = fmt.Sprintf("%d hours ago", hours)
	case hours > 1:
		when = "over an hour ago"
	case hours == 1:
		when = "an hour ago"
	case minutes >= 2:
		when = fmt.Sprintf("%d minutes ago", minutes)
	case minutes > 1:
		when = "a minute ago"
	default:
		when = "just now"
	}
	return when
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
