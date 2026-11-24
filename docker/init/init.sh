#!/bin/sh
# One-shot setup for the compose stack: database migrations and the Garage node
# (cluster layout, API key, public image bucket with its CORS and lifecycle
# rules). Safe to run again. Seed data is loaded manually.
set -eu

: "${DATABASE_URL:?DATABASE_URL is required}"
: "${GARAGE_RPC_SECRET:?GARAGE_RPC_SECRET is required}"
: "${STORAGE_ENDPOINT:?STORAGE_ENDPOINT is required}"
: "${STORAGE_BUCKET:?STORAGE_BUCKET is required}"
: "${STORAGE_ACCESS_KEY_ID:?STORAGE_ACCESS_KEY_ID is required}"
: "${STORAGE_SECRET_ACCESS_KEY:?STORAGE_SECRET_ACCESS_KEY is required}"
: "${CORS_ALLOWED_ORIGIN:?CORS_ALLOWED_ORIGIN is required}"

echo "==> migrations"
migrate up

echo "==> storage"
# The CLI reads the node key from the Garage metadata volume, which exists once
# the node has started.
until node_id="$(garage node id -q 2>/dev/null | cut -d@ -f1)" && [ -n "$node_id" ]; do
	echo "waiting for garage"
	sleep 1
done

garage() {
	command garage -h "$node_id@${GARAGE_RPC_HOST:-garage:3901}" "$@"
}

until garage status >/dev/null 2>&1; do
	echo "waiting for garage"
	sleep 1
done

# The layout is applied once; a node that already has a role keeps it.
if garage layout show | grep -q "No nodes currently have a role"; then
	garage layout assign -z dc1 -c 1G "$node_id"
	garage layout apply --version 1
fi

if ! garage key info "$STORAGE_ACCESS_KEY_ID" >/dev/null 2>&1; then
	garage key import -n sakukota-api --yes "$STORAGE_ACCESS_KEY_ID" "$STORAGE_SECRET_ACCESS_KEY"
fi

if ! garage bucket info "$STORAGE_BUCKET" >/dev/null 2>&1; then
	garage bucket create "$STORAGE_BUCKET"
fi
garage bucket allow --read --write --owner --key "$STORAGE_ACCESS_KEY_ID" "$STORAGE_BUCKET"
garage bucket website --allow "$STORAGE_BUCKET"

export AWS_ACCESS_KEY_ID="$STORAGE_ACCESS_KEY_ID"
export AWS_SECRET_ACCESS_KEY="$STORAGE_SECRET_ACCESS_KEY"
export AWS_DEFAULT_REGION="${STORAGE_REGION:-garage}"

aws() {
	command aws --endpoint-url "$STORAGE_ENDPOINT" "$@"
}

# Browsers upload straight to the bucket with presigned PUT URLs. Garage matches
# header names case-sensitively, and browsers send them in lowercase.
aws s3api put-bucket-cors --bucket "$STORAGE_BUCKET" --cors-configuration "{
	\"CORSRules\": [{
		\"AllowedOrigins\": [\"$CORS_ALLOWED_ORIGIN\"],
		\"AllowedMethods\": [\"PUT\"],
		\"AllowedHeaders\": [\"content-type\"],
		\"MaxAgeSeconds\": 3600
	}]
}"

# Uploads that are never confirmed are removed after a day.
aws s3api put-bucket-lifecycle-configuration --bucket "$STORAGE_BUCKET" --lifecycle-configuration '{
	"Rules": [{
		"ID": "expire-unconfirmed-uploads",
		"Status": "Enabled",
		"Filter": {"Prefix": "uploads/"},
		"Expiration": {"Days": 1}
	}]
}'

echo "init done: bucket $STORAGE_BUCKET"
