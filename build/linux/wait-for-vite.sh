#!/bin/sh
set -eu

port=${1:?}
case $port in
'' | *[!0-9]*)
	echo "invalid Vite port" >&2
	exit 1
	;;
esac

uri="http://127.0.0.1:${port}/"
attempt=0
while [ "$attempt" -lt 600 ]; do
	if curl -fsS --max-time 1 "$uri" >/dev/null 2>&1; then
		exit 0
	fi
	attempt=$((attempt + 1))
	sleep 0.1
done

echo "Vite dev server did not become ready at $uri within 60 seconds." >&2
exit 1
