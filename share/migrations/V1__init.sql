CREATE TABLE pings (
	started timestamp with time zone PRIMARY KEY,
	duration_ms double precision,
	timeout bool NOT NULL
);

CREATE INDEX ON pings(timeout);
