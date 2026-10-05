#!/usr/bin/env bash
# shellcheck disable=SC2016 # Quoted variables below must expand inside the database container.

set -Eeuo pipefail

fail() {
  printf 'backup-local: %s\n' "$*" >&2
  exit 1
}

if [[ $# -ne 1 ]]; then
  fail "usage: $0 /absolute/empty/backup-directory"
fi

requested_target=$1
[[ "$requested_target" == /* ]] || fail "backup directory must be an absolute path"
[[ "$requested_target" != "/" ]] || fail "refusing to use filesystem root as backup directory"

script_dir=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd -P)
repo_root=$(cd "$script_dir/../.." && pwd -P)
core_snapshot_sql="$script_dir/sql/backup_core_snapshot.sql"
review_snapshot_sql="$script_dir/sql/backup_review_snapshot.sql"
[[ -f "$core_snapshot_sql" ]] || fail "missing core backup snapshot SQL"
[[ -f "$review_snapshot_sql" ]] || fail "missing review backup snapshot SQL"
case "$requested_target" in
  "$repo_root"|"$repo_root"/*)
    fail "backup directory must stay outside the repository"
    ;;
esac
env_file="$repo_root/backend/.env"
[[ -f "$env_file" ]] || fail "missing local environment file: $env_file"

command -v docker >/dev/null 2>&1 || fail "docker is required"
docker compose version >/dev/null 2>&1 || fail "docker compose is required"

umask 077
mkdir -p "$requested_target"
target=$(cd "$requested_target" && pwd -P)
case "$target" in
  "$repo_root"|"$repo_root"/*)
    fail "backup directory must stay outside the repository"
    ;;
esac
[[ -z "$(find "$target" -mindepth 1 -maxdepth 1 -print -quit)" ]] || fail "backup directory must be empty"
chmod 700 "$target"

cd "$repo_root"
compose=(docker compose --env-file "$env_file")
"${compose[@]}" config >/dev/null
running_services=$("${compose[@]}" ps --status running --services)
db_running=false
while IFS= read -r running; do
  if [[ "$running" == "db" ]]; then
    db_running=true
    break
  fi
done <<<"$running_services"
[[ "$db_running" == true ]] || fail "database service is not running"

suffix=$(od -An -N6 -tx1 /dev/urandom | tr -d ' \n')
[[ "$suffix" =~ ^[a-f0-9]{12}$ ]] || fail "could not create a safe restore-check suffix"
core_restore_db="inkwords_restore_core_${suffix}"
review_restore_db="inkwords_restore_review_${suffix}"

writers=(core-api llm-stream parser-service export-service review-service course-runner)
active_writers=()
for writer in "${writers[@]}"; do
  while IFS= read -r running; do
    if [[ "$running" == "$writer" ]]; then
      active_writers+=("$writer")
      break
    fi
  done <<<"$running_services"
done
[[ ${#active_writers[@]} -gt 0 ]] || fail "no running writer services were found"

paused=false
core_restore_created=false
review_restore_created=false
cleanup() {
  set +e
  if [[ "$core_restore_created" == true ]]; then
    "${compose[@]}" exec -T -e RESTORE_DB="$core_restore_db" db sh -lc 'dropdb -U "$POSTGRES_USER" --if-exists "$RESTORE_DB"' >/dev/null
  fi
  if [[ "$review_restore_created" == true ]]; then
    "${compose[@]}" exec -T -e RESTORE_DB="$review_restore_db" db sh -lc 'dropdb -U "$POSTGRES_USER" --if-exists "$RESTORE_DB"' >/dev/null
  fi
  if [[ "$paused" == true ]]; then
    "${compose[@]}" unpause "${active_writers[@]}" >/dev/null
  fi
}
trap cleanup EXIT

"${compose[@]}" pause "${active_writers[@]}" >/dev/null
paused=true

core_dump="$target/core.dump"
review_dump="$target/review.dump"
core_source_counts="$target/core-source-counts.txt"
review_source_counts="$target/review-source-counts.txt"
images_file="$target/images.txt"

"${compose[@]}" images >"$images_file"
"${compose[@]}" exec -T db sh -lc 'exec pg_dump -U "$POSTGRES_USER" -Fc "$POSTGRES_DB"' >"$core_dump"
"${compose[@]}" exec -T db sh -lc 'exec pg_dump -U "$POSTGRES_USER" -Fc inkwords_review_db' >"$review_dump"
[[ -s "$core_dump" ]] || fail "core database dump is empty"
[[ -s "$review_dump" ]] || fail "review database dump is empty"
"${compose[@]}" exec -T db sh -lc 'psql -X -v ON_ERROR_STOP=1 -U "$POSTGRES_USER" -d "$POSTGRES_DB" -At' <"$core_snapshot_sql" >"$core_source_counts"
"${compose[@]}" exec -T db sh -lc 'psql -X -v ON_ERROR_STOP=1 -U "$POSTGRES_USER" -d inkwords_review_db -At' <"$review_snapshot_sql" >"$review_source_counts"

"${compose[@]}" unpause "${active_writers[@]}" >/dev/null
paused=false

"${compose[@]}" exec -T -e RESTORE_DB="$core_restore_db" db sh -lc 'createdb -U "$POSTGRES_USER" "$RESTORE_DB"'
core_restore_created=true
"${compose[@]}" exec -T -e RESTORE_DB="$review_restore_db" db sh -lc 'createdb -U "$POSTGRES_USER" "$RESTORE_DB"'
review_restore_created=true

"${compose[@]}" exec -T -e RESTORE_DB="$core_restore_db" db sh -lc 'exec pg_restore -U "$POSTGRES_USER" -d "$RESTORE_DB" --exit-on-error --no-owner --no-privileges' <"$core_dump"
"${compose[@]}" exec -T -e RESTORE_DB="$review_restore_db" db sh -lc 'exec pg_restore -U "$POSTGRES_USER" -d "$RESTORE_DB" --exit-on-error --no-owner --no-privileges' <"$review_dump"

core_restored_counts="$target/core-restored-counts.txt"
review_restored_counts="$target/review-restored-counts.txt"
"${compose[@]}" exec -T -e RESTORE_DB="$core_restore_db" db sh -lc 'psql -X -v ON_ERROR_STOP=1 -U "$POSTGRES_USER" -d "$RESTORE_DB" -At' <"$core_snapshot_sql" >"$core_restored_counts"
"${compose[@]}" exec -T -e RESTORE_DB="$review_restore_db" db sh -lc 'psql -X -v ON_ERROR_STOP=1 -U "$POSTGRES_USER" -d "$RESTORE_DB" -At' <"$review_snapshot_sql" >"$review_restored_counts"

cmp -s "$core_source_counts" "$core_restored_counts" || fail "core restore-check counts do not match the source database"
cmp -s "$review_source_counts" "$review_restored_counts" || fail "review restore-check counts do not match the source database"

core_sha=$(shasum -a 256 "$core_dump" | awk '{print $1}')
review_sha=$(shasum -a 256 "$review_dump" | awk '{print $1}')
created_at=$(date -u +%Y-%m-%dT%H:%M:%SZ)
cat >"$target/manifest.txt" <<EOF
format=inkwords-local-backup.v2
created_at=$created_at
core_file=core.dump
core_sha256=$core_sha
review_file=review.dump
review_sha256=$review_sha
images_file=images.txt
core_restore_check=passed
review_restore_check=passed
EOF

chmod 600 "$target"/*
printf 'backup-local: verified backup created at %s\n' "$target"
printf 'backup-local: core sha256 %s\n' "$core_sha"
printf 'backup-local: review sha256 %s\n' "$review_sha"
