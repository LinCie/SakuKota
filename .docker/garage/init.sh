#!/usr/bin/env bash
# Sets up the local Garage node started by `docker compose up -d garage`: the
# cluster layout, the API's access key, and the public image bucket with its
# CORS and lifecycle rules. Safe to run again. The key owns the bucket so that it
# can set those rules.
set -euo pipefail

: "${STORAGE_BUCKET:?STORAGE_BUCKET is required}"
: "${STORAGE_ACCESS_KEY_ID:?STORAGE_ACCESS_KEY_ID is required}"
: "${STORAGE_SECRET_ACCESS_KEY:?STORAGE_SECRET_ACCESS_KEY is required}"
CORS_ALLOWED_ORIGIN="${CORS_ALLOWED_ORIGIN:-http://localhost:5173}"

garage() {
	docker compose exec -T garage /garage "$@"
}

container="$(docker compose ps -q garage)"
if [ -z "$container" ]; then
	echo "garage is not running; start it with: docker compose up -d garage" >&2
	exit 1
fi

# The layout is applied once; a node that already has a role keeps it.
if garage layout show | grep -q "No nodes currently have a role"; then
	node_id="$(garage node id -q | cut -d@ -f1)"
	garage layout assign -z dc1 -c 1G "$node_id"
	garage layout apply --version 1
fi

if ! garage key info "$STORAGE_ACCESS_KEY_ID" >/dev/null 2>&1; then
	garage key import -n rantau-api --yes "$STORAGE_ACCESS_KEY_ID" "$STORAGE_SECRET_ACCESS_KEY"
fi

if ! garage bucket info "$STORAGE_BUCKET" >/dev/null 2>&1; then
	garage bucket create "$STORAGE_BUCKET"
fi
garage bucket allow --read --write --owner --key "$STORAGE_ACCESS_KEY_ID" "$STORAGE_BUCKET"
garage bucket website --allow "$STORAGE_BUCKET"

# The AWS CLI runs in garage's network namespace, so the S3 API is on localhost.
aws() {
	docker run --rm --network "container:$container" \
		-e AWS_ACCESS_KEY_ID="$STORAGE_ACCESS_KEY_ID" \
		-e AWS_SECRET_ACCESS_KEY="$STORAGE_SECRET_ACCESS_KEY" \
		-e AWS_DEFAULT_REGION=garage \
		docker.io/amazon/aws-cli:2.39.3 --endpoint-url http://localhost:3900 "$@"
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

echo "garage is ready: bucket $STORAGE_BUCKET"
