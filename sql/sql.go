package sql

import (
	"database/sql"
	"errors"
	"fmt"
	"github.com/golang/glog"
)

type Schema struct {
	CreateTables       map[string]string
	CreateIndexes      []string
	PreparedStatements map[StatementKey]Statement
	statements         map[StatementKey]*sql.Stmt
}

type StatementKey int
type Statement struct {
	Query string
	Args  func(...interface{}) ([]interface{}, error)
}

func (this *Schema) Initialize(db *sql.DB) (err error) {
	if this.statements == nil {
		this.statements = make(map[StatementKey]*sql.Stmt, 0)
	}

	tx, err := db.Begin()
	if err != nil {
		return
	}

	for _, stmt := range this.CreateTables {
		if _, err := db.Exec(stmt); err != nil {
			return tx.Rollback()
		}
	}
	for _, stmt := range this.CreateIndexes {
		if _, err := db.Exec(stmt); err != nil {
			glog.Warningln(stmt, "err:", err)
		}
	}
	for key, p := range this.PreparedStatements {
		if s, err := db.Prepare(p.Query); err != nil {
			return err
		} else {
			this.statements[key] = s
		}
	}

	return tx.Commit()
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
	var err error
	for table, _ := range this.CreateTables {
		_, err = db.Exec("drop table " + table)
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
