package sql

import (
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"github.com/golang/glog"
	_ "github.com/lib/pq"
	"sync"
)

var (
	maxOpenConns = flag.Int("sql_max_open_conn", 0, "Max number of open connections, 0 is no limit")
	maxIdleConns = flag.Int("sql_max_idle_conn", 10, "Max number of open connections")
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

	Ssl         bool
	conn        *sql.DB
	conn_string string
}

var (
	initialize_system sync.Once
)

var (
	mutex1                    sync.Mutex
	sync_by_connection_string = make(map[string]*sync.Once, 0)
	conn_by_connection_string = make(map[string]*sql.DB, 0)
)

func (this *Postgres) Conn() *sql.DB {
	return this.conn
}

func (this *Postgres) Open() error {

	conn_string := this.connection_string()
	this.conn_string = conn_string

	glog.Infoln("Postgres connection string:", conn_string)

	mutex1.Lock()
	once, has := sync_by_connection_string[conn_string]
	if !has {
		once = &sync.Once{}
		sync_by_connection_string[conn_string] = once
	}
	mutex1.Unlock()

	once.Do(func() {
		glog.Infoln("Connecting to db:", conn_string)
		db, err := sql.Open(string(POSTGRES), conn_string)
		if err != nil {
			panic(err)
		}
		db.SetMaxIdleConns(*maxIdleConns)
		db.SetMaxOpenConns(*maxOpenConns)
		conn_by_connection_string[conn_string] = db
		glog.Infoln("Connected to db:", conn_string, "caching the connection handle")
	})

	this.conn, has = conn_by_connection_string[conn_string]
	if !has {
		panic(errors.New("error-no-initialized-db-connections"))
	}

	if this.conn == nil {
		panic(errors.New("error-db-connection-is-nil"))
	}
	glog.Infoln("Connected (PING):", this.conn.Ping())
	initialize_system.Do(func() {
		// bootstrap the system schema
		err1 := postgres_schema.Initialize(this.conn)
		glog.Infoln("Initialized system schema:", err1)
		if err1 != nil {
			panic(err1)
		}
		err2 := postgres_schema.PrepareStatements(this.conn)
		glog.Infoln("Prepared statements:", err2)
		if err2 != nil {
			panic(err2)
		}
	})
	err := sync_schemas(this.conn, this.Schemas, this.DoCreateSchemas, this.DoUpdateSchemas)
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

func (this *Postgres) Insert(schema *Schema, insert StatementKey, args ...interface{}) error {
	return schema.Insert(this.conn, insert, args...)
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

func (this *Postgres) GetAll(schema *Schema, get StatementKey, opt *Options, c Collect, args ...interface{}) error {
	return schema.GetAll(this.conn, get, opt, c, args...)
}

func (this *Postgres) Close() error {
	// HAKK -- this avoids db connections from getting closed during test tear downs.
	return nil
}

func (this *Postgres) ReallyClose() error {
	if this.conn == nil {
		return ErrNotConnected
	}
	err := this.conn.Close()
	// Remove the entry in the global maps by connection string.
	// This way, we will connect again when Open is called.
	mutex1.Lock()
	glog.Infoln("Removing sync / db handles for", this.conn_string)
	delete(sync_by_connection_string, this.conn_string)
	delete(conn_by_connection_string, this.conn_string)
	mutex1.Unlock()
	return err
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
    repo_url    varchar,
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
insert into system_schema_versions (schema_name, version, repo_url, commit_hash)
values ($1, $2, $3, $4)
`,
			Args: func(args ...interface{}) ([]interface{}, error) {
				if len(args) != 4 {
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
				repo, ok := args[2].(string)
				if !ok {
					return nil, errors.New("bad-repo")
				}
				commit_hash, ok := args[3].(string)
				if !ok {
					return nil, errors.New("bad-commit-hash")
				}
				return []interface{}{
					schema_name,
					version,
					repo,
					commit_hash,
				}, nil
			},
		},
		kUpdateVersionInfo: Statement{
			Query: `
update system_schema_versions
set version=$1, repo_url=$2, commit_hash=$3
where schema_name=$4
`,
			Args: func(args ...interface{}) ([]interface{}, error) {
				if len(args) != 4 {
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
				repo, ok := args[2].(string)
				if !ok {
					return nil, errors.New("bad-repo")
				}
				commit_hash, ok := args[3].(string)
				if !ok {
					return nil, errors.New("bad-commit-hash")
				}
				return []interface{}{
					version,
					repo,
					commit_hash,
					schema_name,
				}, nil
			},
		},
	},
}
