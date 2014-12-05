package sql

import (
	"database/sql"
	"errors"
	"fmt"
	"github.com/golang/glog"
	"github.com/qorio/omni/runtime"
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
	kInsertVersionInfo
	kUpdateVersionInfo
	kDeleteVersionInfo
)

func update_schema_version(db *sql.DB, schema *Schema) error {
	if schema.Platform == Platform("") || schema.Name == "" || schema.Version == 0 {
		panic("need-schema-information")
	}

	system, has := platform_schemas[schema.Platform]
	if !has {
		panic(errors.New("no-systems-versions"))
	}
	hash := schema.CommitHash
	if hash == "" {
		hash = runtime.BuildInfo().GetCommitHash()
	}
	repo := schema.RepoUrl
	if repo == "" {
		repo = runtime.BuildInfo().GetRepoUrl()
	}

	err := system.PrepareStatements(db)
	if err != nil {
		return err
	}
	return system.Upsert(db, kUpdateVersionInfo, kInsertVersionInfo, schema.Name, schema.Version, repo, hash)
}

func check_schema(s *Schema) {
	if s.Name == "" || string(s.Platform) == "" || s.Version == 0 {
		panic(errors.New("schema-missing-version-info"))
	}
}

func sync_schemas(db *sql.DB, schemas []*Schema, create, update bool) error {
	if db == nil {
		return ErrNotConnected
	}

	creates := []*Schema{}
	updates := []*Schema{}

	for _, schema := range schemas {

		check_schema(schema)

		version, hash, err := schema.CurrentVersion(db)
		glog.Infoln(schema.Name, schema.Version, " -- curent db:", err, version, hash)
		switch {
		case err == ErrNotFound:
			glog.Warningln(schema.Name, schema.Version, " -- is not in db")
			creates = append(creates, schema)
		case err != nil:
			return err
		case version < schema.Version:
			glog.Warningln(schema.Name, schema.Version, " -- current db version", version, hash,
				"needs update to", schema.Version)
			updates = append(updates, schema)
		case version >= schema.Version:
			glog.Infoln(schema.Name, schema.Version, " -- current db version", version, hash,
				"is newer than or equal to ", schema.Version, "--> no action.")
		}
	}

	if len(creates) > 0 {
		glog.Infoln("Tables to create:", len(creates))

		if create {
			for _, s := range creates {
				err := s.Initialize(db)
				if err != nil {
					return nil
				}
			}
		} else {
			// Dump out the create tables to stdout
			fmt.Println("-- RUN THE FOLLOWING SQL COMMANDS --")
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
			fmt.Println("-- RUN THE FOLLOWING SQL COMMANDS --")
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
	if (!create && len(creates) > 0) || (!update && len(updates) > 0) {
		panic(ErrSchemaMismatch)
	}
	return nil
}
