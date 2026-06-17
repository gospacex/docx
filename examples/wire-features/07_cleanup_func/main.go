package main

import (
	"fmt"
	"log"
)

// CleanupFunc is a function type for cleanup.
type CleanupFunc func()

// Database wraps a connection that needs cleanup.
type Database struct {
	conn string
}

// NewDatabase creates a Database instance.
func NewDatabase() *Database {
	return &Database{conn: "db-connection-string"}
}

// Close cleans up the database connection.
func (d *Database) Close() error {
	fmt.Println("Database closed")
	return nil
}

// Cache wraps a cache connection that needs cleanup.
type Cache struct {
	addr string
}

// NewCache creates a Cache instance.
func NewCache() *Cache {
	return &Cache{addr: "localhost:6379"}
}

// Close cleans up the cache connection.
func (c *Cache) Close() error {
	fmt.Println("Cache closed")
	return nil
}

// Service depends on Database and Cache, both requiring cleanup.
type Service struct {
	DB    *Database
	Cache *Cache
}

// NewService creates a Service with injected dependencies.
func NewService(db *Database, cache *Cache) *Service {
	return &Service{DB: db, Cache: cache}
}

// Close cleans up both Database and Cache resources.
func (s *Service) Close() error {
	fmt.Println("Service cleanup started")
	if err := s.DB.Close(); err != nil {
		return err
	}
	if err := s.Cache.Close(); err != nil {
		return err
	}
	fmt.Println("Service cleanup done")
	return nil
}

// ServiceOwner holds a service and its cleanup function.
type ServiceOwner struct {
	Service *Service
	Cleanup CleanupFunc
}

// newCleanupFunc creates a cleanup function from the service.
func newCleanupFunc(svc *Service) (CleanupFunc, error) {
	return func() {
		svc.Close()
	}, nil
}

// ManualDI demonstrates how to manually wire dependencies and handle cleanup.
func ManualDI() (*Service, func(), error) {
	db := NewDatabase()
	cache := NewCache()
	svc := NewService(db, cache)
	cleanup := func() {
		fmt.Println("Manual cleanup called")
		svc.Close()
	}
	return svc, cleanup, nil
}

func main() {
	// Manual dependency injection with cleanup
	fmt.Println("=== Manual DI ===")
	_, cleanup, err := ManualDI()
	if err != nil {
		log.Fatal(err)
	}
	cleanup()

	// Wire-generated dependency injection with cleanup
	fmt.Println("\n=== Wire DI ===")
	owner, err := InitializeService()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Service created with DB: %s, Cache: %s\n", owner.Service.DB.conn, owner.Service.Cache.addr)
	owner.Cleanup()
}
