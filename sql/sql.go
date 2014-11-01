package sql

import (
	"database/sql"
	"errors"
	"fmt"
	"github.com/golang/glog"
)

type Platform string

const (
	POSTGRES Platform = Platform("postgres")
)

var (
	ErrNotConnected   = errors.New("not-connected")
	ErrSchemaMismatch = errors.New("schemas-mismatch")
	ErrNotFound       = errors.New("not-found")
	ErrNoChange       = errors.New("no-change")

	platform_schemas = make(map[Platform]*Schema, 0)
)

func init() {
	platform_schemas[POSTGRES] = postgres_schema
}

const (
	kSelectVersionInfoBySchemaName StatementKey = iota
)

func sync_schemas(db *sql.DB, schemas []*Schema, create, update bool) error {
	if db == nil {
		return ErrNotConnected
	}

	creates := []*Schema{}
	updates := []*Schema{}

	for _, schema := range schemas {
		version, hash, err := schema.CurrentVersion(db)
		glog.Infoln(schema.Name, schema.Version, " -- curent db:", err, version, hash)
		switch {
		case err == ErrNotFound:
			glog.Warningln(schema.Name, " -- is not in db")
			creates = append(creates, schema)
		case version < schema.Version:
			glog.Warningln(schema.Name, " -- current db version", version, hash,
				"needs update to", schema.Version)
			updates = append(updates, schema)
		case version >= schema.Version:
			glog.Infoln(schema.Name, " -- current db version", version, hash,
				"is newer than", schema.Version)
		case err != nil:
			return err
		}
	}

	if len(creates) > 0 {
		if create {
			for _, s := range creates {
				err := s.Initialize(db)
				if err != nil {
					return nil
				}
			}
		} else {
			// Dump out the create tables to stdout
			for _, s := range creates {
				for t, ct := range s.CreateTables {
					fmt.Println("-- Table", t)
					fmt.Println(ct)
					fmt.Println()
				}
				for i, index := range s.CreateIndexes {
					fmt.Println("-- Index", i)
					fmt.Println(index)
					fmt.Println()
				}
			}
		}
	}
	if len(updates) > 0 {
		if update {
			for _, s := range updates {
				err := s.Update(db)
				if err != nil {
					return nil
				}
			}
		} else {
			// Dump out the create tables to stdout
			for _, s := range updates {
				for t, ct := range s.AlterTables {
					fmt.Println("-- Table", t)
					fmt.Println(ct)
					fmt.Println()
				}
				for i, index := range s.UpdateIndexes {
					fmt.Println("-- Update index", i)
					fmt.Println(index)
					fmt.Println()
				}
			}
		}
	}
	// Then crash
	if len(creates) > 0 || len(updates) > 0 {
		panic(ErrSchemaMismatch)
	}
	return nil
}
