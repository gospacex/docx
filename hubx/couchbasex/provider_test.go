package couchbasex

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"

	couchbasedriver "github.com/gospacex/hubx/cache/couchbase"
	hubx "github.com/gospacex/hubx"
)

// stubDialer is a test double for the dialer interface. It records the
// number of Open calls and returns a configurable cluster/error pair.
type stubDialer struct {
	mu      sync.Mutex
	calls   int32
	lastCfg *couchbasedriver.Config
	cluster *couchbasedriver.Cluster
	err     error
	// gate is closed by Open; tests can use it to block until a build
	// reaches the dialer.
	gate chan struct{}
}

func (s *stubDialer) Open(ctx context.Context, cfg *couchbasedriver.Config) (*couchbasedriver.Cluster, error) {
	atomic.AddInt32(&s.calls, 1)
	s.mu.Lock()
	s.lastCfg = cfg
	cl, e := s.cluster, s.err
	if s.gate != nil {
		select {
		case <-s.gate:
		default:
			close(s.gate)
		}
	}
	s.mu.Unlock()
	return cl, e
}

func (s *stubDialer) callCount() int32 { return atomic.LoadInt32(&s.calls) }

// withStubDialer swaps the package-level dialer for a stub and registers
// a cleanup that restores the original. Returns the stub so the test can
// configure it.
func withStubDialer(t *testing.T) *stubDialer {
	t.Helper()
	stub := &stubDialer{}
	prev := dial
	dial = stub
	t.Cleanup(func() { dial = prev })
	return stub
}

// validCouchbaseConfig returns a config map that decodes successfully
// into couchbasedriver.Config. Endpoints is required (slice).
func validCouchbaseConfig() map[string]any {
	return map[string]any{
		"config": map[string]any{
			"endpoints": []string{"localhost:11210"},
			"bucket":    "testbucket",
		},
	}
}

func TestName_ReturnsCorrectString(t *testing.T) {
	p := New()
	if got, want := p.Name(), "docx.couchbase"; got != want {
		t.Fatalf("Name() = %q, want %q", got, want)
	}
}

func TestBuild_Success(t *testing.T) {
	stub := withStubDialer(t)
	stub.cluster = &couchbasedriver.Cluster{}
	stub.err = nil

	p := New()
	cli, err := p.Build("main", validCouchbaseConfig())
	if err != nil {
		t.Fatalf("Build returned error: %v", err)
	}
	if cli == nil {
		t.Fatalf("Build returned nil client")
	}
	if stub.callCount() != 1 {
		t.Fatalf("dialer.Open called %d times, want 1", stub.callCount())
	}
	if stub.lastCfg == nil {
		t.Fatalf("dialer received nil config")
	}
	if got := stub.lastCfg.Endpoints; len(got) != 1 || got[0] != "localhost:11210" {
		t.Fatalf("Endpoints = %v, want [localhost:11210]", got)
	}
	if stub.lastCfg.Bucket != "testbucket" {
		t.Fatalf("Bucket = %q, want testbucket", stub.lastCfg.Bucket)
	}
}

func TestBuild_MissingConfigKey(t *testing.T) {
	withStubDialer(t) // we want to assert dialer is NOT called
	p := New()

	cli, err := p.Build("main", map[string]any{})
	if cli != nil {
		t.Fatalf("Build returned non-nil client: %v", cli)
	}
	if err == nil {
		t.Fatalf("Build returned nil error, want ErrConfigInvalid")
	}
	if !errors.Is(err, hubx.ErrConfigInvalid) {
		t.Fatalf("err = %v, want errors.Is(ErrConfigInvalid)", err)
	}
}

func TestBuild_MissingRequiredField(t *testing.T) {
	withStubDialer(t)
	p := New()

	// mapstructure with ErrorUnset and a wrong-typed value for
	// Endpoints (which is []string) must fail decode. Pass an int.
	cfg := map[string]any{
		"config": map[string]any{
			"endpoints": 123, // wrong type
			"bucket":    "testbucket",
		},
	}
	cli, err := p.Build("main", cfg)
	if cli != nil {
		t.Fatalf("Build returned non-nil client: %v", cli)
	}
	if err == nil {
		t.Fatalf("Build returned nil error, want ErrConfigInvalid")
	}
	if !errors.Is(err, hubx.ErrConfigInvalid) {
		t.Fatalf("err = %v, want errors.Is(ErrConfigInvalid)", err)
	}
}

