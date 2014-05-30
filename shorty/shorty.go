package shorty

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/garyburd/redigo/redis"
	"github.com/golang/glog"
	"github.com/qorio/omni/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type Settings struct {
	RedisUrl       string
	RedisPrefix    string
	RestrictDomain string
	UrlLength      int
}

type FingerprintedVisit struct {
	Fingerprint string
	UUID        string `json:"uuid"`
	ShortCode   string `json:"shortCode"`
	Deeplink    string `json:"deeplink"`
	Timestamp   int64
	Referrer    string `json:"sourceApplication"`
}

type RoutingRule struct {

	// Matching rule -- all or any of the Match<type> properties
	MatchAll bool `json:"match-all"`

	// Specify one of the following matching criteria: platform, os, make, or browser
	MatchPlatform string `json:"platform,omitempty"`
	MatchOS       string `json:"os,omitempty"`
	MatchMake     string `json:"make,omitempty"`
	MatchBrowser  string `json:"browser,omitempty"`

	MatchMobile   string `json:"mobile",omitempty`
	MatchReferrer string `json:"referer",omitempty`

	// True to check if there's an install of the app by AppUrlScheme
	MatchInstalled string `json:"installed",omitempty`

	// For specifying mobile appstore install url and app custom url scheme
	// If specified, check cookie to see if the app's url scheme exists, if not, direct to appstore
	AppUrlScheme string `json:"scheme",omitempty`
	AppStoreUrl  string `json:"appstore",omitempty`

	// Destination resource url - can be app url on mobile device
	Destination string `json:"destination"`

	// Fetch content from url
	ContentSourceUrl string `json:"content-src-url",omitempty`

	// True to harvest the cookied uuid via a redirect to a url containing the uuid
	HarvestCookiedUUID bool `json:"x-harvest-cookied-uuid",omitempty`

	// Send to an interstitial page
	SendToInterstitial bool `json:"x-send-to-interstitial"`

	// True to disasble app store redirection
	NoAppStoreRedirect bool `json:"x-no-app-store-redirect"`

	// True to indicate that this is a http url destination but mapped in the intent filter to an app.
	IsAndroidIntentFilter bool `json:"x-android-intent-filter"`
}

func (this *RoutingRule) Match(service Shorty, ua *http.UserAgent, origin *http.RequestOrigin, cookies http.Cookies) bool {
	// use bit mask to match
	var actual, expect int = 0, 0

	if len(this.MatchPlatform) > 0 {
		expect |= 1 << 0
		if matches, _ := regexp.MatchString(this.MatchPlatform, ua.Platform); matches {
			actual |= 1 << 0
		}
	}
	if len(this.MatchOS) > 0 {
		expect |= 1 << 1
		if matches, _ := regexp.MatchString(this.MatchOS, ua.OS); matches {
			actual |= 1 << 1
		}
	}
	if len(this.MatchMake) > 0 {
		expect |= 1 << 2
		if matches, _ := regexp.MatchString(this.MatchMake, ua.Make); matches {
			actual |= 1 << 2
		}
	}
	if len(this.MatchBrowser) > 0 {
		expect |= 1 << 3
		if matches, _ := regexp.MatchString(this.MatchBrowser, ua.Browser); matches {
			actual |= 1 << 3
		}
	}
	if len(this.MatchMobile) > 0 {
		expect |= 1 << 4
		if matches, _ := regexp.MatchString(this.MatchMobile, strconv.FormatBool(ua.Mobile)); matches {
			actual |= 1 << 4
		}
	}
	if len(this.MatchReferrer) > 0 {
		expect |= 1 << 5
		if matches, _ := regexp.MatchString(this.MatchReferrer, origin.Referrer); matches {
			actual |= 1 << 5
		}
	}
	if len(this.MatchInstalled) > 0 {
		expect |= 1 << 6
		if this.AppUrlScheme != "" {
			uuid, _ := cookies.GetPlainString(uuidCookieKey)
			_, found, _ := service.FindInstall(uuid, this.AppUrlScheme)
			glog.Infoln("checking install", uuid, this.AppUrlScheme, found)
			if matches, _ := regexp.MatchString(this.MatchInstalled, strconv.FormatBool(found)); matches {
				actual |= 1 << 6
			}
		}
	}
	// By the time we get here, we have done a match all
	return actual == expect && expect > 0
}

