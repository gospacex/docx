package main

import (
	"fmt"
	"math/rand"
	"time"

	"gorm.io/gorm"
)

// User is the test row used by all 5 TestDbxWireJaeger_* cases. We embed
// gorm.Model to inherit ID/CreatedAt/UpdatedAt/DeletedAt and add a unique
// Name so each test row is identifiable without colliding across runs.
type User struct {
	gorm.Model
	Name string `gorm:"size:64;uniqueIndex"`
}

// TableName pins the SQL table name to `users` regardless of GORM's
// default pluralization rules; makes span attributes and MySQL audit
// queries deterministic.
func (User) TableName() string { return "users" }

// newTestUser returns a User with a unique Name keyed off the current
// nanosecond + test-local random suffix, so parallel/repeated runs cannot
// collide on the unique index. Cleanup is the caller's responsibility.
func newTestUser() User {
	return User{
		Name: fmt.Sprintf("user-%d-%d", time.Now().UnixNano(), rand.Int63()),
	}
}
