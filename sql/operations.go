package sql

import (
	"database/sql"
	"encoding/json"
)

func (this *Schema) Insert(db *sql.DB, insert StatementKey, args ...interface{}) error {
	result, err := this.Exec(db, insert, args...)
	if err != nil {
		return err
	}
	inserted, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if inserted == 0 {
		return ErrNoChange
	}
	return nil
}

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
			return ErrNoChange
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
	Alloc         func() interface{}
	NotFoundError error
}

func (this *Schema) GetOne(db *sql.DB, get StatementKey, opt *Options, args ...interface{}) error {
	if opt == nil {
		return ErrOptIsNull
	}

	row, err := this.QueryRow(db, get, args...)
	if err != nil {
		return err
	}

	buff := ""
	if opt.Version != nil {
		err = row.Scan(&buff, opt.Version)
	} else {
		err = row.Scan(&buff)
	}

	switch {
	case err == sql.ErrNoRows:
		if opt.NotFoundError != nil {
			return opt.NotFoundError
		} else {
			return ErrNotFound
		}
	case err != nil:
		return err
	}
	if opt.Found != nil {
		err = json.Unmarshal([]byte(buff), opt.Found)
	}
	return err
}

// Returns false to stop
type Collect func(interface{}) bool

func (this *Schema) GetAll(db *sql.DB, get StatementKey, opt *Options, collect Collect, args ...interface{}) error {
	if opt == nil {
		return ErrOptIsNull
	}
	if collect == nil {
		return ErrNoCollect
	}

	rows, err := this.Query(db, get, args...)
	if err != nil {
		return err
	}

	for rows.Next() {
		buff := ""
		if opt.Version != nil {
			err = rows.Scan(&buff, opt.Version)
		} else {
			err = rows.Scan(&buff)
		}
		switch {
		case err == sql.ErrNoRows:
			break
		case err != nil:
			return err
		}
		if opt.Alloc != nil {
			obj := opt.Alloc()
			err = json.Unmarshal([]byte(buff), obj)
			if err != nil {
				return err
			}
			if !collect(obj) {
				return nil
			}
		}
	}
	return nil
}
