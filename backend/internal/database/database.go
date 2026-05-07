package database

import (
	"fmt"
	"os"
	"strings"

	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Open returns a GORM DB configured from DATABASE_* environment variables.
func Open() (*gorm.DB, error) {
	cfg := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	}

	driver := strings.ToLower(strings.TrimSpace(os.Getenv("DATABASE_DRIVER")))
	pgURL := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	sqlitePath := strings.TrimSpace(os.Getenv("DATABASE"))

	switch driver {
	case "", "auto":
		if pgURL != "" {
			driver = "postgres"
		} else {
			driver = "sqlite"
		}
	case "postgresql":
		driver = "postgres"
	}

	switch driver {
	case "postgres":
		dsn := pgURL
		if dsn == "" {
			dsn = sqlitePath
		}
		if dsn == "" {
			return nil, fmt.Errorf("postgres selected but DATABASE_URL and DATABASE are empty")
		}
		return gorm.Open(postgres.Open(dsn), cfg)
	case "sqlite":
		path := sqlitePath
		if path == "" {
			path = "gelift.db"
		}
		return gorm.Open(sqlite.Open(path), cfg)
	default:
		return nil, fmt.Errorf("unknown DATABASE_DRIVER %q (use sqlite, postgres, or auto)", driver)
	}
}
