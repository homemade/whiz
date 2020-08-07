package publishers

import (
	"github.com/go-sql-driver/mysql"
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
		if driverErr.Number == 1062 { // Duplicate entry
			return true
		}
	}
	return false
	// TODO add Postgres support
}
