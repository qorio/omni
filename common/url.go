package common

import (
	"net/url"
)

type Url url.URL

func (u *Url) UnmarshalJSON(s []byte) error {
	parsed, err := url.Parse(string(s))
	if err != nil {
		return err
	}

	u.Scheme = parsed.Scheme
	u.Opaque = parsed.Opaque
	u.User = parsed.User
	u.Host = parsed.Host
	u.Path = parsed.Path
	u.RawQuery = parsed.RawQuery
	u.Fragment = parsed.Fragment
	return nil
}

func (u *Url) MarshalJSON() ([]byte, error) {
	return []byte((*url.URL)(u).String()), nil
}
