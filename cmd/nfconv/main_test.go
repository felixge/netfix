package main

import (
	"database/sql"
	"testing"

	"github.com/felixge/netfix"
)

func TestConvert(t *testing.T) {
	db := netfix.TestConfig(t).DB.OpenTest(t)
	if _, err := Convert("test-fixtures/dupe.txt", db); err != nil {
		t.Fatal(err)
	}

	checkCount(t, db, "SELECT count(*) FROM pings", 2)
	checkCount(t, db, "SELECT count(*) FROM pings WHERE timeout = true", 1)
	checkCount(t, db, "SELECT count(*) FROM pings WHERE timeout = false", 1)
}

func checkCount(t *testing.T, db *sql.DB, sql string, want int) {
	t.Helper()
	var got int
	row := db.QueryRow(sql)
	if err := row.Scan(&got); err != nil {
		t.Fatal(err)
	} else if got != want {
		t.Fatalf("got=%d want=%d", got, want)
	}
}
