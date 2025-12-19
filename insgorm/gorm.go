package insgorm

import (
	"database/sql"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var gormClient *gorm.DB

func WrapWithGorm(sqlDB *sql.DB) (*gorm.DB, error) {
	if gormClient != nil {
		return gormClient, nil
	}

	var err error
	gormClient, err = NewGorm(sqlDB)

	return gormClient, err
}

func NewGorm(sqlDB *sql.DB) (*gorm.DB, error) {
	gormDB, err := gorm.Open(mysql.New(mysql.Config{
		Conn: sqlDB,
	}), &gorm.Config{})

	return gormDB, err
}

func GetGormClient() *gorm.DB {
	return gormClient
}
