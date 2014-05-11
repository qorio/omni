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
	LongUrl string        `json:"longUrl"`
	Rules   []RoutingRule `json:"rules"`
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

		regex := fmt.Sprintf("[A-Za-z0-9]{%d}", service.UrlLength())
		api.router.HandleFunc("/{id:"+regex+"}", api.RedirectHandler).Name("redirect")
		api.router.HandleFunc("/api/v1/url", api.ApiAddHandler).Methods("POST").Name("add")
		api.router.HandleFunc("/api/v1/stats/{id:"+regex+"}", api.StatsHandler).Methods("GET").Name("stats")
		api.router.HandleFunc("/api/v1/events/install/{scheme}/{app_uuid}",
			api.ReportInstallHandler).Methods("GET").Name("app_install")

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

		regex := fmt.Sprintf("[A-Za-z0-9]{%d}", service.UrlLength())
		api.router.HandleFunc("/{id:"+regex+"}", api.RedirectHandler).Name("redirect")

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

	shortUrl, err := this.service.ShortUrl(message.LongUrl, message.Rules)
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

func processCookies(resp http.ResponseWriter, req *http.Request, shortUrl *ShortUrl) (visits int, cookied bool, last, uuid string) {
	secureCookie.ReadCookie(req, "uuid", &uuid)
	secureCookie.ReadCookie(req, "last", &last)
	secureCookie.ReadCookie(req, shortUrl.Id, &visits)

	var cookieError error

	if uuid == "" {
		if uuid, _ = newUUID(); uuid != "" {
			cookieError = secureCookie.SetCookie(resp, "uuid", uuid)
		}
	}

	visits++
	cookieError = secureCookie.SetCookie(resp, "last", shortUrl.Id)
	cookieError = secureCookie.SetCookie(resp, shortUrl.Id, visits)
	cookied = cookieError == nil
	return
}

func (this *ShortyEndPoint) RedirectHandler(resp http.ResponseWriter, req *http.Request) {
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

	// If there are platform-dependent routing
	if len(shortUrl.Rules) > 0 {
		userAgent := omni_http.ParseUserAgent(req)
		for _, rule := range shortUrl.Rules {
			if dest, match := rule.Match(userAgent); match {

				destination = dest // default

				// check to see if the rule specifies app url scheme
				// if yes, then check cookie by the same key exists
				glog.Infoln(">>>> APP URL SCHEME", rule.AppUrlScheme)
				if rule.AppUrlScheme != "" {
					token := ""
					secureCookie.ReadCookie(req, rule.AppUrlScheme, &token)
					glog.Infoln(">>>>> READ INSTALL TOKEN", token)
					if token == "" {
						destination = rule.AppStoreUrl
					}
				}
				break
			}
		}
	}

	omni_http.SetNoCachingHeaders(resp)
	visits, cookied, last, userId := processCookies(resp, req, shortUrl)

	http.Redirect(resp, req, destination, http.StatusMovedPermanently)

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
			Origin:      origin,
			Destination: destination,
			ShortyUUID:  userId,
		})
		shortUrl.Record(origin, visits > 1)
	}()
}

func addQueryParam(url, key, value string) string {
	if strings.ContainsRune(url, '?') {
		return url + "&" + key + "=" + value
	} else {
		return url + "?" + key + "=" + value
	}
}

func (this *ShortyEndPoint) ReportInstallHandler(resp http.ResponseWriter, req *http.Request) {
	omni_http.SetNoCachingHeaders(resp)

	vars := mux.Vars(req)

	// Two parameters
	// 1. app custom url scheme -> this allows us to key by mobile app per user
	// 2. some uuid for the app -> this tracks a user. on ios, idfa uuid is used.

	customUrlScheme := vars["scheme"]
	if customUrlScheme == "" {
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

	secureCookie.ReadCookie(req, "uuid", &userId)
	secureCookie.ReadCookie(req, "last", &lastViewed)

	// set a cookie to note that we know the app has been installed on the device
	setCookieErr := secureCookie.SetCookie(resp, customUrlScheme, "install")

	glog.Infoln(">>>>>  cookied ", customUrlScheme, setCookieErr)

	var shortUrl *ShortUrl
	var err error

	var destination string = customUrlScheme + "://404"
	if lastViewed == "" {
		destination = addQueryParam(destination, "cookie", userId)
		http.Redirect(resp, req, destination, http.StatusMovedPermanently)
		goto stat
	}

	shortUrl, err = this.service.Find(lastViewed)
	if err != nil {
		renderError(resp, req, err.Error(), http.StatusInternalServerError)
		return
	} else if shortUrl == nil {
		destination = addQueryParam(destination, "cookie", userId)
		http.Redirect(resp, req, destination, http.StatusMovedPermanently)
		goto stat
	}

	// If there are platform-dependent routing
	if len(shortUrl.Rules) > 0 {
		userAgent := omni_http.ParseUserAgent(req)
		for _, rule := range shortUrl.Rules {
			if dest, match := rule.Match(userAgent); match {
				destination = dest
				break
			}
		}
	}

	destination = addQueryParam(destination, "cookie", userId)
	http.Redirect(resp, req, destination, http.StatusMovedPermanently)

stat: // Record stats asynchronously
	go func() {
		origin, geoParseErr := this.requestParser.Parse(req)
		origin.Destination = destination

		if shortUrl != nil {
			origin.ShortCode = shortUrl.Id
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
			Origin:       origin,
			Destination:  destination,
			AppUrlScheme: customUrlScheme,
			AppUUID:      appUuid,
			ShortyUUID:   userId,
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
