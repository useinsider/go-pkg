package inssql

import (
	"database/sql"
	"fmt"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// sqlClient is the singleton client created by InitSql function.
var sqlClient *sql.DB

// gormClient is the singleton client created by WrapWithGorm function.
var gormClient *gorm.DB

// Init creates a client pool for sql connections.
// Driver must be one of these https://golang.org/s/sqldrivers.
func Init(Driver, DBUser, DBPassword, DBHost, DBName string) (*sql.DB, error) {
	if sqlClient != nil {
		return sqlClient, nil
	}

	var err error
	sqlClient, err = New(Driver, DBUser, DBPassword, DBHost, DBName)

	return sqlClient, err
}

// New creates brand new sql client
func New(Driver string, DBUser string, DBPassword string, DBHost string, DBName string) (*sql.DB, error) {
	dsn := fmt.Sprintf(
		"%v:%v@%v/%v?charset=utf8mb4&collation=utf8mb4_unicode_ci&parseTime=true",
		DBUser,
		DBPassword,
		DBHost,
		DBName,
	)

	db, err := sql.Open(Driver, dsn)
	if err != nil {
		return nil, err
	}

	return db, err
}

// GetClient returns globally cached sqlClient.
func GetClient() *sql.DB {
	return sqlClient
}

// WrapWithGorm connection
func WrapWithGorm(sqlDB *sql.DB) (*gorm.DB, error) {
	if gormClient != nil {
		return gormClient, nil
	}

	var err error
	gormClient, err = NewGorm(sqlDB)

	return gormClient, err
}

// NewGorm wrap new sql client
func NewGorm(sqlDB *sql.DB) (*gorm.DB, error) {
	gormDB, err := gorm.Open(mysql.New(mysql.Config{
		Conn: sqlDB,
	}), &gorm.Config{})

	return gormDB, err
}

// GetGormClient returns globally cached gormClient.
func GetGormClient() *gorm.DB {
	return gormClient
}
