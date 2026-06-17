// Package mongox implements hubx.ClientProvider for "docx.mongo".
//
// The provider wraps github.com/gospacex/hubx/cache/mongo (a thin wrapper
// around the official go.mongodb.org/mongo-driver) as a hubx.Client.
// MOC (Mongo Open Client) is the canonical synchronous entry point
// exposed by the driver, and the provider delegates to it.
//
// Because the driver requires a live MongoDB server to perform a real
// Ping() during MOC(), this provider exposes a package-level dialer
// variable (`dialer`) that tests can swap for a stub. Production code
// leaves the default in place and is fully exercised by the
// `integration` build tag tests.
package mongox

import (
	"context"
	"fmt"

	hubx "github.com/gospacex/hubx"
	mongodriver "github.com/gospacex/hubx/cache/mongo"
	"github.com/mitchellh/mapstructure"
)

// dialer is the seam used by Build to construct a *mongodriver.Client.
// Tests can replace it with a stub that does not require a running
// MongoDB server.
type dialer interface {
	Open(ctx context.Context, cfg *mongodriver.Config) (*mongodriver.Client, error)
}

type realDialer struct{}

func (realDialer) Open(ctx context.Context, cfg *mongodriver.Config) (*mongodriver.Client, error) {
	return mongodriver.MOC(ctx, cfg)
}

// dial is the active dialer. Package-level so tests can swap it via
// reassignment; protected by a small critical section only at test
// boundary (reads inside Build happen in a single goroutine).
var dial dialer = realDialer{}

// Provider implements hubx.ClientProvider for the "docx.mongo" driver.
type Provider struct{}

// New returns a new docx.mongo Provider.
func New() *Provider { return &Provider{} }

// Name returns the registry name.
func (p *Provider) Name() string { return "docx.mongo" }

// Build decodes cfg["config"] into mongodriver.Config via mapstructure
// (TagName: "yaml", ErrorUnset / ErrorUnused both enabled) and then
// calls the active dialer. Errors are wrapped with the appropriate
// hubx sentinel:
//   - missing "config" key → hubx.ErrConfigInvalid
//   - mapstructure failure → hubx.ErrConfigInvalid
//   - dialer Open failure  → hubx.ErrBuildFailed
func (p *Provider) Build(instanceName string, cfg map[string]any) (hubx.Client, error) {
	raw, ok := cfg["config"]
	if !ok {
		return nil, fmt.Errorf("%w: docx.mongo/%s: missing 'config' key", hubx.ErrConfigInvalid, instanceName)
	}

	var c mongodriver.Config
	dec, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		TagName:     "yaml",
		ErrorUnset:  true,
		ErrorUnused: true,
		Result:      &c,
		ZeroFields:  false,
	})
	if err != nil {
		return nil, fmt.Errorf("%w: docx.mongo/%s: decoder: %v", hubx.ErrConfigInvalid, instanceName, err)
	}
	if err := dec.Decode(raw); err != nil {
		return nil, fmt.Errorf("%w: docx.mongo/%s: %v", hubx.ErrConfigInvalid, instanceName, err)
	}

	cli, err := dial.Open(context.Background(), &c)
	if err != nil {
		return nil, fmt.Errorf("%w: docx.mongo/%s: %v", hubx.ErrBuildFailed, instanceName, err)
	}
	return &client{c: cli}, nil
}

// HealthCheck is a no-op for the provider itself — the provider owns
// no connection state.
func (p *Provider) HealthCheck(context.Context) error { return nil }

// Close is a no-op for the provider itself.
func (p *Provider) Close() error { return nil }

// client wraps a *mongodriver.Client as a hubx.Client. The driver's
// Close takes a context, so we accept context.Background() from the
// hubx registry; tests should not call this in tight loops.
type client struct{ c *mongodriver.Client }

// HealthCheck delegates to the driver's Ping.
func (c *client) HealthCheck(ctx context.Context) error { return c.c.HealthCheck(ctx) }

// Close delegates to the driver's Disconnect and is best-effort;
// a disconnect error is swallowed because hubx only needs to know
// the client was closed (the registry will shut the process down
// after Close returns).
func (c *client) Close() error { return c.c.Close(context.Background()) }
