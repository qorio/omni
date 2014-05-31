package shorty

import (
	"errors"
	"github.com/golang/glog"
	"github.com/qorio/omni/http"
	"net/url"
	"regexp"
	"strconv"
	"time"
)

func (this *RoutingRule) Validate() (err error) {
	if len(this.Destination) > 0 {
		if _, err = url.Parse(this.Destination); err != nil {
			return
		}
	}
	matching := 0
	if c, err := regexp.Compile(this.MatchPlatform); err != nil {
		return errors.New("Bad platform regex " + this.MatchPlatform)
	} else if c != nil {
		matching++
	}
	if c, err := regexp.Compile(this.MatchOS); err != nil {
		return errors.New("Bad os regex " + this.MatchOS)
	} else if c != nil {
		matching++
	}
	if c, err := regexp.Compile(this.MatchMake); err != nil {
		return errors.New("Bad make regex " + this.MatchMake)
	} else if c != nil {
		matching++
	}
	if c, err := regexp.Compile(this.MatchBrowser); err != nil {
		return errors.New("Bad browser regex " + this.MatchBrowser)
	} else if c != nil {
		matching++
	}
	// Must have 1 or more matching regexp
	if matching == 0 {
		err = errors.New("bad-routing-rule:no matching regexp")
		return
	}
	return
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
	if len(this.MatchInstalled) > 0 && this.AppUrlScheme != "" {
		expect |= 1 << 6
		uuid, _ := cookies.GetPlainString(uuidCookieKey)
		_, found, err := service.FindInstall(UrlScheme(this.AppUrlScheme), UUID(uuid))
		glog.Infoln("checking install", uuid, this.AppUrlScheme, found, err)
		if matches, _ := regexp.MatchString(this.MatchInstalled, strconv.FormatBool(found)); matches {
			actual |= 1 << 6
		}
	}
	if this.MatchNoAppOpenInXDays > -1 && this.AppUrlScheme != "" {
		expect |= 1 << 7
		uuid, _ := cookies.GetPlainString(uuidCookieKey)
		appOpen, found, err := service.FindAppOpen(UrlScheme(this.AppUrlScheme), UUID(uuid))
		glog.Infoln("checking app-open", uuid, this.AppUrlScheme, found, appOpen, err)
		if !found || time.Now().Unix()-appOpen.Timestamp >= this.MatchNoAppOpenInXDays*24*60*60 {
			actual |= 1 << 7
		}
	}
	// By the time we get here, we have done a match all
	return actual == expect && expect > 0
}
