package main

import (
	"database/sql"
	"testing"
	"time"

	ndb "github.com/felixge/netfix/db"
)

func TestStore(t *testing.T) {
	db, err := ndb.Open("")
	noErr(t, err)
	_, _, err = ndb.Migrate(db)
	noErr(t, err)

	now := time.Unix(1508848335, 0)
	pg := ndb.Ping{Start: now}
	noErr(t, store(db, pg))
	checkPings(t, db, "[1508848335.0,null,0]")

	pg.Duration = 100 * time.Millisecond
	noErr(t, store(db, pg))
	checkPings(t, db, "[1508848335.0,100.0,0]")

	pg.Duration = 1 * time.Second
	pg.Timeout = true
	noErr(t, store(db, pg))
	checkPings(t, db, "[1508848335.0,100.0,0]")
}

func noErr(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func checkPings(t *testing.T, db *sql.DB, want string) {
	t.Helper()
	var got string
	sql := "SELECT group_concat(json_array(start, duration, timeout), '\n') FROM pings"
	row := db.QueryRow(sql)
	noErr(t, row.Scan(&got))
	if got != want {
		t.Fatalf("\ngot =%s\nwant=%s", got, want)
	}
}
