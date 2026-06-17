// Package couchbasex implements hubx.ClientProvider for "docx.couchbase".
//
// The provider wraps github.com/gospacex/hubx/cache/couchbase (a thin
// wrapper around the official github.com/couchbase/gocb/v2 SDK) as a
// hubx.Client. COC (Couchbase Open Cluster) is the canonical synchronous
// entry point exposed by the driver, and the provider delegates to it.
//
// Because the driver requires a live Couchbase cluster to perform a real
// WaitUntilReady() during COC(), this provider exposes a package-level
// dialer variable (`dialer`) that tests can swap for a stub. Production
// code leaves the default in place and is fully exercised by the
// `integration` build tag tests.
package couchbasex

import (
	"context"
	"fmt"

	couchbasedriver "github.com/gospacex/hubx/cache/couchbase"
	hubx "github.com/gospacex/hubx"
	"github.com/mitchellh/mapstructure"
)

// dialer is the seam used by Build to construct a *couchbasedriver.Cluster.
// Tests can replace it with a stub that does not require a running
// Couchbase server.
type dialer interface {
	Open(ctx context.Context, cfg *couchbasedriver.Config) (*couchbasedriver.Cluster, error)
}

type realDialer struct{}

func (realDialer) Open(ctx context.Context, cfg *couchbasedriver.Config) (*couchbasedriver.Cluster, error) {
	return couchbasedriver.COC(ctx, cfg)
}

// dial is the active dialer. Package-level so tests can swap it via
// reassignment.
var dial dialer = realDialer{}

// Provider implements hubx.ClientProvider for the "docx.couchbase" driver.
type Provider struct{}

// New returns a new docx.couchbase Provider.
func New() *Provider { return &Provider{} }

// Name returns the registry name.
func (p *Provider) Name() string { return "docx.couchbase" }

// Build decodes cfg["config"] into couchbasedriver.Config via
// mapstructure (TagName: "yaml", ErrorUnset / ErrorUnused both enabled)
// and then calls the active dialer. Errors are wrapped with the
// appropriate hubx sentinel:
//   - missing "config" key → hubx.ErrConfigInvalid
//   - mapstructure failure → hubx.ErrConfigInvalid
//   - dialer Open failure  → hubx.ErrBuildFailed
func (p *Provider) Build(instanceName string, cfg map[string]any) (hubx.Client, error) {
	raw, ok := cfg["config"]
	if !ok {
		return nil, fmt.Errorf("%w: docx.couchbase/%s: missing 'config' key", hubx.ErrConfigInvalid, instanceName)
	}

	var c couchbasedriver.Config
	dec, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		TagName:     "yaml",
		ErrorUnset:  true,
		ErrorUnused: true,
		Result:      &c,
		ZeroFields:  false,
	})
	if err != nil {
		return nil, fmt.Errorf("%w: docx.couchbase/%s: decoder: %v", hubx.ErrConfigInvalid, instanceName, err)
	}
	if err := dec.Decode(raw); err != nil {
		return nil, fmt.Errorf("%w: docx.couchbase/%s: %v", hubx.ErrConfigInvalid, instanceName, err)
	}

	cl, err := dial.Open(context.Background(), &c)
	if err != nil {
		return nil, fmt.Errorf("%w: docx.couchbase/%s: %v", hubx.ErrBuildFailed, instanceName, err)
	}
	return &client{c: cl}, nil
}

// HealthCheck is a no-op for the provider itself — the provider owns
// no connection state.
func (p *Provider) HealthCheck(context.Context) error { return nil }

// Close is a no-op for the provider itself.
func (p *Provider) Close() error { return nil }

// client wraps a *couchbasedriver.Cluster as a hubx.Client. The
// driver's Close ignores the context, which matches the hubx.Client
// signature.
type client struct{ c *couchbasedriver.Cluster }

// HealthCheck delegates to the driver's HealthCheck (which itself
// calls Cluster.Ping).
func (c *client) HealthCheck(ctx context.Context) error { return c.c.HealthCheck(ctx) }

// Close delegates to the driver's Close (which evicts the cache
// entry and closes the underlying gocb cluster).
func (c *client) Close() error { return c.c.Close() }
