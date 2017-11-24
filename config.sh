# This is the netfix configuration file. Everything is configured with
# environment variables.

# == POSTGRES ==
export PGHOST=localhost
export PGPORT=5432
export PGUSER=postgres
export PGPASSWORD=
export PGDATABASE=netfix
export PGSSLMODE=disable

# == PING TARGET ==
export NF_HTTP_ADDR="127.0.0.1:1234"
export NF_TARGET="google.de"
export NF_IP_VERSION=4
export NF_INTERVAL="1s"
export NF_TIMEOUT="1s"

# == DIRECTORIES ==

# PARENT_DIR is a helper variable that holds the absolute path to the directory
# this file is in. The code below uses some heuristics to find the location
# of the directories netfix needs to operate.
PARENT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

# NF_HTTP_DIR is the absolute path to the netfix www directory.
export NF_HTTP_DIR="${PARENT_DIR}/www"

# == DANGER ZONE: DO NOT MODIFY ==

export NF_PGSCHEMAS="netfix"
