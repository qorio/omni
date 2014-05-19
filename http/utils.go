package http

import (
	"crypto/tls"
	"github.com/golang/glog"
	"io/ioutil"
	"net/http"
)

func FetchFromUrl(userAgent, url string) string {
	// don't check certificate for https
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}
	req, err := http.NewRequest("GET", url, nil)
	if userAgent != "" {
		req.Header.Add("User-Agent", userAgent)
	}

	resp, err := client.Do(req)
	if err != nil {
		glog.Warningln("cannot-fetch-content", err)
		return "<html>cannot-fetch-content:" + err.Error() + "</html>"
	}
	content, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		glog.Warningln("cannot-read-content", err)
		return "<html>cannot-read-content:" + err.Error() + "</html>"
	}
	return string(content)
}
