#!/bin/sh
set -eu

base_url="${PRESENCE_URL:-http://localhost:8080}"

cleanup() {
  curl --fail-with-body -X DELETE "$base_url/admin/tenants/acme" \
    -H 'Idempotency-Key: delete-acme-001'
}
trap cleanup EXIT

curl --fail-with-body -X POST "$base_url/admin/tenants" \
  -H 'Content-Type: application/json' \
  -d '{"tenant_id":"acme","request_id":"onboard-acme-001"}'

curl --fail-with-body -X POST "$base_url/admin/tenants/acme/accounts" \
  -H 'Content-Type: application/json' \
  -d '{"account_id":"user-7","request_id":"register-user-7-001"}'

curl --fail-with-body -X GET "$base_url/workspaces/acme/online"