type ShortUrl struct {
	Id                string        `json:"id"`
	Rules             []RoutingRule `json:"rules"`
	Destination       string        `json:"destination"`
	Created           time.Time     `json:"created"`
	Origin            string        `json:"origin"`
	AppKey            string        `json:"appKey"`
	CampaignKey       string        `json:"campaignKey"`
	InstallTTLSeconds int64         `json:"installTTLSeconds"`
	service           *shortyImpl
}

type Campaign struct {
	AppKey            string        `json:"appKey"`
	Id                string        `json:"id"`
	Rules             []RoutingRule `json:"rules"`
	Created           time.Time     `json:"created"`
	InstallTTLSeconds int64         `json:"installTTLSeconds"`
	AppHasSDK         bool          `json:"appHasSDK"`
}

type DecodeEvent struct {
	RequestOrigin *http.RequestOrigin
	Destination   string
	ShortyUUID    string
	// for tracking campaigns
	Origin      string
	AppKey      string
	CampaignKey string

	MatchedRuleIndex int
}

type InstallEvent struct {
	RequestOrigin     *http.RequestOrigin
	AppUrlScheme      string
	AppUUID           string
	Destination       string
	SourceUUID        string
	SourceApplication string

	Origin      string
	AppKey      string
	CampaignKey string

	ReportingMethod string // 'fingerprint', 'browser-switch', 'referred-app-open'
}

type LinkEvent struct {
	RequestOrigin *http.RequestOrigin
	AppUrlScheme  string
	ShortyUUID_A  string
	ShortyUUID_B  string

	Origin      string
	AppKey      string
	CampaignKey string
}

type AppOpenEvent struct {
	RequestOrigin     *http.RequestOrigin
	AppUrlScheme      string
	AppUUID           string
	Destination       string
	SourceUUID        string
	SourceApplication string

	Origin      string
	AppKey      string
	CampaignKey string
}

type UUID string
type UrlScheme string

type Shorty interface {
	UrlLength() int

	// Validates the url, rules and create a new instance with the default properties set in the defaults param.
	ShortUrl(url string, optionalRules []RoutingRule, defaults ShortUrl) (*ShortUrl, error)
	// Validates the url, rules and create a new instance with the default properties set in the defaults param.
	VanityUrl(vanity, url string, optionalRules []RoutingRule, defaults ShortUrl) (*ShortUrl, error)
	Find(id string) (*ShortUrl, error)

	Link(appUrlScheme UrlScheme, shortyUUIDContextPrev, shortyUUIDContextCurrent UUID, shortUrlId string) error
	FindLink(appUuid, uuid UUID) (found bool, err error)

	TrackInstall(shortyUUID, appUrlScheme string) error
	FindInstall(shortyUUID, appUrlScheme string) (expiration int64, found bool, err error)

	TrackAppOpen(appUrlScheme, appUuid, uuid, sourceApplication, shortCode string) error
	FindAppOpen(appUrlScheme, uuid string) (timestamp int64, found bool, err error)

	SaveFingerprintedVisit(visit *FingerprintedVisit) error
	MatchFingerPrint(fingerprint string) (score float64, visit *FingerprintedVisit, err error)

	DecodeEventChannel() <-chan *DecodeEvent
	InstallEventChannel() <-chan *InstallEvent
	AppOpenEventChannel() <-chan *AppOpenEvent

	PublishDecode(event *DecodeEvent)
	PublishInstall(event *InstallEvent)
	PublishLink(event *LinkEvent)
	PublishAppOpen(event *AppOpenEvent)

	Close()
}

///////////////////////////////////////////////////////////////////////////////////

type shortyImpl struct {
	settings            Settings
	pool                *redis.Pool
	decodeEventChannel  chan *DecodeEvent
	installEventChannel chan *InstallEvent
	linkEventChannel    chan *LinkEvent
	appOpenEventChannel chan *AppOpenEvent
}

