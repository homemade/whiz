package sqlbuilder

import (
	"database/sql"
	"fmt"
)

type DriverSpecificSQL struct {
	MySQL    string
	Postgres string
}

func (s DriverSpecificSQL) Switch(db *sql.DB) (string, error) {
	d := fmt.Sprintf("%T", db.Driver())
	switch d {
	case "*mysql.MySQLDriver":
		return s.MySQL, nil
	case "*pq.Driver":
		return s.Postgres, nil
	default:
		return "", fmt.Errorf("unsupported database driver %s", d)
	}
}
