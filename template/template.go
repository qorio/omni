package template

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"text/template"
)

var (
	ErrNoSourceUrl = errors.New("error-no-source-url")
)

type Template struct {
	SourceUrl       string      `json:"source_url"`
	Context         interface{} `json:"-"`
	AppliedTemplate string      `json:"applied_template,omitempty"`
}

func (this *Template) Load() error {
	if this.SourceUrl == "" {
		return nil
	}

	body, _, err := UrlGet(this.SourceUrl)
	if err != nil {
		return err
	}
	str, err := applyTemplate(body, this.Context)
	if err != nil {
		return err
	}
	this.AppliedTemplate = str
	return nil
}

func (this *Template) Unmarshal(prototype interface{}) (err error) {
	if this.AppliedTemplate == "" {
		err = this.Load()
		if err != nil {
			return
		}
	}
	return json.Unmarshal([]byte(this.AppliedTemplate), prototype)
}

func applyTemplate(body string, context interface{}) (string, error) {
	if context == nil {
		return body, nil
	}

	t, err := template.New(body).Parse(body)
	if err != nil {
		return "", err
	}

	var buff bytes.Buffer
	if err := t.Execute(&buff, context); err != nil {
		return "", err
	} else {
		return buff.String(), nil
	}
}

func UrlGet(urlRef string) (body, contentType string, err error) {
	url, err := url.Parse(urlRef)
	if err != nil {
		return "", "", err
	}

	if url.Scheme == "file" {
		file := url.Path
		f, e := os.Open(file)
		if e != nil {
			err = e
			return
		}
		if buff, e := ioutil.ReadAll(f); e != nil {
			err = e
			return
		} else {
			body = string(buff)
			contentType = "text/plain"
		}
		return
	} else {
		// don't check certificate for https
		tr := &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
		client := &http.Client{Transport: tr}
		req, e := http.NewRequest("GET", url.String(), nil)
		if e != nil {
			err = e
			return
		}
		resp, e := client.Do(req)
		if e != nil {
			err = e
			return
		}
		defer resp.Body.Close()
		content, e := ioutil.ReadAll(resp.Body)
		if e != nil {
			err = e
			return
		}

		body = string(content)
		contentType = resp.Header.Get("Content-Type")
		return
	}
}
