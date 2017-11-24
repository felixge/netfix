#!/usr/bin/env bash
set -eu

# ./pi_install.sh installs a netfix build on a Raspberry PI. It doesn't install
# or configure dependencies such as netfix or postgres.
#
# Note: This is not intendet to work for you unless you're me. At least for now
# :).

echo "-> updating systemd unit file"
cp netfix.service /etc/systemd/system/
systemctl daemon-reload
echo "-> stopping service"
systemctl stop netfix.service || true
echo "-> updating files"
mkdir -p /usr/share/netfix
cp -R migrations www /usr/share/netfix
cp config.sh /etc/netfix.sh
cp bin/* /usr/bin/
cat << FOE >> /etc/netfix.sh
export PGDATABASE=postgres
export PGPASSWORD=TZ846sXbFEwD.Pb]2YfP
export NF_HTTP_ADDR=:80
export NF_HTTP_DIR=/usr/share/netfix/www
export NF_FLYWAY_BIN=/home/pi/flyway-4.2.0/flyway
FOE

echo "-> running migrations"
source /etc/netfix.sh
/usr/share/netfix/migrations/flyway.sh migrate

echo "-> starting service"
systemctl start netfix.service
