package template

import (
	"github.com/qorio/omni/common"
	. "gopkg.in/check.v1"
	"testing"
)

func TestTemplateTests(t *testing.T) { TestingT(t) }

type TemplateTests struct {
}

var _ = Suite(&TemplateTests{})

func (suite *TemplateTests) SetUpSuite(c *C) {
}

func (suite *TemplateTests) TearDownSuite(c *C) {
}

type test struct {
	FirstName string     `json:"first_name"`
	LastName  string     `json:"last_name"`
	Url       common.Url `json:"url"`
}

func (suite *TemplateTests) TestFetchTemplateFromUrl(c *C) {

	t := Template{
		SourceUrl: "http://qorio.github.io/omni/test/templates/test1.json",
		Context: map[string]string{
			"FirstName": "david",
			"LastName":  "jones",
		},
	}

	err := t.Load()
	c.Assert(err, Equals, nil)
	c.Assert(t.AppliedTemplate, Not(Equals), "")
	c.Log(t.AppliedTemplate)

	tt := new(test)
	err2 := t.Unmarshal(tt)
	c.Assert(err2, Equals, nil)
	c.Log("parsed", tt.Url)
	c.Assert(tt.FirstName, Equals, "david")
	c.Assert(tt.LastName, Equals, "jones")
}
