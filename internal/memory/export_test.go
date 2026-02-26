package memory

import "database/sql"

// DB exposes the internal *sql.DB for test helpers in memory_test.
// This file only compiles during `go test`.
func (s *Store) DB() *sql.DB {
	return s.db
}
