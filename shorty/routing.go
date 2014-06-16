package shorty

import (
	"errors"
	"github.com/golang/glog"
	"github.com/qorio/omni/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

func (this *ShortUrl) MatchRule(service Shorty, userAgent *http.UserAgent,
	origin *http.RequestOrigin, cookies http.Cookies) (matchedRule *RoutingRule, err error) {

	defer func() {
		if matchedRule != nil {
			glog.V(10).Infoln("matched-rule", "id=", matchedRule.Id, "comment=", matchedRule.Comment,
				"origin=", origin, "referrer=", origin.Referrer, "userAgent=", userAgent)
		}
	}()

	for _, rule := range this.Rules {
		if match := rule.Match(this.service, userAgent, origin, cookies); match {
			matchedRule = &rule
			break
		}
	}
	if matchedRule == nil || matchedRule.Destination == "" {
		err = errors.New("not found")
	} else {
		for _, sub := range matchedRule.Special {
			matchSub := sub.Match(this.service, userAgent, origin, cookies)
			if matchSub {
				matchedRule = &sub
				return
			}
		}
	}
	return
}

func (this OnOff) IsOn() bool {
	return strings.ToLower(string(this)) == "on"
}

func (this *RoutingRule) Validate() (err error) {
	if len(this.Destination) > 0 {
		if _, err = url.Parse(this.Destination); err != nil {
			return
		}
	}
	matching := 0
	if c, err := regexp.Compile(string(this.MatchPlatform)); err != nil {
		return errors.New("Bad platform regex " + string(this.MatchPlatform))
	} else if c != nil {
		matching++
	}
	if c, err := regexp.Compile(string(this.MatchOS)); err != nil {
		return errors.New("Bad os regex " + string(this.MatchOS))
	} else if c != nil {
		matching++
	}
	if c, err := regexp.Compile(string(this.MatchMake)); err != nil {
		return errors.New("Bad make regex " + string(this.MatchMake))
	} else if c != nil {
		matching++
	}
	if c, err := regexp.Compile(string(this.MatchBrowser)); err != nil {
		return errors.New("Bad browser regex " + string(this.MatchBrowser))
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

// TODO - precompile the regexs and store them in the Routing rule
func (this *RoutingRule) Match(service Shorty, ua *http.UserAgent, origin *http.RequestOrigin, cookies http.Cookies) bool {
	// use bit mask to match
	var actual, expect int = 0, 0

	if len(string(this.MatchPlatform)) > 0 {
		expect |= 1 << 0
		if matches, _ := regexp.MatchString(string(this.MatchPlatform), ua.Platform); matches {
			actual |= 1 << 0
		}
	}
	if len(string(this.MatchOS)) > 0 {
		expect |= 1 << 1
		if matches, _ := regexp.MatchString(string(this.MatchOS), ua.OS); matches {
			actual |= 1 << 1
		}
	}
	if len(string(this.MatchMake)) > 0 {
		expect |= 1 << 2
		if matches, _ := regexp.MatchString(string(this.MatchMake), ua.Make); matches {
			actual |= 1 << 2
		}
	}
	if len(string(this.MatchBrowser)) > 0 {
		expect |= 1 << 3
		if matches, _ := regexp.MatchString(string(this.MatchBrowser), ua.Browser); matches {
			actual |= 1 << 3
		}
	}
	if this.MatchMobile.IsOn() {
		expect |= 1 << 4
		if ua.Mobile {
			actual |= 1 << 4
		}
	}
	if len(string(this.MatchReferrer)) > 0 {
		expect |= 1 << 5
		if matches, _ := regexp.MatchString(string(this.MatchReferrer), origin.Referrer); matches {
			actual |= 1 << 5
		}
	}
	if this.MatchNoAppOpenInXDays.IsOn() && this.AppUrlScheme != "" {
		expect |= 1 << 6
		uuid, _ := cookies.GetPlainString(uuidCookieKey)
		appOpen, found, _ := service.FindAppOpen(UrlScheme(this.AppUrlScheme), UUID(uuid))
		if !found || float64(time.Now().Unix()-appOpen.Timestamp) >= this.AppOpenTTLDays*24.*60.*60. {
			actual |= 1 << 6
		}
	}
	// By the time we get here, we have done a match all
	return actual == expect && expect > 0
}
