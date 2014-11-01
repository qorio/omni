package sql

import (
	"database/sql"
	"fmt"
	"github.com/golang/glog"
	_ "github.com/lib/pq"
)

func NewPostgres() *Postgres {
	return &Postgres{
		User:     "ubuntu",
		Db:       "circle_test",
		Ssl:      false,
		DoCreate: true,
		DoUpdate: false,
	}
}

type Postgres struct {
	Host     string
	Port     int
	Db       string
	User     string
	Password string
	Schemas  []*Schema
	DoCreate bool
	DoUpdate bool

	Ssl bool
	db  *sql.DB
}

func (this *Postgres) Open() error {
	db, err := sql.Open(string(POSTGRES), this.connection_string())
	if err != nil {
		return err
	}
	this.db = db
	// bootstrap the system schema
	err = postgres_schema.Initialize(db)
	if err != nil {
		return err
	}
	glog.Infoln("Initialized system schema.")
	return sync_schemas(this.db, this.Schemas, this.DoCreate, this.DoUpdate)
}

func (this *Postgres) Close() error {
	if this.db == nil {
		return ErrNotConnected
	}
	return this.db.Close()
}

func (this *Postgres) connection_string() string {
	cs := fmt.Sprintf("user=%s dbname=%s", this.User, this.Db)
	if this.Password != "" {
		cs += fmt.Sprintf(" password='%s'", this.Password)
	}
	if this.Host != "" {
		cs += fmt.Sprintf(" host=%s", this.Host)
	}
	if this.Port > 0 {
		cs += fmt.Sprintf(" port=%d", this.Port)
	}
	if this.Ssl {
		cs += " sslmode=verify-full"
	} else {
		cs += " sslmode=disable"
	}
	return cs
}

var postgres_schema = &Schema{
	Platform: POSTGRES,
	Name:     "system",
	Version:  1,
	CreateTables: map[string]string{
		"schema_versions": `
create table if not exists system_schema_versions (
    schema_name varchar,
    version     integer,
    commit_hash varchar,
)
		`,
	},
	PreparedStatements: map[StatementKey]Statement{
		kSelectVersionInfoBySchemaName: Statement{
			Query: `
select version, commit_hash from system_schema_versions
where schema_name=$1
`},
	},
}
