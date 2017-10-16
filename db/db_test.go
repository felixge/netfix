package db

import (
	"io/ioutil"
	"os"
	"reflect"
	"testing"
	"time"
)

func TestWriteRead(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)
	db, err := Open(tmpDir)
	if err != nil {
		t.Fatal(err)
	}
	tt := time.Date(2017, time.October, 13, 0, 59, 59, 0, time.FixedZone("somewhere", 3600))
	writeVals := []Val{1, 2, 3}
	if err := db.Write(tt, writeVals); err != nil {
		t.Fatal(err)
	}
	readVals, err := db.Read(tt, tt.Add(3*time.Second))
	if err != nil {
		t.Fatal(err)
	} else if !reflect.DeepEqual(readVals, writeVals) {
		t.Fatalf("got=%v want=%v", readVals, writeVals)
	}
}
