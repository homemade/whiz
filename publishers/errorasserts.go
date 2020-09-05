package publishers

import (
	"github.com/go-sql-driver/mysql"
	"github.com/lib/pq"
)

// ErrorAsserts defines the error assertions used by publishers
type ErrorAsserts interface {
	IsDuplicateDatabaseEntry(error) bool
}

var BuiltinErrorAsserts = builtinErrorAsserts{}

type builtinErrorAsserts struct{}

func (s builtinErrorAsserts) IsDuplicateDatabaseEntry(err error) bool {
	// MySQL
	if driverErr, ok := err.(*mysql.MySQLError); ok {
		if driverErr.Number == 1062 { // Duplicate entry for key
			return true
		}
	}
	// Postgres
	if driverErr, ok := err.(*pq.Error); ok {
		if driverErr.Code == "23505" { // Duplicate key value violates unique constraint
			return true
		}
	}
	return false
}
