package main

import (
	"context"
	"fmt"
	"log"
)

// Database is a service interface for database operations
type Database interface {
	Query(ctx context.Context, sql string) ([]string, error)
	Close() error
}

// Cache is a service interface for cache operations
type Cache interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key, value string) error
	Close() error
}

// Logger is a service interface for logging operations
type Logger interface {
	Info(ctx context.Context, msg string) error
	Error(ctx context.Context, msg string) error
	Close() error
}

// InMemoryDatabase implements Database
type InMemoryDatabase struct {
	host string
	port int
}

// NewDatabase creates a Database instance
func NewDatabase(host string, port int) (Database, error) {
	return &InMemoryDatabase{host: host, port: port}, nil
}

func (d *InMemoryDatabase) Query(ctx context.Context, sql string) ([]string, error) {
	return []string{fmt.Sprintf("result from %s:%d", d.host, d.port)}, nil
}

func (d *InMemoryDatabase) Close() error {
	fmt.Println("Database closed")
	return nil
}

// InMemoryCache implements Cache
type InMemoryCache struct {
	data map[string]string
}

// NewCache creates a Cache instance
func NewCache() (Cache, error) {
	return &InMemoryCache{data: make(map[string]string)}, nil
}

func (c *InMemoryCache) Get(ctx context.Context, key string) (string, error) {
	if v, ok := c.data[key]; ok {
		return v, nil
	}
	return "", fmt.Errorf("key not found: %s", key)
}

func (c *InMemoryCache) Set(ctx context.Context, key, value string) error {
	c.data[key] = value
	return nil
}

func (c *InMemoryCache) Close() error {
	fmt.Println("Cache closed")
	return nil
}

// ConsoleLogger implements Logger
type ConsoleLogger struct {
	prefix string
}

// NewLogger creates a Logger instance
func NewLogger(prefix string) (Logger, error) {
	return &ConsoleLogger{prefix: prefix}, nil
}

func (l *ConsoleLogger) Info(ctx context.Context, msg string) error {
	fmt.Printf("[%s INFO] %s\n", l.prefix, msg)
	return nil
}

func (l *ConsoleLogger) Error(ctx context.Context, msg string) error {
	fmt.Printf("[%s ERROR] %s\n", l.prefix, msg)
	return nil
}

func (l *ConsoleLogger) Close() error {
	fmt.Println("Logger closed")
	return nil
}

// AppService depends on Database, Cache, and Logger
type AppService struct {
	db    Database
	cache Cache
	log   Logger
}

// NewAppService creates AppService with manual DI
func NewAppService(db Database, cache Cache, log Logger) (*AppService, error) {
	return &AppService{
		db:    db,
		cache: cache,
		log:   log,
	}, nil
}

func (s *AppService) DoWork(ctx context.Context) error {
	if err := s.log.Info(ctx, "starting work"); err != nil {
		return err
	}

	if err := s.cache.Set(ctx, "key1", "value1"); err != nil {
		return err
	}

	val, err := s.cache.Get(ctx, "key1")
	if err != nil {
		return err
	}

	results, err := s.db.Query(ctx, "SELECT * FROM table")
	if err != nil {
		return err
	}

	if err := s.log.Info(ctx, fmt.Sprintf("cache hit: %s, db results: %v", val, results)); err != nil {
		return err
	}

	return nil
}

func main() {
	ctx := context.Background()

	// Manual DI way - showing how you'd wire manually
	db, err := NewDatabase("localhost", 5432)
	if err != nil {
		log.Fatalf("failed to create database: %v", err)
	}

	cache, err := NewCache()
	if err != nil {
		log.Fatalf("failed to create cache: %v", err)
	}

	logger, err := NewLogger("APP")
	if err != nil {
		log.Fatalf("failed to create logger: %v", err)
	}

	// Wire manually
	appService, err := NewAppService(db, cache, logger)
	if err != nil {
		log.Fatalf("failed to create app service: %v", err)
	}

	// Run
	if err := appService.DoWork(ctx); err != nil {
		log.Fatalf("app service failed: %v", err)
	}

	// Cleanup
	if err := appService.db.Close(); err != nil {
		log.Printf("failed to close database: %v", err)
	}
	if err := appService.cache.Close(); err != nil {
		log.Printf("failed to close cache: %v", err)
	}
	if err := appService.log.Close(); err != nil {
		log.Printf("failed to close logger: %v", err)
	}
}
