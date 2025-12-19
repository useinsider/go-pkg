package inssql

import (
	"database/sql"
	"fmt"
)

var sqlClient *sql.DB

func Init(Driver, DBUser, DBPassword, DBHost, DBName string) (*sql.DB, error) {
	if sqlClient != nil {
		return sqlClient, nil
	}

	var err error
	sqlClient, err = New(Driver, DBUser, DBPassword, DBHost, DBName)

	return sqlClient, err
}

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

func GetClient() *sql.DB {
	return sqlClient
}
