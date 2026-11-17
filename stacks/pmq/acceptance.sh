#!/bin/sh
# Post-deploy acceptance for the PMQ stack on Lovelace.
# Run from a client whose source IP is inside PMQ_OPERATOR_CIDRS.
# Optionally set PMQ_PUBLIC_IP (run once from outside the private networks)
# to confirm the operator hostname stays private.
set -eu

: "${DOMAIN:?set DOMAIN, for example tanaka.dev.br}"
: "${PMQ_PROBE_IP:?set to the Lovelace address Caddy listens on}"
: "${PMQ_SMOKE_APP:=hello-world}"
: "${PMQ_PUBLIC_IP:=}"

app_host="${PMQ_SMOKE_APP}.quick.${DOMAIN}"
admin_host="pmq-admin.quick.${DOMAIN}"

status() {
  host=$1 path=$2 ip=$3
  curl --silent --show-error --insecure --output /dev/null --write-out '%{http_code}' \
    --resolve "${host}:443:${ip}" "https://${host}${path}"
}
expect() {
  host=$1 path=$2 expected=$3 ip=$4
  actual=$(status "$host" "$path" "$ip")
  if [ "$actual" != "$expected" ]; then
    echo "FAIL https://${host}${path}: expected HTTP ${expected}, received HTTP ${actual}" >&2
    exit 1
  fi
  echo "PASS https://${host}${path}: HTTP ${actual} (expected)"
}

# App hosts reach their SSO boundary; identity-free requests must be
# rejected or redirected to the Google login.
code=$(status "$app_host" "/" "$PMQ_PROBE_IP")
case "$code" in
  401|403|302|303|307|308) echo "PASS app SSO boundary: HTTP ${code}" ;;
  *) echo "FAIL ${app_host}/: expected 401/403 or login redirect, got HTTP ${code}" >&2; exit 1 ;;
esac

# App hosts must never proxy administrative routes.
expect "$app_host" "/_pmq/admin/apps" 404 "$PMQ_PROBE_IP"

# From an allowed operator source the admin API is reached and still gated
# by PMQ's independent bearer token (intentionally omitted here).
expect "$admin_host" "/_pmq/admin/apps" 401 "$PMQ_PROBE_IP"

# The operator host routes nothing but the admin API.
expect "$admin_host" "/" 404 "$PMQ_PROBE_IP"

if [ -n "$PMQ_PUBLIC_IP" ]; then
  # From outside the private networks the operator host must fail closed.
  expect "$admin_host" "/_pmq/admin/apps" 403 "$PMQ_PUBLIC_IP"
fi

echo "PMQ boundary checks passed. Next: sign in with an allowlisted account,"
echo "deploy the smoke app and open https://${app_host}/:"
echo "  pmq deploy ./hello-world --name ${PMQ_SMOKE_APP} --server https://${admin_host}"
