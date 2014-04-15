package shorty

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	// "fmt"
	"github.com/garyburd/redigo/redis"
	//	"math"
	"net/url"
	"regexp"
	//	"sort"
	//	"strconv"
	"strings"
	"time"
)

type Settings struct {
	RedisUrl       string
	RedisPrefix    string
	RestrictDomain string
	UrlLength      int
}

type Shorty interface {
	UrlLength() int
	ShortUrl(url string) (ShortUrl, error)
	Find(id string) (*ShortUrl, error)
}

type shortyImpl struct {
	settings Settings
	pool     *redis.Pool
}

const (
	alphanum = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	keyy     = "year:%d"
	keym     = "month:%0d-%0.2d"
	keyd     = "day:%d-%0.2d-%0.2d"
	keyh     = "hour:%d-%0.2d-%0.2d %0.2d"
	keyi     = "minute:%d-%0.2d-%0.2d %0.2d:%0.2d"
)

type ShortUrl struct {
	Id          string
	Destination string
	Created     time.Time
	service     *shortyImpl
}

//var pool *redis.Pool

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

func (this *shortyImpl) UrlLength() int {
	return this.settings.UrlLength
}

func (this *shortyImpl) ShortUrl(data string) (entity *ShortUrl, err error) {
	data = strings.TrimSpace(data)
	if len(data) == 0 {
		err = errors.New("Please specify an URL")
		return
	}

	if matches, _ := regexp.MatchString("^https?", data); !matches {
		data = "http://" + data
	}

	u, err := url.Parse(data)
	if err != nil {
		return
	} else if matches, _ := regexp.MatchString("[.]+", u.Host); !matches {
		err = errors.New("No valid domain in URL: " + u.Host)
		return
	}

	if matches, _ := regexp.MatchString("^[A-Za-z0-9.]*"+this.settings.RestrictDomain, u.Host); len(this.settings.RestrictDomain) > 0 && !matches {
		err = errors.New("Only URLs on " + this.settings.RestrictDomain + " domain allowed")
		return
	}

	entity = &ShortUrl{Destination: u.String(), Created: time.Now(), service: this}

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

	var url ShortUrl
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
