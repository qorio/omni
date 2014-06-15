package shorty

import (
	"bytes"
	"fmt"
	"github.com/golang/glog"
	"github.com/gorilla/mux"
	omni_http "github.com/qorio/omni/http"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"text/template"
	"time"
)

var (
	deeplinkJsTemplate   = template.New("deeplink.js")
	openTestHtmlTemplate = template.New("opentest.html")
)

type appInstallInterstitialContext struct {
	Rule                  *RoutingRule
	IsCrossBrowserContext bool
	Timestamp             int64
}

func init() {
	var err error

	deeplinkJsTemplate, err = deeplinkJsTemplate.Parse(`
function getCookie(name) {
    var value = "; " + document.cookie;
    var parts = value.split("; " + name + "=");
    if (parts.length == 2) return parts.pop().split(";").shift();
}
function getParameterByName(name) {
    name = name.replace(/[\[]/, "\\[").replace(/[\]]/, "\\]");
    var regex = new RegExp("[\\?&]" + name + "=([^&#]*)"),
        results = regex.exec(location.search);
    return results == null ? null : decodeURIComponent(results[1].replace(/\+/g, " "));
}
function redirectWithLocation(target) {
    navigator.geolocation.getCurrentPosition(function(position) {
        lat = position.coords.latitude
        lng = position.coords.longitude
        window.location = target + "&lat=" + lat + "&lng=" + lng
    })
}
function onLoad() {
    var deeplink = "{{.Rule.Destination}}";
    var appstore = "{{.Rule.AppStoreUrl}}";
    var interstitialUrl = window.location;
    var didNotDetectApp = getParameterByName('__xrl_noapp') != null;
    if (didNotDetectApp) {
{{if .IsCrossBrowserContext }}
        window.location = appstore;
{{else}}
        var el = document.getElementById("has-app")
        el.innerHTML = "<h1>Still here?  Try open this in Safari.</h1>";
{{end}}

    } else {
        var scheme = deeplink.split("://").shift();
        var shortCode = window.location.pathname.substring(1);
        deeplink += "&__xrlc=" + getCookie("uuid") + "&__xrlp=" + scheme + "&__xrls=" + shortCode;
        setTimeout(function() {
{{if eq .Rule.InterstitialToAppStoreOnTimeout "on"}}
              if (!document.webkitHidden) {
                  setTimeout(function(){
                      redirectWithLocation(interstitialUrl + "&__xrl_noapp=");
                  }, 2000)
                  window.location = appstore;
              }
{{else}}
              if (!document.webkitHidden) {
                  redirectWithLocation(interstitialUrl + "&__xrl_noapp=");
              }
{{end}}
        }, {{.Rule.InterstitialAppLinkTimeoutMillis}});
        window.location = deeplink;
    }
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
  <script type="text/javascript" src="./deeplink.js?{{.Timestamp}}"></script>
 </head>
 <body onload="onLoad()">
  <div id="has-app"></div>
  <xmp theme="journal" style="display:none;">

{{if .IsCrossBrowserContext }}
  Install the app <a href="{{.Rule.AppStoreUrl}}">here.</a>
{{else}}
  Opening the link in app...  If the app does not open, open this link via Safari.
{{end}}

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

func (this *ShortyEndPoint) CheckAppInstallInterstitialHandler(resp http.ResponseWriter, req *http.Request) {
	req.ParseForm()
	_, noapp := req.Form[noAppInstallParam]

	vars := mux.Vars(req)
	uuid := vars["uuid"]
	appUrlScheme := vars["scheme"]

	shortUrl, err := this.service.FindUrl(vars["shortCode"])

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

	omni_http.SetNoCachingHeaders(resp)
	cookies := omni_http.NewCookieHandler(secureCookie, resp, req)

	userAgent := omni_http.ParseUserAgent(req)
	origin, _ := this.requestParser.Parse(req)

	matchedRule, _ := shortUrl.MatchRule(this.service, userAgent, origin, cookies)

	// visits, cookied, last, userId := processCookies(cookies, shortUrl)
	_, _, lastViewed, userId := processCookies(cookies, shortUrl.Id)
	glog.Infoln(">>> harvest - processed cookies", lastViewed, userId, shortUrl.Id, "matchedRule=", matchedRule.Id, matchedRule.Comment)

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

		glog.Infoln(">>>> harvest phase, noapp=", noapp)

		// We got the user to come here via a different context (browser) than the one that created
		// this url in the first place.  So link the two ids together and redirect back to the short url.

		this.service.Link(UrlScheme(appUrlScheme), UUID(uuid), UUID(userId), shortUrl.Id)

		// Now, look for an app-open in context of userId.  If we have somehow opened the app
		// before, then we can just create an app-open entry for *this* context (uuid) because
		// we know that the app already exists on the device and was opened in a different context.

		appOpen, found, _ := this.service.FindAppOpen(UrlScheme(appUrlScheme), UUID(userId))

		glog.Infoln("find app-open", appUrlScheme, userId, appOpen, found)

		if found {
			// create a record *as if* the app was also opened in the other context
			appOpen.SourceContext = UUID(uuid)
			appOpen.SourceApplication = origin.Referrer
			this.service.TrackAppOpen(UrlScheme(appUrlScheme), appOpen.AppContext, appOpen)
		}
	}

	// save a fingerprint
	go func() {
		// check and see if we have params for location
		if lat, exists := req.Form["lat"]; exists {
			if lng, exists := req.Form["lng"]; exists {
				if latitude, err := strconv.ParseFloat(lat[0], 64); err == nil {
					if longitude, err := strconv.ParseFloat(lng[0], 64); err == nil {
						origin.Location.Latitude = latitude
						origin.Location.Longitude = longitude
					}
				}
			}
		}

		fingerprint := omni_http.FingerPrint(origin)
		glog.Infoln(">> New fingerprint: ", fingerprint)

		this.service.SaveFingerprintedVisit(&FingerprintedVisit{
			Fingerprint: fingerprint,
			Context:     UUID(userId),
			ShortCode:   shortUrl.Id,
			Timestamp:   timestamp(),
			Referrer:    origin.Referrer,
		})
	}()

	if fetchFromUrl, exists := req.Form["f"]; exists && fetchFromUrl[0] != "" {
		content = omni_http.FetchFromUrl(userAgent.Header, fetchFromUrl[0])
		resp.Write([]byte(content))
		return
	}

	if matchedRule != nil {
		openTestHtmlTemplate.Execute(resp, appInstallInterstitialContext{
			Rule: matchedRule,
			IsCrossBrowserContext: userId != uuid,
			Timestamp:             time.Now().Unix(),
		})
		return
	}

	return
}

func (this *ShortyEndPoint) CheckAppInstallInterstitialJSHandler(resp http.ResponseWriter, req *http.Request) {
	omni_http.SetNoCachingHeaders(resp)

	glog.Infoln(">>>> JS")

	vars := mux.Vars(req)
	shortCode := vars["shortCode"]
	uuid := vars["uuid"]

	shortUrl, err := this.service.FindUrl(shortCode)
	if err != nil || shortUrl == nil {
		renderError(resp, req, err.Error(), http.StatusInternalServerError)
		return
	}

	cookies := omni_http.NewCookieHandler(secureCookie, resp, req)
	userAgent := omni_http.ParseUserAgent(req)
	origin, _ := this.requestParser.Parse(req)
	origin.Referrer = "DIRECT" // otherwise it's whatever the url of the page that includes the script

	_, _, _, userId := processCookies(cookies, shortUrl.Id)

	matchedRule, notFound := shortUrl.MatchRule(this.service, userAgent, origin, cookies)
	if notFound != nil {
		renderError(resp, req, "not found", http.StatusNotFound)
		return
	}

	glog.Infoln(">>>>>> Using rule id=", matchedRule.Id, "comment=", matchedRule.Comment)

	context := &appInstallInterstitialContext{
		Rule: matchedRule,
		IsCrossBrowserContext: userId != uuid,
		Timestamp:             time.Now().Unix(),
	}

	var buff bytes.Buffer
	deeplinkJsTemplate.Execute(&buff, context)
	resp.Write(buff.Bytes())

	fmt.Println(buff.String())
	return
}
