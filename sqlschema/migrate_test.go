package sqlschema

import (
	"database/sql"
	"testing"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
)

func TestMySQLMigrate(t *testing.T) {
	db, err := sql.Open("mysql", "root@(whiz_mysql)/whiz")
	if err == nil {
		err = Migrate(db, SQLScripts)
	}
	if err != nil {
		t.Error(err)
	}
}

func TestPostgresMigrate(t *testing.T) {
	db, err := sql.Open("postgres", "postgres://whiz:password123@whiz_postgres/whiz?sslmode=disable")
	if err == nil {
		err = Migrate(db, SQLScripts)
	}
	if err != nil {
		t.Error(err)
	}
}
