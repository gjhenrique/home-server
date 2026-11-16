#!/bin/sh
set -eu
: "${DOMAIN:?set DOMAIN, for example tanaka.dev.br}"
: "${SLOPPER_PUBLIC_IP:?run this from outside the private networks and set the public server IP}"
APP="${SLOPPER_ACCEPTANCE_APP:-slopper-demo}"
app_host="$APP.slopper.$DOMAIN"
api_host="api.slopper.$DOMAIN"

expect_status() {
	url=$1 expected=$2
	actual=$(curl -sS -o /dev/null -w '%{http_code}' --resolve "$3:443:$SLOPPER_PUBLIC_IP" "$url")
	if [ "$actual" != "$expected" ]; then
		echo "FAIL $url: expected HTTP $expected, received $actual" >&2
		exit 1
	fi
	echo "PASS $url returned HTTP $actual"
}

# Valid SNI and Host reach the intended virtual hosts from the public network.
expect_status "https://$app_host/" 403 "$app_host"
expect_status "https://$api_host/$APP/identity" 403 "$api_host"
echo "TLS certificate validation passed for both wildcard hostnames."
echo "Now test from LAN and tailnet: app page should redirect to Google SSO; API should return JSON 401."
echo "After login, verify cross-app SSO, portal logout, CRUD, upload/download, and AI manually."
