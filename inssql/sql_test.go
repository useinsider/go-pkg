package inssql

import (
	"database/sql"
	"database/sql/driver"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

type captureDriver struct {
	mu      sync.Mutex
	lastDSN string
}

func (d *captureDriver) Open(name string) (driver.Conn, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.lastDSN = name
	return nil, errors.New("captureDriver: connections not supported")
}

func (d *captureDriver) dsn() string {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.lastDSN
}

var testDriver = &captureDriver{}

var registerOnce sync.Once

func registerTestDriver(t *testing.T) {
	t.Helper()
	registerOnce.Do(func() {
		sql.Register("inssql-test", testDriver)
	})
}

func TestNew(t *testing.T) {
	registerTestDriver(t)

	t.Run("it_should_return_error_for_unregistered_driver", func(t *testing.T) {
		db, err := New("no-such-driver", "u", "p", "h", "d")
		if err == nil {
			t.Fatal("New() error = nil, want unknown driver error")
		}
		if db != nil {
			t.Errorf("New() db = %v, want nil", db)
		}
	})

	t.Run("it_should_build_the_expected_dsn", func(t *testing.T) {
		db, err := New("inssql-test", "user", "pass", "tcp(dbhost:3306)", "dbname")
		if err != nil {
			t.Fatalf("New() error = %v", err)
		}
		defer db.Close()

		_ = db.Ping()

		want := "user:pass@tcp(dbhost:3306)/dbname?charset=utf8mb4&collation=utf8mb4_unicode_ci&parseTime=true"
		if got := testDriver.dsn(); got != want {
			t.Errorf("driver received DSN %q, want %q", got, want)
		}
	})
}

func TestInit(t *testing.T) {
	registerTestDriver(t)

	sqlClient = nil

	t.Run("it_should_propagate_open_error_and_stay_uninitialized", func(t *testing.T) {
		db, err := Init("no-such-driver", "u", "p", "h", "d")
		if err == nil {
			t.Fatal("Init() error = nil, want unknown driver error")
		}
		if db != nil {
			t.Errorf("Init() db = %v, want nil", db)
		}
	})

	t.Run("it_should_initialize_the_singleton", func(t *testing.T) {
		db, err := Init("inssql-test", "u", "p", "h", "d")
		if err != nil {
			t.Fatalf("Init() error = %v", err)
		}
		if db == nil {
			t.Fatal("Init() db = nil, want client")
		}
		if GetClient() != db {
			t.Error("GetClient() returned a different client than Init()")
		}
	})

	t.Run("it_should_return_cached_client_and_ignore_new_arguments", func(t *testing.T) {
		first := GetClient()
		db, err := Init("no-such-driver", "other", "other", "other", "other")
		if err != nil {
			t.Fatalf("Init() error = %v, want nil from cached client", err)
		}
		if db != first {
			t.Error("Init() second call returned a different client, want the cached singleton")
		}
	})
}

func TestGetClient(t *testing.T) {
	t.Run("it_should_return_current_singleton", func(t *testing.T) {
		if GetClient() != sqlClient {
			t.Error("GetClient() != sqlClient")
		}
	})
}

func TestMockSql(t *testing.T) {
	t.Run("it_should_return_usable_db_and_mock", func(t *testing.T) {
		db, mock := MockSql()
		defer db.Close()

		mock.ExpectQuery("SELECT 1").WillReturnRows(sqlmock.NewRows([]string{"1"}).AddRow(1))

		rows, err := db.Query("SELECT 1")
		if err != nil {
			t.Fatalf("Query() error = %v", err)
		}
		defer rows.Close()

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("unmet sqlmock expectations: %v", err)
		}
	})
}

func TestAnyTime_Match(t *testing.T) {
	t.Run("it_should_match_time_values", func(t *testing.T) {
		if !(AnyTime{}).Match(time.Now()) {
			t.Error("Match(time.Time) = false, want true")
		}
	})

	t.Run("it_should_reject_non_time_values", func(t *testing.T) {
		if (AnyTime{}).Match("2024-01-01") {
			t.Error("Match(string) = true, want false")
		}
	})
}
