package config

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"tixora/internal/models"
)

// createDatabaseIfNotExists connects to the MySQL server (without selecting a
// database) and creates the target database if it doesn't exist yet, so a
// fresh checkout can just `go run` without any manual setup.
//
// Managed providers (Railway, PlanetScale, RDS) pre-create the database and
// often don't grant CREATE DATABASE to the app user, so callers treat a
// failure here as a warning, not fatal - the database already exists.
func createDatabaseIfNotExists(cfg *Config) error {
	db, err := sql.Open("mysql", cfg.ServerDSN())
	if err != nil {
		return fmt.Errorf("failed to open server connection: %w", err)
	}
	defer db.Close()

	query := fmt.Sprintf(
		"CREATE DATABASE IF NOT EXISTS `%s` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci",
		cfg.DBName,
	)
	if _, err := db.Exec(query); err != nil {
		return fmt.Errorf("failed to create database %q: %w", cfg.DBName, err)
	}

	return nil
}

// InitDB connects to MySQL via GORM, configures the connection pool,
// and auto-migrates all application models.
func InitDB(cfg *Config) (*gorm.DB, error) {
	if err := createDatabaseIfNotExists(cfg); err != nil {
		log.Printf("warning: could not ensure database %q exists, assuming it already does: %v", cfg.DBName, err)
	}

	db, err := gorm.Open(mysql.Open(cfg.DatabaseDSN()), &gorm.Config{
		Logger: logger.Default.LogMode(gormLogLevel(cfg)),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	sqlDB.SetMaxIdleConns(cfg.DBMaxIdleConns)
	sqlDB.SetMaxOpenConns(cfg.DBMaxOpenConns)
	sqlDB.SetConnMaxLifetime(time.Hour)

	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	if err := db.AutoMigrate(
		&models.User{},
		&models.Admin{},
		&models.RefreshToken{},
		&models.Category{},
		&models.File{},
		&models.Event{},
		&models.Order{},
		&models.Payment{},
	); err != nil {
		return nil, fmt.Errorf("failed to auto migrate database: %w", err)
	}

	return db, nil
}

func gormLogLevel(cfg *Config) logger.LogLevel {
	if cfg.Environment == "production" {
		return logger.Error
	}
	return logger.Info
}
