package shorty

import (
	"crypto/rand"
	"encoding/json"
	"errors"
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
	MatchPlatform string `json:"platform,omitempty"`
	MatchOS       string `json:"os,omitempty"`
	MatchMake     string `json:"make,omitempty"`
	MatchBrowser  string `json:"browser,omitempty"`
	Destination   string `json:"destination"`
}

func (this *RoutingRule) Match(ua *http.UserAgent) (destination string, match bool) {
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
	return "", false
}

type ShortUrl struct {
	Id          string        `json:"id"`
	Rules       []RoutingRule `json:"rules"`
	Destination string        `json:"destination"`
	Created     time.Time     `json:"created"`
	service     *shortyImpl
}

type Shorty interface {
	UrlLength() int
	ShortUrl(url string, optionalRules []RoutingRule) (*ShortUrl, error)
	Find(id string) (*ShortUrl, error)
	Channel() <-chan *http.RequestOrigin
	Publish(req *http.RequestOrigin)
	Close()
}

///////////////////////////////////////////////////////////////////////////////////

type shortyImpl struct {
	settings Settings
	pool     *redis.Pool
	channel  chan *http.RequestOrigin
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

func (this *shortyImpl) Publish(req *http.RequestOrigin) {
	if this.channel != nil {
		this.channel <- req
	}
}

func (this *shortyImpl) Channel() <-chan *http.RequestOrigin {
	if this.channel == nil {
		this.channel = make(chan *http.RequestOrigin)
	}
	return this.channel
}

func (this *shortyImpl) UrlLength() int {
	return this.settings.UrlLength
}

func (this *shortyImpl) ShortUrl(data string, rules []RoutingRule) (entity *ShortUrl, err error) {
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

	bytes := make([]byte, this.settings.UrlLength)
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

	entity.Save()

	return entity, nil
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
