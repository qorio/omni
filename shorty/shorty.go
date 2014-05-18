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
	"strings"
	"time"
)

type Settings struct {
	RedisUrl       string
	RedisPrefix    string
	RestrictDomain string
	UrlLength      int
}

type RoutingRule struct {

	// Specify one of the following matching criteria: platform, os, make, or browser
	MatchPlatform string `json:"platform,omitempty"`
	MatchOS       string `json:"os,omitempty"`
	MatchMake     string `json:"make,omitempty"`
	MatchBrowser  string `json:"browser,omitempty"`

	// Destination resource url - can be app url on mobile device
	Destination string `json:"destination"`

	// Inline html.  Inline html takes precedence over destination.
	InlineContent string `json:"inline",omitempty`

	// For specifying mobile appstore install url and app custom url scheme
	// If specified, check cookie to see if the app's url scheme exists, if not, direct to appstore
	AppUrlScheme string `json:"scheme",omitempty`
	AppStoreUrl  string `json:"appstore",omitempty`

	// For matching on mobile=true and referer is equal to the value
	MatchMobileReferrer string `json:"x-mobile-referer",omitempty`
}

func (this *RoutingRule) Match(ua *http.UserAgent, origin *http.RequestOrigin) (destination string, match bool) {
	if len(this.MatchPlatform) > 0 {
		if matches, _ := regexp.MatchString(this.MatchPlatform, ua.Platform); matches {
			return this.Destination, true
		}
	}
	if len(this.MatchOS) > 0 {
		if matches, _ := regexp.MatchString(this.MatchOS, ua.OS); matches {
			return this.Destination, true
		}
	}
	if len(this.MatchMake) > 0 {
		if matches, _ := regexp.MatchString(this.MatchMake, ua.Make); matches {
			return this.Destination, true
		}
	}
	if len(this.MatchBrowser) > 0 {
		if matches, _ := regexp.MatchString(this.MatchBrowser, ua.Browser); matches {
			return this.Destination, true
		}
	}
	if origin != nil && ua.Mobile && len(this.MatchMobileReferrer) > 0 {
		if matches, _ := regexp.MatchString(this.MatchMobileReferrer, origin.Referrer); matches {
			return this.Destination, true
		}
	}
	return "", false
}

type ShortUrl struct {
	Id          string        `json:"id"`
	Rules       []RoutingRule `json:"rules"`
	Destination string        `json:"destination"`
	Created     time.Time     `json:"created"`
	Origin      string        `json:"origin"`
	AppKey      string        `json:"appKey"`
	CampaignKey string        `json:"campaignKey"`
	service     *shortyImpl
}

type DecodeEvent struct {
	RequestOrigin *http.RequestOrigin
	Destination   string
	ShortyUUID    string
	// for tracking campaigns
	Origin      string
	AppKey      string
	CampaignKey string
}

type InstallEvent struct {
	RequestOrigin *http.RequestOrigin
	AppUrlScheme  string
	AppUUID       string
	Destination   string
	ShortyUUID    string
	// for tracking campaigns
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
	DecodeEventChannel() <-chan *DecodeEvent
	InstallEventChannel() <-chan *InstallEvent
	TrackInstall(shortyUUID, appUUID, appUrlScheme string) error
	FindInstall(shortyUUID, appUrlScheme string) (appUUID string, found bool, err error)
	PublishDecode(event *DecodeEvent)
	PublishInstall(event *InstallEvent)
	Close()
}

///////////////////////////////////////////////////////////////////////////////////

type shortyImpl struct {
	settings            Settings
	pool                *redis.Pool
	decodeEventChannel  chan *DecodeEvent
	installEventChannel chan *InstallEvent
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

func (this *shortyImpl) DecodeEventChannel() <-chan *DecodeEvent {
	if this.decodeEventChannel == nil {
		this.decodeEventChannel = make(chan *DecodeEvent)
	}
	return this.decodeEventChannel
}

func (this *shortyImpl) InstallEventChannel() <-chan *InstallEvent {
	if this.installEventChannel == nil {
		this.installEventChannel = make(chan *InstallEvent)
	}
	return this.installEventChannel
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
	if len(rules) > 0 {
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
	}

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

func (this *shortyImpl) TrackInstall(shortyUUID, appUUID, appUrlScheme string) error {
	c := this.pool.Get()
	defer c.Close()

	key := fmt.Sprintf("%s-%s", shortyUUID, appUrlScheme)
	reply, err := c.Do("SET", this.settings.RedisPrefix+"install:"+key, appUUID)
	if err == nil && reply != "OK" {
		err = errors.New("Invalid Redis response")
	}
	return err
}

func (this *shortyImpl) FindInstall(shortyUUID, appUrlScheme string) (appUUID string, found bool, err error) {
	c := this.pool.Get()
	defer c.Close()

	key := fmt.Sprintf("%s-%s", shortyUUID, appUrlScheme)
	reply, err := c.Do("GET", this.settings.RedisPrefix+"install:"+key)
	found = reply != nil

	if found {
		appUUID, err = redis.String(reply, err)
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
