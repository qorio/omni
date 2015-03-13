package email

import (
	"bytes"
	"encoding/json"
	"errors"
	"github.com/golang/glog"
	"io/ioutil"
	"net/http"
	"net/url"
	"text/template"
	"time"
)

/*
curl -X POST https://api.mailgun.net/v2/mail.qor.io/messages \
--data-urlencode 'from=dchung@qoriolabs.com'  \
--data-urlencode 'to=davidc616@gmail.com' \
--data-urlencode 'to=david@blinker.com' \
--data-urlencode 'subject=Hello' \
--data-urlencode 'text=Testing some Mailgun awesomness!'
-u 'api:key-c29c589f369ecd184e69b4df89cb149d' \
*/
const (
	MailgunMessageEndpoint = "https://api.mailgun.net/v2/{{.SmtpDomain}}/messages"
	MailgunTimeFormat      = time.RFC1123Z //"Mon, 02 Jan 2006 15:04:05 -0700"
)

var (
	MailgunEndpointTemplate *template.Template
)

func init() {
	MailgunEndpointTemplate = template.Must(template.New("twilio-endpoint").Parse(MailgunMessageEndpoint))
}

type Address string

func (email Address) IsZero() bool {
	return "" == string(email)
}

func (email Address) UnmarshalJSON(d []byte) error {
	email = Address(string(d))
	return nil
}

type MailgunAccount struct {
	SmtpDomain string `json:"smtp_domain"`
	ApiKey     string `json:"api_key"`
}

type Mailgun struct {
	MailgunAccount

	From         func(*Message) Address
	BodyTemplate func(*Message) string
	ContentType  string
}

type Message struct {
	To      Address     `json:"to"`
	Subject string      `json:"subject"`
	Context interface{} `json:"context"` // For apply template
}

type Response struct {
	Message string `json:"message"`
	Id      string `json:"id"`
}

var (
	ErrNoFrom         = errors.New("error-from-missing")
	ErrNoTo           = errors.New("error-to-missing")
	ErrNoBodyTemplate = errors.New("error-no-body-template")
)

func (this *Mailgun) Send(message *Message) (*Response, error) {
	if this.BodyTemplate == nil {
		return nil, ErrNoBodyTemplate
	}
	if this.From == nil {
		return nil, ErrNoFrom
	}
	if message.To.IsZero() {
		return nil, ErrNoTo
	}
	bt, err := template.New("email-body").Parse(this.BodyTemplate(message))
	if err != nil {
		return nil, err
	}
	var buff bytes.Buffer
	err = bt.Execute(&buff, message.Context)
	if err != nil {
		return nil, err
	}

	var endpoint bytes.Buffer
	MailgunEndpointTemplate.Execute(&endpoint, this.MailgunAccount)

	client := &http.Client{}

	data := url.Values{}
	data.Set("from", string(this.From(message)))
	data.Set("to", string(message.To))
	data.Set("subject", string(message.Subject))

	switch this.ContentType {
	case "text/plain":
		data.Set("text", buff.String())
	case "text/html":
		data.Set("html", buff.String())
	default:
		data.Set("text", buff.String())
	}
	post, err := http.NewRequest("POST", endpoint.String(), bytes.NewBufferString(data.Encode()))
	if err != nil {
		return nil, err
	}
	post.SetBasicAuth("api", this.ApiKey)
	post.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	resp, err := client.Do(post)
	if err != nil {
		return nil, err
	}
	content, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()

	glog.V(200).Infoln("Email: Response=", string(content))

	r := new(Response)
	err = json.Unmarshal(content, r)
	return r, err
}
