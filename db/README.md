# Storage

1 file per day, 86,400 * 2 bytes little endian where each element contains the
ping in ms. The highest 1024 values, e.g. 256^2 - 1, 256^2 - 2, etc. are
reserved for error codes. The filename is the date in UTC. Extension is .pingdb.