const (
	alphanum = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
)

func Init(settings Settings) *shortyImpl {
	return &shortyImpl{
		settings: settings,
		pool: &redis.Pool{
			MaxIdle:     5,
			IdleTimeout: 240 * time.Second,
			Dial: func() (redis.Conn, error) {
				c, err := redis.Dial("tcp", settings.RedisUrl)
				if err != nil {
					return nil, err
				}
				return c, nil
			},
			TestOnBorrow: func(c redis.Conn, t time.Time) error {
				_, err := c.Do("PING")
				return err
			},
		},
	}
}

func (this *shortyImpl) Close() {
	if err := this.pool.Close(); err != nil {
		glog.Warningln("error-closing-connection-pool", err)
	}
}

func (this *shortyImpl) PublishDecode(event *DecodeEvent) {
	if this.decodeEventChannel != nil {
		this.decodeEventChannel <- event
	}
}

func (this *shortyImpl) PublishInstall(event *InstallEvent) {
	if this.installEventChannel != nil {
		this.installEventChannel <- event
	}
}

func (this *shortyImpl) PublishAppOpen(event *AppOpenEvent) {
	if this.appOpenEventChannel != nil {
		this.appOpenEventChannel <- event
	}
}

func (this *shortyImpl) PublishLink(event *LinkEvent) {
	if this.linkEventChannel != nil {
		this.linkEventChannel <- event
	}
}

func (this *shortyImpl) DecodeEventChannel() <-chan *DecodeEvent {
	if this.decodeEventChannel == nil {
		this.decodeEventChannel = make(chan *DecodeEvent)
	}
	return this.decodeEventChannel
}

func (this *shortyImpl) LinkEventChannel() <-chan *LinkEvent {
	if this.linkEventChannel == nil {
		this.linkEventChannel = make(chan *LinkEvent)
	}
	return this.linkEventChannel
}

func (this *shortyImpl) InstallEventChannel() <-chan *InstallEvent {
	if this.installEventChannel == nil {
		this.installEventChannel = make(chan *InstallEvent)
	}
	return this.installEventChannel
}

func (this *shortyImpl) AppOpenEventChannel() <-chan *AppOpenEvent {
	if this.appOpenEventChannel == nil {
		this.appOpenEventChannel = make(chan *AppOpenEvent)
	}
	return this.appOpenEventChannel
}

func (this *shortyImpl) UrlLength() int {
	return this.settings.UrlLength
}

func (this *shortyImpl) ShortUrl(data string, rules []RoutingRule, defaults ShortUrl) (entity *ShortUrl, err error) {
	return this.VanityUrl("", data, rules, defaults)
}

