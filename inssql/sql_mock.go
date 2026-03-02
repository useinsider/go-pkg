package inssql

import (
	"database/sql"
	"database/sql/driver"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func MockSql() (*sql.DB, sqlmock.Sqlmock) {
	sqlDB, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	if err != nil {
		panic(err)
	}

	return sqlDB, mock
}

type AnyTime struct{}

func (a AnyTime) Match(v driver.Value) bool {
	_, ok := v.(time.Time)
	return ok
}
