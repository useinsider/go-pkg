package insgorm

import (
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestNewGorm(t *testing.T) {
	t.Run("it_should_return_error_when_version_query_fails", func(t *testing.T) {
		// A sqlmock with no expectations rejects the driver's SELECT VERSION()
		// probe, which surfaces as a gorm.Open error.
		sqlDB, _, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
		if err != nil {
			t.Fatalf("sqlmock.New() error = %v", err)
		}
		defer sqlDB.Close()

		// Note: gorm.Open returns a non-nil *gorm.DB alongside the error, and
		// NewGorm passes both through unchanged — only the error is asserted.
		_, err = NewGorm(sqlDB)
		if err == nil {
			t.Fatal("NewGorm() error = nil, want version query failure")
		}
	})

	t.Run("it_should_wrap_existing_connection", func(t *testing.T) {
		sqlDB, m, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
		if err != nil {
			t.Fatalf("sqlmock.New() error = %v", err)
		}
		m.ExpectQuery("SELECT VERSION()").
			WillReturnRows(sqlmock.NewRows([]string{"VERSION()"}).AddRow("8.0.30"))

		gormDB, err := NewGorm(sqlDB)
		if err != nil {
			t.Fatalf("NewGorm() error = %v", err)
		}
		if gormDB == nil {
			t.Fatal("NewGorm() db = nil, want gorm client")
		}
	})
}

func TestWrapWithGorm(t *testing.T) {
	// The package keeps a singleton *gorm.DB; run the subtests in order
	// against a reset singleton.
	gormClient = nil

	t.Run("it_should_initialize_the_singleton", func(t *testing.T) {
		sqlDB, m, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
		if err != nil {
			t.Fatalf("sqlmock.New() error = %v", err)
		}
		m.ExpectQuery("SELECT VERSION()").
			WillReturnRows(sqlmock.NewRows([]string{"VERSION()"}).AddRow("8.0.30"))

		gormDB, err := WrapWithGorm(sqlDB)
		if err != nil {
			t.Fatalf("WrapWithGorm() error = %v", err)
		}
		if gormDB == nil {
			t.Fatal("WrapWithGorm() db = nil, want gorm client")
		}
		if GetGormClient() != gormDB {
			t.Error("GetGormClient() returned a different client than WrapWithGorm()")
		}
	})

	t.Run("it_should_return_cached_client_on_second_call", func(t *testing.T) {
		first := GetGormClient()

		// No sqlmock expectations here: a cached return must not touch the DB.
		sqlDB, _, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
		if err != nil {
			t.Fatalf("sqlmock.New() error = %v", err)
		}
		defer sqlDB.Close()

		gormDB, err := WrapWithGorm(sqlDB)
		if err != nil {
			t.Fatalf("WrapWithGorm() error = %v", err)
		}
		if gormDB != first {
			t.Error("WrapWithGorm() second call returned a new client, want the cached singleton")
		}
	})
}

func TestMockGorm(t *testing.T) {
	t.Run("it_should_return_usable_gorm_and_mock", func(t *testing.T) {
		gormDB, mock := MockGorm()
		if gormDB == nil {
			t.Fatal("MockGorm() db = nil")
		}

		mock.ExpectQuery("SELECT 1").WillReturnRows(sqlmock.NewRows([]string{"1"}).AddRow(1))

		var n int
		if err := gormDB.Raw("SELECT 1").Scan(&n).Error; err != nil {
			t.Fatalf("Raw().Scan() error = %v", err)
		}
		if n != 1 {
			t.Errorf("scanned %d, want 1", n)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("unmet sqlmock expectations: %v", err)
		}
	})
}
