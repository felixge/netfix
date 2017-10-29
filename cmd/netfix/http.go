package main

import (
	"database/sql"
	"net/http"
	"time"

	"github.com/felixge/netfix"
)

func serveHttp(c netfix.Config, db *sql.DB) error {
	server := &http.Server{
		Addr:         c.HttpAddr,
		Handler:      OutagesHandler(db),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
	return server.ListenAndServe()
}

func OutagesHandler(db *sql.DB) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		return
		//f := ndb.OutageFilter{
		//MinLoss:        0.01,
		//OutageLoss:     0.01,
		//OutageDuration: 2 * time.Minute,
		//OutageGap:      5 * time.Minute,
		//}
		//fmt.Sscan(r.URL.Query().Get("min_loss"), &f.MinLoss)
		//fmt.Sscan(r.URL.Query().Get("outage_loss"), &f.OutageLoss)
		//if d, err := time.ParseDuration(r.URL.Query().Get("outage_duration")); err == nil {
		//f.OutageDuration = d
		//}
		//if d, err := time.ParseDuration(r.URL.Query().Get("outage_gap")); err == nil {
		//f.OutageGap = d
		//}

		//outages, err := ndb.Outages(db, f)
		//if err != nil {
		//w.WriteHeader(http.StatusInternalServerError)
		//fmt.Fprintf(w, "%s\n", err)
		//return
		//}

		//fmt.Fprintf(w, "%s\n", f)
		//fmt.Fprintf(w, "%d outages (%s):\n\n", len(outages), outages.Duration())
		//sort.Slice(outages, func(i, j int) bool {
		//return outages[i].Start.After(outages[j].Start)
		//})
		//for _, o := range outages {
		//fmt.Fprintf(w, "%s\n", o)
		//}
	})
}
