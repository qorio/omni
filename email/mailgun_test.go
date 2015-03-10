package email

import (
	. "gopkg.in/check.v1"
	"testing"
)

func TestEmailTests(t *testing.T) { TestingT(t) }

type EmailTests struct {
}

var _ = Suite(&EmailTests{})

// Database set up for circle_ci:
// psql> create role ubuntu login password 'password';
// psql> create database circle_ci with owner ubuntu encoding 'UTF8';
func (suite *EmailTests) SetUpSuite(c *C) {
}

func (suite *EmailTests) TearDownSuite(c *C) {
}

func (suite *EmailTests) TestSendMessage(c *C) {
	t := &Mailgun{
		MailgunAccount: MailgunAccount{
			SmtpDomain: "mail.qor.io",
			ApiKey:     "key-c29c589f369ecd184e69b4df89cb149d",
		},

		From: func(*Message) Address {
			return Address("ops@qoriolabs.com")
		},
		BodyTemplate: func(*Message) string {
			return "Hello {{.Username}}, here is the link where you can reset your password: {{.Link}}."
		},
	}

	resp, err := t.Send(&Message{
		To:      Address("dchung+test@qoriolabs.com"),
		Subject: "TESTING - Password reset",
		Context: map[string]string{
			"Username": "David",
			"Link":     "http://yahoo.com",
		},
	})

	c.Log("Sent message", *resp, err)
	c.Assert(err, Equals, nil)

}
