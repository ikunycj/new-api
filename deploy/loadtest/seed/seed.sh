#!/bin/sh
set -eu

users="${LOADTEST_USERS:-1000}"
case "$users" in
  ''|*[!0-9]*) echo "LOADTEST_USERS must be a positive integer" >&2; exit 2 ;;
esac
if [ "$users" -lt 1 ] || [ "$users" -gt 10000 ]; then
  echo "LOADTEST_USERS must be between 1 and 10000" >&2
  exit 2
fi

until psql -v ON_ERROR_STOP=1 -Atqc "SELECT to_regclass('public.channels') IS NOT NULL" | grep -q t; do
  echo "Waiting for new-api migrations..."
  sleep 2
done

psql -v ON_ERROR_STOP=1 -v loadtest_users="$users" -f /seed/seed.sql
echo "Seeded $users load-test users and tokens."
