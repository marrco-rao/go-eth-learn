package database

import (
	"errors"
	"fmt"
	"path/filepath"

	"gorm.io/driver/mysql"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"ntf-auction-backend/internal/config"
)

func Open(cfg config.DatabaseConfig) (*gorm.DB, error) {
	switch cfg.Driver {
	case "sqlite", "":
		sqlitePath := cfg.SQLitePath
		if sqlitePath == "" {
			sqlitePath = filepath.Join("data", "app.db")
		}
		return gorm.Open(sqlite.Open(sqlitePath), &gorm.Config{})
	case "mysql":
		if cfg.MySQLDSN == "" {
			return nil, errors.New("mysql_dsn is required when database.driver is mysql")
		}
		return gorm.Open(mysql.Open(cfg.MySQLDSN), &gorm.Config{})
	default:
		return nil, fmt.Errorf("unsupported database driver: %s", cfg.Driver)
	}
}
