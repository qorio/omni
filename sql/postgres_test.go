package sql

import (
	. "gopkg.in/check.v1"
	"testing"
)

func TestSqlPostgres(t *testing.T) { TestingT(t) }

type SqlPostgresTests struct {
	pg *Postgres
}

var _ = Suite(&SqlPostgresTests{})

// Database set up for circle_ci:
// psql> create role ubuntu login password 'password';
// psql> create database circle_ci with owner ubuntu encoding 'UTF8';
func (suite *SqlPostgresTests) SetUpSuite(c *C) {
	var err error
	suite.pg = NewPostgres()
	suite.pg.DoCreate = false
	suite.pg.DoUpdate = false

	err = suite.pg.Open()
	if err != nil {
		panic(err)
	}
}

func (suite *SqlPostgresTests) TearDownSuite(c *C) {
	suite.pg.Close()
}

func (suite *SqlPostgresTests) TestConnectToDb(c *C) {
	c.Log("pgImpl=", suite.pg)
	c.Assert(suite.pg.db, Not(Equals), nil)
}
