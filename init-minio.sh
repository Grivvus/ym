#!/bin/sh
set -eu

WORKDIR="/workdir"
ENV_DIR="$WORKDIR/env_files"
ENV_FILE="$ENV_DIR/.env.minio"

MC_ALIAS="myminio"
MC_ENDPOINT="http://storage:9000"
APP_USER="goclient"
SERVICE_NAME="app-service"

ROOT_USER="${MINIO_ROOT_USER:-minio}"
ROOT_PASS="${MINIO_ROOT_PASSWORD:-hackme123}"

mkdir -p "$ENV_DIR"

tries=0
until mc --quiet alias set "$MC_ALIAS" "$MC_ENDPOINT" "$ROOT_USER" "$ROOT_PASS" || [ $tries -ge 30 ]; do
  tries=$((tries+1))
  echo "Waiting for minio... ($tries)"
  sleep 1
done

# create app user if missing
if ! mc admin user info "$MC_ALIAS" "$APP_USER" >/dev/null 2>&1; then
  mc admin user add "$MC_ALIAS" "$APP_USER" "temporary-password-$(date +%s)"
fi

# granting access to read/write operations
mc admin policy attach "$MC_ALIAS" readwrite --user="$APP_USER"

# create service account and capture output
SA_JSON=$(mc admin user svcacct add "$MC_ALIAS" "$APP_USER" --name "$SERVICE_NAME" --json 2>&1) || {
  echo "Failed to create service account: $SA_JSON"
  exit 2
}

ACCESS_KEY=$(echo "$SA_JSON" | jq -r '.response.accessKey // .accessKey' 2>/dev/null || true)
SECRET_KEY=$(echo "$SA_JSON" | jq -r '.response.secretKey // .secretKey' 2>/dev/null || true)

if [ -z "$ACCESS_KEY" ] || [ -z "$SECRET_KEY" ]; then
  ACCESS_KEY=$(echo "$SA_JSON" | grep -Eo 'AccessKey[: ]+[A-Za-z0-9_-]+' | awk '{print $2}' || true)
  SECRET_KEY=$(echo "$SA_JSON" | grep -Eo 'SecretKey[: ]+[A-Za-z0-9_-]+' | awk '{print $2}' || true)
fi

if [ -z "$ACCESS_KEY" ] || [ -z "$SECRET_KEY" ]; then
  echo "Parsing keys failed. Output:"
  echo "$SA_JSON"
  exit 3
fi

echo "$ACCESS_KEY"
echo "$SECRET_KEY"

# write .env file for compose
cat > "$ENV_FILE" <<EOF
MINIO_ACCESS_KEY=$ACCESS_KEY
MINIO_SECRET_KEY=$SECRET_KEY
EOF

echo "Wrote $ENV_FILE"
