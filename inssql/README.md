# SQL Package


This is a simple mockable wrapper for SQL databases. You can help extend add other sql clients here as well.

## Usage in Apps
```go
package main

import "github.com/useinsider/go-pkg/inssql"

func main() {
	sqlClient, err := inssql.Init("mysql", "root", "root", "localhost", "demo")
	if err != nil {
		panic(err)
	}

	err := sqlClient.Ping()
	if err != nil {
		panic(err)
	}
}
```

## Usage in Tests

```go
package main

import (
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/useinsider/go-pkg/inssql"
	"gorm.io/gorm"
	"testing"
)

type Data struct {
	gorm.Model
	ID   int
	Name string
}

func Test(t *testing.T) {
	gormDB, sqlMock := inssql.MockGorm()

	sqlQuery := "INSERT INTO `data` (`created_at`,`updated_at`,`deleted_at`,`name`) VALUES (?,?,?,?)"

	sqlMock.ExpectBegin()
	sqlMock.
		ExpectExec(sqlQuery).
		WithArgs(
			inssql.AnyTime{}, inssql.AnyTime{}, nil, "some_name",
		).
		WillReturnResult(sqlmock.NewResult(1, 1))
	sqlMock.ExpectCommit()

	err := insert(gormDB, Data{Name: "some_name"})
	assert.NoError(t, err)

	err = sqlMock.ExpectationsWereMet()
	assert.NoError(t, err)
}

func insert(gormDB *gorm.DB, data Data) error {
	return gormDB.Create(&data).Error
}
```
