// Package main wires dbx into a native *gorm.DB handle for go test.
// Unlike cachex's CachexProvider, DbxProvider does NOT set up OTel globals;
// the caller (TestMain) owns the TracerProvider three-step:
// dbsql.CreateExporter -> sdktrace.NewTracerProvider -> otel.SetTracerProvider.
// This mirrors how production dbx consumers must wire tracing themselves,
// because dbsql.ExtractTracingAndApply is a documented no-op.
package main

import (
	"fmt"

	"github.com/gospacex/dbx/dbsql"
	"gorm.io/gorm"
)

// DbxProvider loads dbx mysql configs and produces native *gorm.DB handles.
// It is stateless and safe for concurrent use.
type DbxProvider struct{}

// NewDbxProvider returns a fresh DbxProvider for wire injection.
func NewDbxProvider() *DbxProvider { return &DbxProvider{} }

// ProvideDB loads cfgPath via dbx's config.Load and returns the pooled
// native *gorm.DB. GORM v2 callbacks (dbx/orm/gorm_tracing.go) emit spans
// into whatever TracerProvider is installed globally by TestMain.
//
// cfgPath is passed straight through to dbsql.OpenPath; config.Load is
// called inside OpenPath so callers see a single error path. We do NOT
// re-validate here — dbx is the source of truth for schema.
//
// Errors are wrapped with cfgPath so callers can debug, but the password
// field from yaml is intentionally not echoed (no config.Password leak).
func (p *DbxProvider) ProvideDB(cfgPath string) (*gorm.DB, error) {
	if cfgPath == "" {
		return nil, fmt.Errorf("provide db: cfgPath is empty")
	}
	db, err := dbsql.OpenPath(cfgPath)
	if err != nil {
		return nil, fmt.Errorf("provide db: dbsql.OpenPath %q: %w", cfgPath, err)
	}
	return db, nil
}
