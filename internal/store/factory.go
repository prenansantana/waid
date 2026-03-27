package store

import "fmt"

// New constructs a Store for the given driver and connection URL.
// Supported drivers: "sqlite", "postgres".
func New(driver, url string) (Store, error) {
	switch driver {
	case "sqlite":
		return NewSQLite(url)
	case "postgres":
		return NewPostgres(url)
	default:
		return nil, fmt.Errorf("unsupported database driver: %s", driver)
	}
}
