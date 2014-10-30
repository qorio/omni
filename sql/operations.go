package sql

import (
	"database/sql"
	"encoding/json"
	"errors"
)

var (
	ErrNotFound = errors.New("not-found")
	ErrNoChange = errors.New("no-change")
)

func (this *Schema) Upsert(db *sql.DB, update, insert StatementKey, args ...interface{}) error {
	// Do update first...
	result, err := this.Exec(db, update, args...)
	if err != nil {
		return err
	}
	updated, err := result.RowsAffected()
	if err != nil || updated == 0 {
		// try insert
		result, err = this.Exec(db, insert, args...)
		if err != nil {
			return err
		}

		inserted, err := result.RowsAffected()
		if err != nil {
			return err
		}
		if inserted == 0 {
			return errors.New("not-saved")
		}
	}
	return nil
}

func (this *Schema) Delete(db *sql.DB, delete StatementKey, args ...interface{}) error {
	result, err := this.Exec(db, delete, args...)
	if err != nil {
		return err
	}
	deleted, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if deleted == 0 {
		return ErrNoChange
	}
	return nil
}

// Assumes that a single result and a version value is in the result set, in order
// The result can be unrmshaled from json string from the db result.
type Options struct {
	Found         interface{}
	Version       interface{}
	NotFoundError error
}

func (this *Schema) GetOne(db *sql.DB, get StatementKey, opt *Options, args ...interface{}) error {
	row, err := this.QueryRow(db, get, args...)
	if err != nil {
		return err
	}

	buff := ""
	if opt != nil && opt.Version != nil {
		err = row.Scan(&buff, opt.Version)
	} else {
		err = row.Scan(&buff)
	}

	switch {
	case err == sql.ErrNoRows:
		if opt != nil && opt.NotFoundError != nil {
			return opt.NotFoundError
		} else {
			return ErrNotFound
		}
	case err != nil:
		return err
	}
	if opt != nil && opt.Found != nil {
		err = json.Unmarshal([]byte(buff), opt.Found)
	}
	return err
}
