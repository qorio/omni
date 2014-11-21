package sql

import (
	"database/sql"
	"errors"
	"fmt"
	"github.com/golang/glog"
	_ "github.com/lib/pq"
	"sync"
)

func NewPostgres() *Postgres {
	return &Postgres{
		User:            "ubuntu",
		Db:              "circle_test",
		Ssl:             false,
		DoCreateSchemas: true,
		DoUpdateSchemas: false,
	}
}

type Postgres struct {
	Host            string
	Port            int
	Db              string
	User            string
	Password        string
	Schemas         []*Schema
	DoCreateSchemas bool
	DoUpdateSchemas bool

	Ssl  bool
	conn *sql.DB
}

var (
	initialize_system sync.Once
)

func (this *Postgres) Conn() *sql.DB {
	return this.conn
}

func (this *Postgres) Open() error {
	glog.Infoln("Postgres connection string:", this.connection_string())

	db, err := sql.Open(string(POSTGRES), this.connection_string())
	if err != nil {
		return err
	}

	this.conn = db
	glog.Infoln("Connected:", this.conn)
	initialize_system.Do(func() {
		// bootstrap the system schema
		err1 := postgres_schema.Initialize(db)
		glog.Infoln("Initialized system schema:", err1)
		if err1 != nil {
			panic(err1)
		}
		err2 := postgres_schema.PrepareStatements(db)
		glog.Infoln("Prepared statements:", err2)
		if err2 != nil {
			panic(err2)
		}
	})
	err = sync_schemas(this.conn, this.Schemas, this.DoCreateSchemas, this.DoUpdateSchemas)
	if err != nil {
		return err
	}
	// prepare statements
	for _, s := range this.Schemas {
		err = s.PrepareStatements(this.conn)
		if err != nil {
			return err
		}
	}
	return nil
}

func (this *Postgres) Upsert(schema *Schema, update, insert StatementKey, args ...interface{}) error {
	return schema.Upsert(this.conn, update, insert, args...)
}

func (this *Postgres) Delete(schema *Schema, delete StatementKey, args ...interface{}) error {
	return schema.Delete(this.conn, delete, args...)
}

func (this *Postgres) GetOne(schema *Schema, get StatementKey, opt *Options, args ...interface{}) error {
	return schema.GetOne(this.conn, get, opt, args...)
}

func (this *Postgres) Close() error {
	if this.conn == nil {
		return ErrNotConnected
	}
	return this.conn.Close()
}

func (this *Postgres) DropAll() error {
	// This just drops everything... dangerous!
	for _, s := range this.Schemas {
		err := s.DropTables(this.conn)
		if err != nil {
			return err
		}
	}
	return nil
}

func (this *Postgres) TruncateAll() error {
	for _, s := range this.Schemas {
		for t, _ := range s.CreateTables {
			glog.Infoln("Truncating", t)
			_, err := this.conn.Exec("delete from " + t + " where true")
			if err != nil {
				return err
			}
		}
	}
	return nil
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
    commit_hash varchar null
)
		`,
	},
	PreparedStatements: map[StatementKey]Statement{
		kSelectVersionInfoBySchemaName: Statement{
			Query: `
select version, commit_hash from system_schema_versions
where schema_name=$1
`},
		kDeleteVersionInfo: Statement{
			Query: `
delete from system_schema_versions where schema_name=$1
`,
			Args: func(args ...interface{}) ([]interface{}, error) {
				if len(args) != 1 {
					return nil, errors.New("args-mismatch")
				}
				schema_name, ok := args[0].(string)
				if !ok {
					return nil, errors.New("bad-schema-name")
				}
				return []interface{}{
					schema_name,
				}, nil
			},
		},
		kInsertVersionInfo: Statement{
			Query: `
insert into system_schema_versions (schema_name, version, commit_hash)
values ($1, $2, $3)
`,
			Args: func(args ...interface{}) ([]interface{}, error) {
				if len(args) != 3 {
					return nil, errors.New("args-mismatch")
				}
				schema_name, ok := args[0].(string)
				if !ok {
					return nil, errors.New("bad-schema-name")
				}
				version, ok := args[1].(int)
				if !ok {
					return nil, errors.New("bad-version")
				}
				commit_hash, ok := args[2].(string)
				if !ok {
					return nil, errors.New("bad-commit-hash")
				}
				return []interface{}{
					schema_name,
					version,
					commit_hash,
				}, nil
			},
		},
		kUpdateVersionInfo: Statement{
			Query: `
update system_schema_versions
set version=$1, commit_hash=$2
where schema_name=$3
`,
			Args: func(args ...interface{}) ([]interface{}, error) {
				if len(args) != 3 {
					return nil, errors.New("args-mismatch")
				}
				schema_name, ok := args[0].(string)
				if !ok {
					return nil, errors.New("bad-schema-name")
				}
				version, ok := args[1].(int)
				if !ok {
					return nil, errors.New("bad-version")
				}
				commit_hash, ok := args[2].(string)
				if !ok {
					return nil, errors.New("bad-commit-hash")
				}
				return []interface{}{
					version,
					commit_hash,
					schema_name,
				}, nil
			},
		},
	},
}
