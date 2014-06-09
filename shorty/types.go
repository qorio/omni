package shorty

import (
	"github.com/qorio/omni/http"
	"strings"
	"time"
)

type UUID string
type UrlScheme string

type Settings struct {
	RedisUrl       string
	RedisPrefix    string
	RestrictDomain string
	UrlLength      int
}

type ShortyAddRequest struct {
	Vanity   string        `json:"vanity"`
	LongUrl  string        `json:"longUrl"`
	Rules    []RoutingRule `json:"rules"`
	Origin   string        `json:"origin"`
	ApiToken string        `json:"token"` // user facing token that resolves to appKey
	Campaign string        `json:"campaign"`
}

type Campaign struct {
	Id          UUID          `json:"id,omitempty"`
	Name        string        `json:"name,omitempty"`
	Description string        `json:"description,omitempty"`
	AppKey      UUID          `json:"appKey"`
	Rules       []RoutingRule `json:"rules,omitempty"`
	Created     int64         `json:"created,omitempty"`
	IOS_SDK     bool          `json:"iosSDK,omitempty"`

	service *shortyImpl
}

type OnOff string
type Regex string

func (this OnOff) isTrue() bool {
	return strings.ToLower(string(this)) == "on"
}

type RoutingRule struct {
	Id      string `json:"id,omitempty"`
	Comment string `json:"comment,omitempty"`

	// For specifying mobile appstore install url and app custom url scheme
	// If specified, check cookie to see if the app's url scheme exists, if not, direct to appstore
	AppUrlScheme string `json:"scheme,omitempty"`
	AppStoreUrl  string `json:"appstore,omitempty"`

	// TTL max days for the app open to be considered expired (eg. app possibly deleted from device)
	AppOpenTTLDays float64 `json:"app-open-ttl-days,omitempty"`

	// Specify one of the following matching criteria: platform, os, make, or browser
	MatchPlatform Regex `json:"platform,omitempty"`
	MatchOS       Regex `json:"os,omitempty"`
	MatchMake     Regex `json:"make,omitempty"`
	MatchBrowser  Regex `json:"browser,omitempty"`

	MatchMobile   OnOff `json:"mobile,omitempty"`
	MatchReferrer Regex `json:"referer,omitempty"`

	// Match by no app-open or if app-open is X days ago -- uses the AppOpenTTLDays
	MatchNoAppOpenInXDays OnOff `json:"match-no-app-open-in-ttl-days,omitempty"`

	// Destination resource url - can be app url on mobile device
	Destination string `json:"destination,omitempty"`

	// Fetch content from url
	ContentSourceUrl string `json:"content-src-url,omitempty"`

	// Send to an interstitial page
	SendToInterstitial OnOff `json:"x-send-to-interstitial,omitempty"`

	InterstitialToAppStoreOnTimeout  OnOff `json:"x-interstitial-to-appstore-on-timeout,omitempty"`
	InterstitialAppLinkTimeoutMillis int64 `json:"x-interstitial-open-app-timeout-millis,omitempty"`

	// True to indicate that this is a http url destination but mapped in the intent filter to an app.
	IsAndroidIntentFilter OnOff `json:"x-android-intent-filter,omitempty"`

	// Nested rules that can further provide overrides -- e.g. on 'ios', now with FB app
	Special []RoutingRule `json:"special,omitempty"`
}

type ShortUrl struct {
	Id          string        `json:"id"`
	Rules       []RoutingRule `json:"rules,omitempty"`
	Destination string        `json:"destination"`
	Created     time.Time     `json:"created,omitempty"`
	Origin      string        `json:"origin,omitempty"`
	AppKey      UUID          `json:"appKey"`
	CampaignKey UUID          `json:"campaignKey"`
	service     *shortyImpl
}

type FingerprintedVisit struct {
	Fingerprint string
	Context     UUID   `json:"uuid,omitempty"`
	ShortCode   string `json:"shortCode,omitempty"`
	Deeplink    string `json:"deeplink,omitempty"`
	Visit       string
	Timestamp   int64
	Referrer    string `json:"sourceApplication,omitempty"`
}

type AppOpen struct {
	SourceApplication string `json:"sourceApplication,omitempty"`
	SourceContext     UUID   `json:"uuid,omitempty"`
	ShortCode         string `json:"shortCode,omitempty"`
	Deeplink          string `json:"deeplink,omitempty"`
	Timestamp         int64
	AppContext        UUID
}

type AppOpenEvent struct {
	RequestOrigin *http.RequestOrigin

	App         UrlScheme
	AppContext  UUID
	Destination string

	SourceContext     UUID
	SourceApplication string

	Origin      string
	AppKey      UUID
	CampaignKey UUID
}

type DecodeEvent struct {
	RequestOrigin *http.RequestOrigin
	Destination   string
	Context       UUID

	Origin      string
	AppKey      UUID
	CampaignKey UUID

	MatchedRuleIndex int
}

type InstallEvent struct {
	RequestOrigin *http.RequestOrigin
	App           UrlScheme
	AppContext    UUID
	Destination   string

	SourceContext     UUID
	SourceApplication string

	Origin      string
	AppKey      UUID
	CampaignKey UUID

	ReportingMethod string // 'fingerprint', 'browser-switch', 'referred-app-open'
}

type LinkEvent struct {
	RequestOrigin *http.RequestOrigin
	App           UrlScheme
	Context1      UUID
	Context2      UUID

	Origin      string
	AppKey      UUID
	CampaignKey UUID
}