func (this *shortyImpl) VanityUrl(vanity, data string, rules []RoutingRule, defaults ShortUrl) (entity *ShortUrl, err error) {
	data = strings.TrimSpace(data)
	if len(data) == 0 {
		err = errors.New("Please specify an URL")
		return
	}

	u, err := url.Parse(data)
	if err != nil {
		return
	}

	if len(this.settings.RestrictDomain) > 0 {
		if matches, _ := regexp.MatchString("^[A-Za-z0-9.]*"+this.settings.RestrictDomain, u.Host); !matches {
			err = errors.New("Only URLs on " + this.settings.RestrictDomain + " domain allowed")
			return
		}
	}

	entity = &ShortUrl{Destination: u.String(), Created: time.Now(), service: this}
	for _, rule := range entity.Rules {
		if len(rule.Destination) > 0 {
			if _, err = url.Parse(rule.Destination); err != nil {
				return
			}
		}

		matching := 0
		if c, err := regexp.Compile(rule.MatchPlatform); err != nil {
			return nil, errors.New("Bad platform regex " + rule.MatchPlatform)
		} else if c != nil {
			matching++
		}
		if c, err := regexp.Compile(rule.MatchOS); err != nil {
			return nil, errors.New("Bad os regex " + rule.MatchOS)
		} else if c != nil {
			matching++
		}
		if c, err := regexp.Compile(rule.MatchMake); err != nil {
			return nil, errors.New("Bad make regex " + rule.MatchMake)
		} else if c != nil {
			matching++
		}
		if c, err := regexp.Compile(rule.MatchBrowser); err != nil {
			return nil, errors.New("Bad browser regex " + rule.MatchBrowser)
		} else if c != nil {
			matching++
		}
		// Must have 1 or more matching regexp
		if matching == 0 {
			err = errors.New("bad-routing-rule:no matching regexp")
			return
		}
	}
	entity.Rules = rules

	c := this.pool.Get()
	defer c.Close()

	if vanity != "" {
		if exists, _ := redis.Bool(c.Do("EXISTS", this.settings.RedisPrefix+"url:"+vanity)); !exists {
			entity.Id = vanity
		}
		if exists, _ := this.Find(vanity); exists == nil {
			entity.Id = vanity
		}
	} else {
		bytes := make([]byte, this.settings.UrlLength)
		// TODO - probably should add a limit so we don't spend forever exploring the number space
		// Not adding it because right now the number space is large, don't expect collisions.
		for {
			rand.Read(bytes)
			for i, b := range bytes {
				bytes[i] = alphanum[b%byte(len(alphanum))]
			}
			id := string(bytes)
			if exists, _ := redis.Bool(c.Do("EXISTS", this.settings.RedisPrefix+"url:"+id)); !exists {
				entity.Id = id
				break
			}

			if exists, _ := this.Find(id); exists == nil {
				entity.Id = id
				break
			}
		}
	}

	if entity.Id != "" {
		// copy the defaults
		entity.Origin = defaults.Origin
		entity.AppKey = defaults.AppKey
		entity.CampaignKey = defaults.CampaignKey

		entity.Save()
		return entity, nil
	} else if vanity != "" {
		return nil, errors.New("Vanity code taken:" + vanity)
	} else {
		return nil, errors.New("Failed to assign code")
	}
}

func (this *shortyImpl) Link(appUrlScheme UrlScheme, shortyUUIDContextPrev, shortyUUIDContextCurrent UUID, shortUrlId string) error {
	c := this.pool.Get()
	defer c.Close()

	// The key allows searching A:B or B:A by making it A:B:A
	key := fmt.Sprintf("%s:%s:%s:%s", appUrlScheme, shortyUUIDContextPrev, shortyUUIDContextCurrent, shortyUUIDContextPrev)
	reply, err := c.Do("SET", this.settings.RedisPrefix+"uuid-pair:"+key, shortUrlId)
	if err == nil && reply != "OK" {
		err = errors.New("Invalid Redis response")
	}
	return err
}

func (this *shortyImpl) FindLink(appUuid, uuid UUID) (found bool, err error) {
	c := this.pool.Get()
	defer c.Close()

	// The key allows searching A:B or B:A by making it A:B:A
	wildcard := fmt.Sprintf("*%s:%s*", appUuid, uuid)
	reply, err := redis.String(c.Do("GET", this.settings.RedisPrefix+"uuid-pair:"+wildcard))
	if err == nil && reply != "" {
		found = true
		return
	}
	return
}

func (this *shortyImpl) SaveFingerprintedVisit(visit *FingerprintedVisit) error {
	c := this.pool.Get()
	defer c.Close()

	value, err := json.Marshal(visit)
	if err != nil {
		return err
	}
	reply, err := c.Do("SET", this.settings.RedisPrefix+"fingerprint:"+visit.Fingerprint, value)
	if err == nil && reply != "OK" {
		err = errors.New("Invalid Redis response")
	}
	return err
}

