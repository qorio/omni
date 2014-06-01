package shorty

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/golang/glog"
	"github.com/gorilla/mux"
	omni_http "github.com/qorio/omni/http"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"text/template"
	"time"
)

var secureCookie *omni_http.SecureCookie

var (
	contextQueryParam = "__xrlc"
	appUrlSchemeParam = "__xrlp"
	shortCodeParam    = "__xrls"

	uuidCookieKey       = "uuid"
	lastViewedCookieKey = "last"

	regexFmt string = "[_A-Za-z0-9\\.\\-]{%d,}"

	deeplinkJsTemplate   = template.New("deeplink.js")
	openTestHtmlTemplate = template.New("opentest.html")
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

	deeplinkJsTemplate, err = deeplinkJsTemplate.Parse(`
function getCookie(name) {
    var value = "; " + document.cookie;
    var parts = value.split("; " + name + "=");
    if (parts.length == 2) return parts.pop().split(";").shift();
}
function onLoad() {
    var shortUrl = window.location;
    var deeplink = "{{.Destination}}";
    var scheme = deeplink.split("://").shift();
    var shortCode = shortUrl.pathname.substring(1);
    deeplink += "&__xrlc=" + getCookie("uuid") + "&__xrlp=" + scheme + "&__xrls=" + shortCode;
    setTimeout(function() {
        if (!document.webkitHidden) {
            var el = document.getElementById("has-app")
            el.innerHTML = "<h1>Still here?  Try open this in Safari to install the app.</h1>";
	}
    }, 2000);
    window.location = deeplink;
}
`)
	if err != nil {
		glog.Warningln("Bad template for deeplink.js!")
		panic(err)
	}

	openTestHtmlTemplate, err = openTestHtmlTemplate.Parse(`
<html>
 <head>
  <title>Getting content...</title>
  <script type="text/javascript" src="./deeplink.js"></script>
 </head>
 <body onload="onLoad()">
  <div id="has-app"></div>
  <!-- Markdown text -->
  <xmp theme="journal" style="display:none;">
  Opening the link in app...  If the app does not open, open this link via Safari to install the app.
  </xmp>
 </body>
 <script src="http://strapdownjs.com/v/0.2/strapdown.js"></script>
</html>
`)
	if err != nil {
		glog.Warningln("Bad template for html test/open!")
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

		api.router.HandleFunc("/m/{shortCode:"+regex+"}/{uuid}/",
			api.CheckAppInstallInterstitialHandler).Methods("GET").Name("app_install_interstitial")

		// Dynamically generated so that any html @fetchUrl loaded by above can reference it as src="../deeplink.js"
		api.router.HandleFunc("/m/{shortCode:"+regex+"}/{uuid}/deeplink.js",
			api.CheckAppInstallInterstitialJSHandler).Methods("GET").Name("app_install_interstitial_js")

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

	userAgent := omni_http.ParseUserAgent(req)
	origin, _ := this.requestParser.Parse(req)

	// If there are platform-dependent routing
	if len(shortUrl.Rules) > 0 {
		for i, rule := range shortUrl.Rules {
			if match := rule.Match(this.service, userAgent, origin, cookies); match {

				matchedRule = &rule
				matchedRuleIndex = i

				// next level
				if len(rule.Special) > 0 {
					// The subRule has been preprocessed to be the merge of
					// the parent and the overrides
					for _, subRule := range rule.Special {
						if matchSub := subRule.Match(this.service, userAgent, origin, cookies); matchSub {
							matchedRule = &subRule
							break
						}
					}
				}
				break
			} // if match
		} // foreach rule
	} // if there are rules

	// Rule selected, now decide what to do.
	if matchedRule != nil {
		switch {

		case matchedRule.CheckAppInstallViaInterstitial:

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
			destination = fmt.Sprintf("/m/%s/%s/?f=%s&s=%s", shortUrl.Id, userId, fetchUrl, appUrlScheme)

		case matchedRule.ContentSourceUrl != "":
			renderInline = true
			destination = omni_http.FetchFromUrl(userAgent.Header, matchedRule.ContentSourceUrl)

			// TODO - Review this... why is this even necessary?
		case matchedRule.AppUrlScheme != "":
			renderInline = false
			if !matchedRule.NoAppStoreRedirect {
				destination = matchedRule.AppStoreUrl
			} else {
				destination = matchedRule.Destination
			}

		default:
			renderInline = false
			destination = matchedRule.Destination
		}
	}

	if renderInline {
		resp.Write([]byte(destination))
	} else {
		destination = injectContext(destination, matchedRule, shortUrl, userId)
		http.Redirect(resp, req, destination, http.StatusMovedPermanently)
	}

	deeplink := ""
	if matchedRule != nil {
		deeplink = matchedRule.Destination
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
		this.service.SaveFingerprintedVisit(&FingerprintedVisit{
			Fingerprint: fingerprint,
			Context:     UUID(userId),
			ShortCode:   shortUrl.Id,
			Visit:       destination,
			Deeplink:    deeplink,
			Timestamp:   timestamp,
			Referrer:    origin.Referrer,
		})

		this.service.PublishDecode(&DecodeEvent{
			RequestOrigin:    origin,
			Destination:      destination,
			Context:          UUID(userId),
			Origin:           shortUrl.Origin,
			AppKey:           shortUrl.AppKey,
			CampaignKey:      shortUrl.CampaignKey,
			MatchedRuleIndex: matchedRuleIndex,
		})
		shortUrl.Record(origin, visits > 1)
	}()
}

func (this *ShortUrl) MatchRule(service Shorty, userAgent *omni_http.UserAgent,
	origin *omni_http.RequestOrigin, cookies omni_http.Cookies) (matchedRule *RoutingRule, err error) {

	for _, rule := range this.Rules {
		if match := rule.Match(this.service, userAgent, origin, cookies); match {
			matchedRule = &rule
			break
		}
	}
	if matchedRule == nil || matchedRule.Destination == "" {
		err = errors.New("not found")
	}
	return
}

func (this *ShortyEndPoint) CheckAppInstallInterstitialHandler(resp http.ResponseWriter, req *http.Request) {
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
	_, _, lastViewed, userId := processCookies(cookies, shortUrl.Id)
	glog.Infoln(">>> harvest - processed cookies", lastViewed, userId, shortUrl.Id)

	userAgent := omni_http.ParseUserAgent(req)
	origin, _ := this.requestParser.Parse(req)

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

		glog.Infoln(">>>> harvest phase")

		// We got the user to come here via a different context (browser) than the one that created
		// this url in the first place.  So link the two ids together and redirect back to the short url.

		if appUrlSchemeParam, exists := req.Form["s"]; exists {

			appUrlScheme = appUrlSchemeParam[0]
			this.service.Link(UrlScheme(appUrlScheme), UUID(uuid), UUID(userId), shortUrl.Id)

			// Now, look for an app-open in context of userId.  If we have somehow opened the app
			// before, then we can just create an app-open entry for *this* context (uuid) because
			// we know that the app already exists on the device and was opened in a different context.

			appOpen, found, _ := this.service.FindAppOpen(UrlScheme(appUrlScheme), UUID(userId))

			glog.Infoln("find app-open", appUrlScheme, userId, appOpen, found)

			if !found {

				// no app-open even in another context.... so just send it to the appstore
				matchedRule, _ := shortUrl.MatchRule(this.service, userAgent, origin, cookies)
				if matchedRule != nil {
					http.Redirect(resp, req, matchedRule.AppStoreUrl, http.StatusMovedPermanently)
					return
				}

			} else {

				// create a record *as if* the app was also opened in the other context
				appOpen.SourceContext = UUID(uuid)
				appOpen.SourceApplication = origin.Referrer
				this.service.TrackAppOpen(UrlScheme(appUrlScheme), appOpen.AppContext, appOpen)
			}
		}
		if next, err := this.router.Get("redirect").URL("id", shortUrl.Id); err == nil {

			http.Redirect(resp, req, next.String(), http.StatusMovedPermanently)
			return

		}
	} else {

		// The user is still coming from the same browser context because the two ids are the same.
		// In this case we just show the static content.

		// we expect the fetch url to be included in the 'f' parameter
		if fetchFromUrl, exists := req.Form["f"]; exists {

			content = omni_http.FetchFromUrl(userAgent.Header, fetchFromUrl[0])
			resp.Write([]byte(content))

		} else {
			// Some dummy content
			openTestHtmlTemplate.Execute(resp, "")
		}
	}

	return
}

func (this *ShortyEndPoint) CheckAppInstallInterstitialJSHandler(resp http.ResponseWriter, req *http.Request) {
	omni_http.SetNoCachingHeaders(resp)

	vars := mux.Vars(req)
	shortCode := vars["shortCode"]

	shortUrl, err := this.service.Find(shortCode)
	if err != nil || shortUrl == nil {
		renderError(resp, req, err.Error(), http.StatusInternalServerError)
		return
	}

	cookies := omni_http.NewCookieHandler(secureCookie, resp, req)
	userAgent := omni_http.ParseUserAgent(req)
	origin, _ := this.requestParser.Parse(req)

	matchedRule, notFound := shortUrl.MatchRule(this.service, userAgent, origin, cookies)
	if notFound != nil {
		renderError(resp, req, "not found", http.StatusNotFound)
		return
	}
	deeplinkJsTemplate.Execute(resp, matchedRule)
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
