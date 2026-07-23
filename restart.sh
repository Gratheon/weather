#!/bin/sh
set -eu

PROJECT_NAME="${COMPOSE_PROJECT_NAME:-weather}"
COMPOSE_FILE="${COMPOSE_FILE:-docker-compose.yml}"
LEGACY_PROJECT_NAME="${LEGACY_PROJECT_NAME:-gratheon}"
SERVICE_NAME="${SERVICE_NAME:-weather}"
VERIFY_URL="${VERIFY_URL:-http://127.0.0.1:8070/health}"
VERIFY_ATTEMPTS="${VERIFY_ATTEMPTS:-30}"
WEATHER_IMAGE="${WEATHER_IMAGE:-}"

script_dir="$(CDPATH='' cd -- "$(dirname -- "$0")" && pwd)"
cd "$script_dir"

compose() {
  image="$1"
  shift
  COMPOSE_PROJECT_NAME="$PROJECT_NAME" WEATHER_IMAGE="$image" \
    docker-compose -f "$COMPOSE_FILE" "$@"
}

require_immutable_image() {
  if ! printf '%s\n' "$WEATHER_IMAGE" \
      | grep -Eq '^ghcr\.io/gratheon/weather@sha256:[0-9a-f]{64}$'; then
    echo "WEATHER_IMAGE must be an immutable ghcr.io/gratheon/weather digest, got: ${WEATHER_IMAGE:-<unset>}" >&2
    exit 2
  fi
}

current_container() {
  docker ps -q \
    --filter "label=com.docker.compose.project=${PROJECT_NAME}" \
    --filter "label=com.docker.compose.service=${SERVICE_NAME}" \
    | head -n 1
}

stop_legacy_gratheon_service_container() {
  docker ps -aq \
    --filter "label=com.docker.compose.project=${LEGACY_PROJECT_NAME}" \
    --filter "label=com.docker.compose.service=${SERVICE_NAME}" \
    | while IFS= read -r container_id; do
        [ -n "$container_id" ] || continue
        docker rm -f "$container_id" >/dev/null 2>&1 || true
      done
}

verify_deployment() {
  attempt=1
  while [ "$attempt" -le "$VERIFY_ATTEMPTS" ]; do
    if curl --fail --silent --show-error --max-time 5 "$VERIFY_URL" >/dev/null; then
      return 0
    fi
    sleep 2
    attempt=$((attempt + 1))
  done
  return 1
}

require_immutable_image

previous_container="$(current_container)"
previous_image=""
if [ -n "$previous_container" ]; then
  previous_image="$(docker inspect "$previous_container" --format '{{.Config.Image}}')"
fi

# WHY: production may pull and activate an image, but must never build one.
compose "$WEATHER_IMAGE" pull "$SERVICE_NAME"
stop_legacy_gratheon_service_container

if ! compose "$WEATHER_IMAGE" up -d --no-build --force-recreate "$SERVICE_NAME"; then
  if [ -n "$previous_image" ]; then
    compose "$previous_image" up -d --no-build --force-recreate "$SERVICE_NAME" || true
  fi
  exit 1
fi

if ! verify_deployment; then
  echo "Deployment health check failed for $WEATHER_IMAGE" >&2
  if [ -n "$previous_image" ]; then
    echo "Rolling back to $previous_image" >&2
    compose "$previous_image" up -d --no-build --force-recreate "$SERVICE_NAME"
    verify_deployment || true
  fi
  exit 1
fi

printf 'Deployed %s\n' "$WEATHER_IMAGE"
