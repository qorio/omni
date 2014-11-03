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
	suite.pg.DoCreateSchemas = false
	suite.pg.DoUpdateSchemas = false

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
	c.Assert(suite.pg.conn, Not(Equals), nil)
}

func (suite *SqlPostgresTests) TestCRUDSchemaVersion(c *C) {
	err := postgres_schema.Upsert(suite.pg.conn, kUpdateVersionInfo, kInsertVersionInfo, "test_schema", 0, "hash")
	c.Assert(err, Equals, nil)

	row, err := postgres_schema.QueryRow(suite.pg.conn, kSelectVersionInfoBySchemaName, "test_schema")
	c.Assert(err, Equals, nil)
	version, hash := -1, ""
	err = row.Scan(&version, &hash)
	c.Assert(err, Equals, nil)
	c.Assert(version, Equals, 0)
	c.Assert(hash, Equals, "hash")

	err = postgres_schema.Delete(suite.pg.conn, kDeleteVersionInfo, "test_schema")
	c.Assert(err, Equals, nil)

	row, err = postgres_schema.QueryRow(suite.pg.conn, kSelectVersionInfoBySchemaName, "test_schema")
	c.Assert(err, Equals, nil)
	err = row.Scan(&version, &hash)
	c.Assert(err, Not(Equals), nil)
}

var test_schema = &Schema{
	Platform:   POSTGRES,
	Name:       "test",
	Version:    1,
	CommitHash: "hashhhh",
	CreateTables: map[string]string{
		"testers": `
create table if not exists testers (
    last_name  varchar,
    first_name varchar,
    version    integer
)
		`,
	},
}

func (suite *SqlPostgresTests) TestInitPostgresNewSchema(c *C) {
	pg := NewPostgres()
	pg.Schemas = []*Schema{test_schema}

	err := pg.Open()
	c.Assert(err, Equals, nil)
	version, hash, err := test_schema.CurrentVersion(pg.conn)
	c.Assert(version, Equals, test_schema.Version)
	c.Assert(hash, Equals, test_schema.CommitHash)
}

func (suite *SqlPostgresTests) TestInitPostgresNewVersion(c *C) {
	pg := NewPostgres()
	pg.Schemas = []*Schema{test_schema}
	pg.DoUpdateSchemas = true
	test_schema.Version = 2
	test_schema.CommitHash = "new-hash"
	test_schema.AlterTables = map[string]string{
		"testers": "alter table testers add column age integer",
	}

	err := pg.Open()
	c.Assert(err, Equals, nil)
	version, hash, err := test_schema.CurrentVersion(pg.conn)
	c.Assert(version, Equals, test_schema.Version)
	c.Assert(hash, Equals, test_schema.CommitHash)

	// Do clean up so that the test is repeatable.
	test_schema.DropTables(pg.conn)
}
