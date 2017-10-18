package netfix

import (
	"database/sql"
	"fmt"
)

func Migrate(db *sql.DB) (from, to int, err error) {
	tx, err := db.Begin()
	if err != nil {
		return 0, 0, err
	}
	defer tx.Rollback()

	migrations := []func() error{
		func() error {
			_, err = tx.Exec(`
CREATE TABLE schema_versions (
	version int
);

CREATE TABLE pings (
	time integer PRIMARY KEY,
	millisec integer,
	timeout bool
);`,
			)
			return err
		},
	}

	_ = tx.QueryRow("SELECT max(version) FROM schema_versions").Scan(&from)
	if from > len(migrations) {
		return 0, 0, fmt.Errorf("unknown schema_version: %d", from)
	}
	for i, mig := range migrations[from:] {
		if err := mig(); err != nil {
			return 0, 0, err
		} else if _, err := tx.Exec("INSERT INTO schema_versions VALUES ($1);", i+from+1); err != nil {
			return 0, 0, err
		}
	}
	to = len(migrations)
	return from, to, tx.Commit()
}
