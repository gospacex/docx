//go:build !wireinject

package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/gospacex/dbx/config"
	"github.com/gospacex/dbx/dbsql"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	sdkresource "go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"gorm.io/gorm"
)

const (
	cfgPath         = "mysql.yaml"
	jaegerBase      = "http://localhost:16686"
	serviceName     = "examples-dbxwirejaeger"
	jaegerWaitMax   = 5 * time.Second
	jaegerPollEvery = 200 * time.Millisecond
)

var (
	tpShutdown func(context.Context) error
	injector   *Injector
	dbxDB      *gorm.DB
	sqlDB      *sql.DB
)

func TestMain(m *testing.M) {
	ctx := context.Background()

	// 1. Load both mysql and tracing blocks via dbx's typed loader.
	//    config.LoadMySQL returns (*MySQLConfig, *TracingConfig, error).
	mysqlCfg, traceCfg, err := config.LoadMySQL(cfgPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "config.LoadMySQL %q: %v\n", cfgPath, err)
		os.Exit(1)
	}
	_ = mysqlCfg // reserved; dbsql.OpenPath reads it again internally

	// 2. Three-step OTel TracerProvider setup (caller owns TracerProvider;
	//    dbx.ExtractTracingAndApply is a documented no-op).
	exp, err := dbsql.CreateExporter(ctx, traceCfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "dbsql.CreateExporter: %v\n", err)
		os.Exit(1)
	}
	res, err := sdkresource.New(ctx,
		sdkresource.WithAttributes(
			semconv.ServiceName(traceCfg.Service),
		),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "sdkresource.New: %v\n", err)
		os.Exit(1)
	}
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp),
		sdktrace.WithResource(res),
	)
	otel.SetTracerProvider(tp)
	tpShutdown = tp.Shutdown

	// 3. Wire injection.
	inj, err := InitializeInjector(cfgPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "InitializeInjector: %v\n", err)
		os.Exit(1)
	}
	injector = inj

	// 4. Provide *gorm.DB via Provider.
	db, err := injector.DB.ProvideDB(cfgPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ProvideDB: %v\n", err)
		os.Exit(1)
	}
	dbxDB = db

	// 5. Hold the underlying *sql.DB so cleanup can Close() the pool AFTER
	//    tp.Shutdown flushes spans.
	sqlDBHolder, err := db.DB()
	if err != nil {
		fmt.Fprintf(os.Stderr, "db.DB(): %v\n", err)
		os.Exit(1)
	}
	sqlDB = sqlDBHolder

	// 6. Verify connectivity to legoB mysql (catches wrong host/credentials early).
	if err := sqlDB.PingContext(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "mysql ping: %v\n", err)
		os.Exit(1)
	}

	// 7. One-shot AutoMigrate so all 5 cases share the `users` table.
	if err := dbxDB.WithContext(ctx).AutoMigrate(&User{}); err != nil {
		fmt.Fprintf(os.Stderr, "AutoMigrate: %v\n", err)
		os.Exit(1)
	}

	code := m.Run()

	// Cleanup ORDER IS FIXED: flush spans BEFORE closing the connection pool.
	// If we Close() first, in-flight BatchSpanProcessor can fail to read
	// attribute values from a closed pool.
	tpShutdown(ctx)
	_ = sqlDB.Close()
	os.Exit(code)
}

// requireSpanReported polls jaeger /api/traces for an operation matching
// the given name within jaegerWaitMax. Fatal on timeout and prints the
// last response body so a future maintainer can debug.
func requireSpanReported(t *testing.T, operation string) {
	t.Helper()

	q := url.Values{}
	q.Set("service", serviceName)
	q.Set("operation", operation)
	q.Set("lookback", "1m")
	endpoint := fmt.Sprintf("%s/api/traces?%s", jaegerBase, q.Encode())

	deadline := time.Now().Add(jaegerWaitMax)
	var lastBody string
	for time.Now().Before(deadline) {
		resp, err := http.Get(endpoint) // #nosec G107 -- jaegerBase is localhost, intentional
		if err == nil {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			lastBody = string(body)
			var data struct {
				Data []struct {
					Spans []struct {
						OperationName string `json:"operationName"`
					} `json:"spans"`
				} `json:"data"`
			}
			if json.Unmarshal(body, &data) == nil {
				for _, trace := range data.Data {
					for _, span := range trace.Spans {
						if span.OperationName == operation {
							return
						}
					}
				}
			}
		}
		time.Sleep(jaegerPollEvery)
	}
	t.Fatalf("expected span operation=%q in service=%q within %s; last response: %s",
		operation, serviceName, jaegerWaitMax, lastBody)
}

