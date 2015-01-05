package sql

import (
	"database/sql"
	"errors"
	"fmt"
	"github.com/golang/glog"
)

type Schema struct {
	Platform           Platform
	Name               string
	Version            int
	RepoUrl            string
	CommitHash         string
	CreateTables       map[string]string
	CreateIndexes      []string
	PreparedStatements map[StatementKey]Statement
	AlterTables        map[string]string
	UpdateIndexes      []string

	statements map[StatementKey]*sql.Stmt
}

type StatementKey int
type Statement struct {
	Query string
	Args  func(...interface{}) ([]interface{}, error)
}

var (
	ErrNoSystemSchema = errors.New("no-system-schema")
	ErrOptIsNull      = errors.New("options-is-null")
	ErrNoCollect      = errors.New("no-collect")
)

func (this *Schema) CurrentVersion(db *sql.DB) (int, string, error) {
	version := -1
	hash := ""

	// Get the system schema
	system, has := platform_schemas[this.Platform]
	if !has {
		return version, hash, ErrNoSystemSchema
	}

	row, err := system.QueryRow(db, kSelectVersionInfoBySchemaName, this.Name)
	if err != nil {
		return version, hash, err
	}
	err = row.Scan(&version, &hash)
	switch {
	case err == sql.ErrNoRows:
		return version, hash, ErrNotFound
	case err != nil:
		return version, hash, err
	}
	glog.Infoln("Checking", this.Name, this.Version, "but finds in db:", version, hash)
	return version, hash, err
}

func (this *Schema) Initialize(db *sql.DB) (err error) {
	if this.statements == nil {
		this.statements = make(map[StatementKey]*sql.Stmt, 0)
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}

	for _, stmt := range this.CreateTables {
		glog.V(40).Infoln(stmt)
		if _, err := db.Exec(stmt); err != nil {
			tx.Rollback()
			return err
		}
	}
	for _, stmt := range this.CreateIndexes {
		glog.V(40).Infoln(stmt)
		if _, err := db.Exec(stmt); err != nil {
			glog.Warningln(stmt, "err:", err)
		}
	}

	err = update_schema_version(db, this)
	if err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}

func (this *Schema) Update(db *sql.DB) (err error) {
	if this.statements == nil {
		this.statements = make(map[StatementKey]*sql.Stmt, 0)
	}

	tx, err := db.Begin()
	if err != nil {
		return
	}

	for _, stmt := range this.AlterTables {
		if _, err := db.Exec(stmt); err != nil {
			return tx.Rollback()
		}
	}
	for _, stmt := range this.UpdateIndexes {
		if _, err := db.Exec(stmt); err != nil {
			glog.Warningln(stmt, "err:", err)
		}
	}

	err = update_schema_version(db, this)
	if err != nil {
		tx.Rollback()
		return
	}
	return tx.Commit()
}

func (this *Schema) PrepareStatements(db *sql.DB) error {
	if this.statements == nil {
		this.statements = make(map[StatementKey]*sql.Stmt, 0)
	}

	for key, p := range this.PreparedStatements {
		if s, err := db.Prepare(p.Query); err != nil {
			return err
		} else {
			this.statements[key] = s
		}
	}
	return nil
}

func (this *Schema) Exec(db *sql.DB, key StatementKey, params ...interface{}) (sql.Result, error) {
	s, stmt, err := this.statement(key)
	if err != nil {
		return nil, err
	}
	args := params
	if s.Args != nil {
		args, err = s.Args(params...)
		if err != nil {
			return nil, err
		}
	}
	return stmt.Exec(args...)
}

func (this *Schema) Query(db *sql.DB, key StatementKey, params ...interface{}) (*sql.Rows, error) {
	s, stmt, err := this.statement(key)
	if err != nil {
		return nil, err
	}
	args := params
	if s.Args != nil {
		args, err = s.Args(params...)
		if err != nil {
			return nil, err
		}
	}
	return stmt.Query(args...)
}

func (this *Schema) QueryRow(db *sql.DB, key StatementKey, params ...interface{}) (*sql.Row, error) {
	s, stmt, err := this.statement(key)
	if err != nil {
		return nil, err
	}
	args := params
	if s.Args != nil {
		args, err = s.Args(params...)
		if err != nil {
			return nil, err
		}
	}
	return stmt.QueryRow(args...), nil
}

func (this *Schema) DropTables(db *sql.DB) error {
	system, has := platform_schemas[this.Platform]
	if !has {
		panic(errors.New("no-systems-versions"))
	}
	var err error
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	for table, _ := range this.CreateTables {
		_, err = db.Exec("drop table " + table)
		if err != nil {
			tx.Rollback()
			return err
		}
	}
	// remove from the systems info
	err = system.Delete(db, kDeleteVersionInfo, this.Name)
	if err != nil {
		tx.Rollback()
		return err
	} else {
		tx.Commit()
	}
	return err
}

func (this *Schema) statement(key StatementKey) (*Statement, *sql.Stmt, error) {
	stmt, has1 := this.statements[key]
	s, has2 := this.PreparedStatements[key]

	if !has1 || !has2 {
		return nil, nil, errors.New(fmt.Sprintf("no-statement-for-key: %s", key))
	}
	return &s, stmt, nil
}
