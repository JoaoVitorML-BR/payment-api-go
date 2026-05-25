#!/usr/bin/env sh

set -eu

COMPOSE_SERVICE="${COMPOSE_SERVICE:-postgres}"

docker compose up -d "$COMPOSE_SERVICE" >/dev/null

DB_USER="$(docker compose exec -T "$COMPOSE_SERVICE" printenv POSTGRES_USER | tr -d '\r')"
DB_NAME="$(docker compose exec -T "$COMPOSE_SERVICE" printenv POSTGRES_DB | tr -d '\r')"
DB_PASSWORD="$(docker compose exec -T "$COMPOSE_SERVICE" printenv POSTGRES_PASSWORD | tr -d '\r')"

if [ -z "$DB_USER" ] || [ -z "$DB_NAME" ] || [ -z "$DB_PASSWORD" ]; then
  echo "Could not read Postgres credentials from the running container." >&2
  exit 1
fi

MIGRATIONS_DIR="$(cd "$(dirname "$0")/../internal/infra/database/migrations" && pwd)"

for file in "$MIGRATIONS_DIR"/*.sql; do
  [ -e "$file" ] || continue
  echo "Applying $(basename "$file")..."
  docker compose exec -T "$COMPOSE_SERVICE" sh -lc "set -e; PGPASSWORD='$DB_PASSWORD' psql -U '$DB_USER' -d '$DB_NAME' -v ON_ERROR_STOP=1 -f -" < "$file"
done

echo "All migrations applied."