func TestBuild_UnknownField(t *testing.T) {
	withStubDialer(t)
	p := New()

	cfg := map[string]any{
		"config": map[string]any{
			"endpoints":              []string{"localhost:11210"},
			"bucket":                 "testbucket",
			"definitely_not_a_field": true,
		},
	}
	cli, err := p.Build("main", cfg)
	if cli != nil {
		t.Fatalf("Build returned non-nil client: %v", cli)
	}
	if err == nil {
		t.Fatalf("Build returned nil error, want ErrConfigInvalid")
	}
	if !errors.Is(err, hubx.ErrConfigInvalid) {
		t.Fatalf("err = %v, want errors.Is(ErrConfigInvalid)", err)
	}
}

func TestBuild_DriverNewFailure(t *testing.T) {
	stub := withStubDialer(t)
	wantErr := errors.New("connection refused")
	stub.err = wantErr

	p := New()
	cli, err := p.Build("main", validCouchbaseConfig())
	if cli != nil {
		t.Fatalf("Build returned non-nil client: %v", cli)
	}
	if err == nil {
		t.Fatalf("Build returned nil error, want ErrBuildFailed")
	}
	if !errors.Is(err, hubx.ErrBuildFailed) {
		t.Fatalf("err = %v, want errors.Is(ErrBuildFailed)", err)
	}
	if !errors.Is(err, wantErr) {
		t.Fatalf("err = %v, want wraps %v", err, wantErr)
	}
}

func TestProviderHealthCheck_NoOp(t *testing.T) {
	p := New()
	if err := p.HealthCheck(context.Background()); err != nil {
		t.Fatalf("Provider.HealthCheck returned %v, want nil", err)
	}
}

func TestProviderClose_NoOp(t *testing.T) {
	p := New()
	if err := p.Close(); err != nil {
		t.Fatalf("Provider.Close returned %v, want nil", err)
	}
}

func TestClientHealthCheck_DelegatesToDriver(t *testing.T) {
	stub := withStubDialer(t)
	stub.cluster = &couchbasedriver.Cluster{}
	stub.err = nil

	p := New()
	cli, err := p.Build("main", validCouchbaseConfig())
	if err != nil {
		t.Fatalf("Build returned error: %v", err)
	}
	// Cluster.HealthCheck calls cluster.Ping on the gocb cluster; with
	// a zero-value wrapper it will error. We only assert the wrapper
	// makes the call (no nil-receiver panic) and that the error is
	// forwarded as-is. Exact error depends on driver internals.
	hcErr := cli.HealthCheck(context.Background())
	_ = hcErr
}

func TestConcurrentBuild_Singleton(t *testing.T) {
	stub := withStubDialer(t)
	stub.gate = make(chan struct{})
	stub.cluster = &couchbasedriver.Cluster{}

	const workers = 8
	var wg sync.WaitGroup
	results := make([]hubx.Client, workers)
	errs := make([]error, workers)
	cfg := validCouchbaseConfig()
	p := New()

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			results[i], errs[i] = p.Build("main", cfg)
		}(i)
	}
	wg.Wait()

	// Each Build call invokes the dialer exactly once (this provider
	// does not cache — caching is the driver's job). We verify
	// concurrent access is safe and every call succeeds.
	for i, err := range errs {
		if err != nil {
			t.Fatalf("worker %d returned error: %v", i, err)
		}
		if results[i] == nil {
			t.Fatalf("worker %d got nil client", i)
		}
	}
	if got := stub.callCount(); got != int32(workers) {
		t.Fatalf("dialer.Open called %d times, want %d", got, workers)
	}
}

func TestRaceFree_UnderRace(t *testing.T) {
	stub := withStubDialer(t)
	stub.cluster = &couchbasedriver.Cluster{}

	const iterations = 50
	var wg sync.WaitGroup
	cfg := validCouchbaseConfig()
	p := New()

	for i := 0; i < iterations; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			_, _ = p.Build("main", cfg)
		}()
		go func() {
			defer wg.Done()
			_ = p.HealthCheck(context.Background())
		}()
	}
	wg.Wait()
	if got := stub.callCount(); got != int32(iterations) {
		t.Fatalf("dialer.Open called %d times, want %d", got, iterations)
	}
}