// _ keeps attribute import alive across refactors; the canonical SDK uses
// attribute.KeyValue in WithAttributes — declared here so the import is
// never pruned by goimports when future code adds per-attribute values.
var _ = attribute.KeyValue{}

// -----------------------------------------------------------------------------
// 5 GORM CRUD test cases — each issues ONE GORM call (root span), asserts
// the corresponding span operation reaches jaeger, and cleans up its row.
// Mirrors the cachex example's 5-case pattern (String/Hash/Set/List/SortedSet).
// -----------------------------------------------------------------------------

func TestDbxWireJaeger_Create(t *testing.T) {
	ctx := context.Background()
	u := newTestUser()

	if err := dbxDB.WithContext(ctx).Create(&u).Error; err != nil {
		t.Fatalf("Create: %v", err)
	}
	if u.ID == 0 {
		t.Fatalf("expected auto-assigned ID, got 0")
	}
	t.Cleanup(func() {
		_ = dbxDB.WithContext(ctx).Delete(&User{}, u.ID).Error
	})

	requireSpanReported(t, "db.create")
}

func TestDbxWireJaeger_Get(t *testing.T) {
	ctx := context.Background()
	u := newTestUser()

	if err := dbxDB.WithContext(ctx).Create(&u).Error; err != nil {
		t.Fatalf("seed Create: %v", err)
	}
	t.Cleanup(func() {
		_ = dbxDB.WithContext(ctx).Delete(&User{}, u.ID).Error
	})

	var got User
	if err := dbxDB.WithContext(ctx).First(&got, u.ID).Error; err != nil {
		t.Fatalf("First: %v", err)
	}
	if got.ID != u.ID {
		t.Fatalf("expected ID=%d, got %d", u.ID, got.ID)
	}

	requireSpanReported(t, "db.query")
}

func TestDbxWireJaeger_List(t *testing.T) {
	ctx := context.Background()
	u := newTestUser()

	if err := dbxDB.WithContext(ctx).Create(&u).Error; err != nil {
		t.Fatalf("seed Create: %v", err)
	}
	t.Cleanup(func() {
		_ = dbxDB.WithContext(ctx).Delete(&User{}, u.ID).Error
	})

	var users []User
	if err := dbxDB.WithContext(ctx).Find(&users).Error; err != nil {
		t.Fatalf("Find: %v", err)
	}

	requireSpanReported(t, "db.query")
}

func TestDbxWireJaeger_Update(t *testing.T) {
	ctx := context.Background()
	u := newTestUser()

	if err := dbxDB.WithContext(ctx).Create(&u).Error; err != nil {
		t.Fatalf("seed Create: %v", err)
	}
	t.Cleanup(func() {
		_ = dbxDB.WithContext(ctx).Delete(&User{}, u.ID).Error
	})

	u.Name = u.Name + "-updated"
	if err := dbxDB.WithContext(ctx).Save(&u).Error; err != nil {
		t.Fatalf("Save: %v", err)
	}

	requireSpanReported(t, "db.update")
}

func TestDbxWireJaeger_Delete(t *testing.T) {
	ctx := context.Background()
	u := newTestUser()

	if err := dbxDB.WithContext(ctx).Create(&u).Error; err != nil {
		t.Fatalf("seed Create: %v", err)
	}
	// NOTE: no t.Cleanup — the test's own Delete IS the cleanup.

	if err := dbxDB.WithContext(ctx).Delete(&User{}, u.ID).Error; err != nil {
		t.Fatalf("Delete: %v", err)
	}

	requireSpanReported(t, "db.delete")
}
