package shorty

import (
	"github.com/qorio/api"
	"strings"
	"time"
)

const (
	ManageCampaigns api.AuthScope = iota
	CreateShortUrls
)

var AuthScopes = api.AuthScopes{
	ManageCampaigns: "manage_campaigns",
	CreateShortUrls: "create_shorturls",
}

const (
	AddCampaign api.ServiceMethod = iota
	GetCampaign
	UpdateCampaign
	AddShortUrlForCampaign
	AddShortUrl
)

var Methods = api.ServiceMethods{

	AddCampaign: api.MethodSpec{
		AuthScope: AuthScopes[ManageCampaigns],
		Doc: `
Adds a new campaign.
`,
		UrlRoute:     "/api/v1/campaign",
		HttpMethod:   "POST",
		ContentTypes: []string{"application/json"},
		RequestBody: func() interface{} {
			return Campaign{}
		},
		ResponseBody: func() interface{} {
			return Campaign{}
		},
	},

	GetCampaign: api.MethodSpec{
		AuthScope: AuthScopes[ManageCampaigns],
		Doc: `
Returns the campaign specified.
`,
		UrlRoute:     "/api/v1/campaign/{campaignId}",
		HttpMethod:   "GET",
		ContentTypes: []string{"application/json"},
		RequestBody:  nil,
		ResponseBody: func() interface{} {
			return Campaign{}
		},
	},

	UpdateCampaign: api.MethodSpec{
		AuthScope: AuthScopes[ManageCampaigns],
		Doc: `
Updates the campaign specified.
`,
		UrlRoute:     "/api/v1/campaign/{campaignId}",
		HttpMethod:   "POST",
		ContentTypes: []string{"application/json"},
		RequestBody: func() interface{} {
			return Campaign{}
		},
		ResponseBody: func() interface{} {
			return Campaign{}
		},
	},

	AddShortUrlForCampaign: api.MethodSpec{
		AuthScope: AuthScopes[CreateShortUrls],
		Doc: `
Create short url using the rules specified in a campaign as template.
`,
		UrlRoute:     "/api/v1/campaign/{campaignId}/url",
		HttpMethod:   "POST",
		ContentTypes: []string{"application/json"},
		RequestBody: func() interface{} {
			return ShortyAddRequest{}
		},
		ResponseBody: func() interface{} {
			return ShortUrl{}
		},
	},

	AddShortUrl: api.MethodSpec{
		AuthScope: AuthScopes[CreateShortUrls],
		Doc: `
Create short url by specifying all the routing rules.
`,
		UrlRoute:     "/api/v1/url",
		HttpMethod:   "POST",
		ContentTypes: []string{"application/json"},
		RequestBody: func() interface{} {
			return ShortyAddRequest{}
		},
		ResponseBody: func() interface{} {
			return ShortUrl{}
		},
	},
}

type UUID string
type UrlScheme string
type OnOff string
type Regex string

type ShortyAddRequest struct {
	Vanity   string        `json:"vanity"`
	LongUrl  string        `json:"longUrl"`
	Rules    []RoutingRule `json:"rules"`
	Origin   string        `json:"origin"`
	Campaign string        `json:"campaign"`
}

type ShortUrl struct {
	Id          string        `json:"id"`
	Rules       []RoutingRule `json:"rules,omitempty"`
	Destination string        `json:"destination"`
	Created     time.Time     `json:"created,omitempty"`
	Origin      string        `json:"origin,omitempty"`
	AccountId   UUID          `json:"accountId"`
	CampaignId  UUID          `json:"campaignId"`
	//	service     *shortyImpl
}

type Campaign struct {
	Id          UUID          `json:"id,omitempty"`
	Name        string        `json:"name,omitempty"`
	Description string        `json:"description,omitempty"`
	AccountId   UUID          `json:"accountId"`
	Rules       []RoutingRule `json:"rules,omitempty"`
	Created     int64         `json:"created,omitempty"`
	IOS_SDK     bool          `json:"iosSDK,omitempty"`
	//	service *shortyImpl
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

	// iOS8 Safari webkitHidden property doesn't seem to work
	CheckWebkitHidden OnOff `json:"x-check-webkit-hidden,omitempty"`

	// True to indicate that this is a http url destination but mapped in the intent filter to an app.
	IsAndroidIntentFilter OnOff `json:"x-android-intent-filter,omitempty"`

	// Nested rules that can further provide overrides -- e.g. on 'ios', now with FB app
	Special []RoutingRule `json:"special,omitempty"`
}

func (this OnOff) isTrue() bool {
	return strings.ToLower(string(this)) == "on"
}
