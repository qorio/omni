package shorty

import (
	"github.com/qorio/omni/http"
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

type RoutingRule struct {

	// Specify one of the following matching criteria: platform, os, make, or browser
	MatchPlatform string `json:"platform,omitempty"`
	MatchOS       string `json:"os,omitempty"`
	MatchMake     string `json:"make,omitempty"`
	MatchBrowser  string `json:"browser,omitempty"`

	MatchMobile   string `json:"mobile,omitempty"`
	MatchReferrer string `json:"referer,omitempty"`

	// True to check if there's an install of the app by AppUrlScheme
	MatchInstalled string `json:"installed,omitempty"`

	// Match by no app-open or if app-open is X days ago
	MatchNoAppOpenInXDays int64 `json:"no-app-open-in-x-days,omitempty"`

	// For specifying mobile appstore install url and app custom url scheme
	// If specified, check cookie to see if the app's url scheme exists, if not, direct to appstore
	AppUrlScheme string `json:"scheme,omitempty"`
	AppStoreUrl  string `json:"appstore,omitempty"`

	// Destination resource url - can be app url on mobile device
	Destination string `json:"destination,omitempty"`

	// Fetch content from url
	ContentSourceUrl string `json:"content-src-url,omitempty"`

	// Send to an interstitial page
	SendToInterstitial bool `json:"x-send-to-interstitial,omitempty"`

	// True to disasble app store redirection
	NoAppStoreRedirect bool `json:"x-no-app-store-redirect,omitempty"`

	// True to indicate that this is a http url destination but mapped in the intent filter to an app.
	IsAndroidIntentFilter bool `json:"x-android-intent-filter,omitempty"`

	Special []RoutingRule `json:"special,omitempty"`
}

type ShortUrl struct {
	Id                string        `json:"id,omitempty"`
	Rules             []RoutingRule `json:"rules,omitempty"`
	Destination       string        `json:"destination,omitempty"`
	Created           time.Time     `json:"created,omitempty"`
	Origin            string        `json:"origin,omitempty"`
	AppKey            string        `json:"appKey,omitempty"`
	CampaignKey       string        `json:"campaignKey,omitempty"`
	InstallTTLSeconds int64         `json:"installTTLSeconds,omitempty"`
	service           *shortyImpl
}

type Campaign struct {
	AppKey    string        `json:"appKey,omitempty"`
	Id        string        `json:"id,omitempty"`
	Rules     []RoutingRule `json:"rules,omitempty"`
	Created   time.Time     `json:"created,omitempty"`
	AppHasSDK bool          `json:"appHasSDK,omitempty"`
}

type FingerprintedVisit struct {
	Fingerprint string
	Context     UUID   `json:"uuid,omitempty"`
	ShortCode   string `json:"shortCode,omitempty"`
	Deeplink    string `json:"deeplink,omitempty"`
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
	AppKey      string
	CampaignKey string
}

type DecodeEvent struct {
	RequestOrigin *http.RequestOrigin
	Destination   string
	Context       UUID

	Origin      string
	AppKey      string
	CampaignKey string

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
	AppKey      string
	CampaignKey string

	ReportingMethod string // 'fingerprint', 'browser-switch', 'referred-app-open'
}

type LinkEvent struct {
	RequestOrigin *http.RequestOrigin
	App           UrlScheme
	Context1      UUID
	Context2      UUID

	Origin      string
	AppKey      string
	CampaignKey string
}

type Shorty interface {
	UrlLength() int

	// Validates the url, rules and create a new instance with the default properties set in the defaults param.
	ShortUrl(url string, optionalRules []RoutingRule, defaults ShortUrl) (*ShortUrl, error)
	// Validates the url, rules and create a new instance with the default properties set in the defaults param.
	VanityUrl(vanity, url string, optionalRules []RoutingRule, defaults ShortUrl) (*ShortUrl, error)
	Find(id string) (*ShortUrl, error)

	Link(appUrlScheme UrlScheme, prevContext, currentContext UUID, shortUrlId string) error
	FindLink(appUuid, uuid UUID) (found bool, err error)

	TrackInstall(app UrlScheme, context UUID) error
	FindInstall(app UrlScheme, context UUID) (expiration int64, found bool, err error)

	TrackAppOpen(app UrlScheme, appContext UUID, appOpen *AppOpen) error
	FindAppOpen(app UrlScheme, context UUID) (appOpen *AppOpen, found bool, err error)

	SaveFingerprintedVisit(visit *FingerprintedVisit) error
	MatchFingerPrint(fingerprint string) (score float64, visit *FingerprintedVisit, err error)

	DecodeEventChannel() <-chan *DecodeEvent
	InstallEventChannel() <-chan *InstallEvent
	LinkEventChannel() <-chan *LinkEvent
	AppOpenEventChannel() <-chan *AppOpenEvent

	PublishDecode(event *DecodeEvent)
	PublishInstall(event *InstallEvent)
	PublishLink(event *LinkEvent)
	PublishAppOpen(event *AppOpenEvent)

	Close()
}
