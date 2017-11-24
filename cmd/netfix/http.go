package main

import (
	"database/sql"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/felixge/netfix"
	"github.com/julienschmidt/httprouter"
)

func serveHttp(c netfix.Config, ln net.Listener, db *sql.DB) error {
	h := &Handlers{DB: db}

	router := httprouter.New()
	router.GET("/api/heatmap", h.Heatmap)
	router.NotFound = http.FileServer(http.Dir(c.HttpDir))

	server := &http.Server{
		Handler:      router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
	return server.Serve(ln)
}

type Handlers struct {
	DB *sql.DB
}

func (h *Handlers) Heatmap(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	// ?start=xxx&end=xxx&interval=xxx&max_duration=xxx
	sql := `
SELECT json_agg(heatmap ORDER BY time DESC, duration DESC)
FROM (
	SELECT
		date_trunc($3::text, started) AS time,
		-- TODO: Is there a more elegant way to do this?
		ceil(duration_ms/10^floor(log(duration_ms)))*10^floor(log(duration_ms)) AS duration,
		count(*) AS count
	FROM (
		SELECT started, least(duration_ms, $4::numeric) AS duration_ms
		FROM pings
		WHERE
			started >= $1
			AND started < $2
			-- TODO(fg) compute duration for pings pending response?
			AND duration_ms IS NOT NULL
	) pings
	GROUP BY 1, 2
) heatmap;
`

	q := r.URL.Query()
	var response []byte
	row := h.DB.QueryRow(
		sql,
		q.Get("start"),
		q.Get("end"),
		q.Get("interval"),
		q.Get("max_duration"),
	)
	if err := row.Scan(&response); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "%s", err)
		return
	}
	w.Write(response)
}