func (this *shortyImpl) MatchFingerPrint(fingerprint string) (score float64, visit *FingerprintedVisit, err error) {
	c := this.pool.Get()
	defer c.Close()

	q := http.GetFingerPrintMatchQuery(fingerprint)
	reply, err := redis.Strings(c.Do("KEYS", this.settings.RedisPrefix+"fingerprint:"+q))
	if err != nil {
		glog.Warningln("Error", err)
		return
	}
	glog.Infoln("matching fingerprint: fingerprint=", fingerprint, "q=", q, "result=", reply)

	// need to remove the prefix before matching
	candidates := make([]string, len(reply))
	for i, v := range reply {
		candidates[i] = strings.Split(v, this.settings.RedisPrefix+"fingerprint:")[1]
	}
	match, matchScore := http.MatchFingerPrint(fingerprint, candidates)
	glog.Infoln("matching fingerprint: match=", match, "score=", score)

	value, err2 := redis.Bytes(c.Do("GET", this.settings.RedisPrefix+"fingerprint:"+match))
	if err2 != nil {
		glog.Warningln("Error", err2)
	}
	if err3 := json.Unmarshal(value, &visit); err3 == nil {
		score = matchScore
		return
	}

	return
}

func (this *shortyImpl) TrackInstall(shortyUUID, appUrlScheme string) error {
	c := this.pool.Get()
	defer c.Close()

	// TODO - look at every pair that contains this uuid, get the other uuid and create
	// a record as well. This will speed up the read when decoding the shortlink to see if installed.
	key := fmt.Sprintf("%s:%s", shortyUUID, appUrlScheme)
	reply, err := c.Do("SET", this.settings.RedisPrefix+"install:"+key, timestamp())
	if err == nil && reply != "OK" {
		err = errors.New("Invalid Redis response")
	}
	return err
}

func (this *shortyImpl) TrackAppOpen(appUrlScheme, appUuid, uuid, sourceApplication, shortCode string) error {
	c := this.pool.Get()
	defer c.Close()

	key := fmt.Sprintf("%s:%s:%s:%s:%s", appUrlScheme, appUuid, uuid, shortCode, sourceApplication)
	reply, err := c.Do("SET", this.settings.RedisPrefix+"app-open:"+key, timestamp())
	if err == nil && reply != "OK" {
		err = errors.New("Invalid Redis response")
	}
	return err
}

func (this *shortyImpl) FindAppOpen(appUrlScheme, shortyUUID string) (timestamp int64, found bool, err error) {
	c := this.pool.Get()
	defer c.Close()

	key := fmt.Sprintf("%s:*:%s:*", appUrlScheme, shortyUUID)
	reply, err := c.Do("GET", this.settings.RedisPrefix+"app-open:"+key)
	found = reply != nil

	if found {
		timestamp, err = redis.Int64(reply, err)
	}
	return
}

func (this *shortyImpl) FindInstall(shortyUUID, appUrlScheme string) (expiration int64, found bool, err error) {
	c := this.pool.Get()
	defer c.Close()

	key := fmt.Sprintf("%s:%s", shortyUUID, appUrlScheme)
	reply, err := c.Do("GET", this.settings.RedisPrefix+"install:"+key)
	found = reply != nil

	if found {
		expiration, err = redis.Int64(reply, err)
	}
	return
}

func (this *shortyImpl) Find(id string) (*ShortUrl, error) {
	c := this.pool.Get()
	defer c.Close()

	reply, err := c.Do("GET", this.settings.RedisPrefix+"url:"+id)
	if reply == nil {
		return nil, nil
	}

	data, err := redis.Bytes(reply, err)
	if err != nil {
		return nil, err
	}

	url := ShortUrl{service: this}
	if err := json.Unmarshal(data, &url); err != nil {
		return nil, err
	}

	return &url, nil
}

func (this *ShortUrl) Save() error {
	c := this.service.pool.Get()
	defer c.Close()

	data, err := json.Marshal(this)
	if err != nil {
		return err
	}

	reply, err := c.Do("SET", this.service.settings.RedisPrefix+"url:"+this.Id, data)
	if err == nil && reply != "OK" {
		err = errors.New("Invalid Redis response")
	}

	if err != nil {
		return err
	}

	return nil
}

func (this *ShortUrl) Delete() error {
	c := this.service.pool.Get()
	defer c.Close()

	reply, err := c.Do("DEL", this.service.settings.RedisPrefix+"url:"+this.Id)
	if err == nil && reply != "OK" {
		return errors.New("Invalid Redis response")
	}

	return nil
}

func timestamp() int64 {
	return time.Now().Unix()
}
