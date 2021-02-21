package sqlschema

import (
	"database/sql"
	"fmt"

	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func Migrate(db *sql.DB, scripts []SQLScript) error {
	if len(scripts) > 0 {
		migrations, err := buildGormMigrations(db, scripts)
		if err == nil {
			return runGormMigrations(db, migrations)
		}
	}
	return nil
}

func buildGormMigrations(db *sql.DB, scripts []SQLScript) ([]*gormigrate.Migration, error) {
	var result []*gormigrate.Migration
	for _, s := range scripts {
		id := s.ID
		sql, err := s.DriverSpecificSQL.Switch(db)
		if err != nil {
			return result, fmt.Errorf("failed to read driver specific sql schema script %s %v", id, err)
		}
		result = append(result, &gormigrate.Migration{
			ID: id,
			Migrate: func(tx *gorm.DB) error {
				if err := tx.Exec(sql).Error; err != nil {
					return fmt.Errorf("failed to run migration for sql schema script %s %v", id, err)
				}
				return nil
			},
			Rollback: func(tx *gorm.DB) error {
				return nil
			},
		})
	}
	return result, nil
}

func runGormMigrations(db *sql.DB, migrations []*gormigrate.Migration) error {

	var gormDB *gorm.DB
	var err error

	d := fmt.Sprintf("%T", db.Driver())
	switch d {
	case "*mysql.MySQLDriver":
		gormDB, err = gorm.Open(mysql.New(mysql.Config{Conn: db}), &gorm.Config{})
	case "*pq.Driver":
		gormDB, err = gorm.Open(postgres.New(postgres.Config{Conn: db}), &gorm.Config{})
	default:
		err = fmt.Errorf("unsupported database driver %s", d)
	}
	if err != nil {
		return fmt.Errorf("failed to create a database connection using gorm %v", err)
	}

	options := gormigrate.DefaultOptions
	options.TableName = "eventz_schema_migrations"
	return gormigrate.New(gormDB, options, migrations).Migrate()

}
