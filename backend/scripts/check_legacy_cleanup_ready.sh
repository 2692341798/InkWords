#!/usr/bin/env bash
# shellcheck disable=SC2016 # Quoted variables below must expand inside the database container.

set -Eeuo pipefail

fail() {
  printf 'legacy-cleanup-preflight: %s\n' "$*" >&2
  exit 1
}

if [[ $# -ne 1 ]]; then
  fail "usage: $0 /absolute/verified/backup-directory"
fi

requested_backup=$1
[[ "$requested_backup" == /* ]] || fail "backup directory must be an absolute path"
[[ "$requested_backup" != "/" ]] || fail "refusing to use filesystem root as backup directory"

script_dir=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd -P)
repo_root=$(cd "$script_dir/../.." && pwd -P)
core_snapshot_sql="$script_dir/sql/backup_core_snapshot.sql"
review_snapshot_sql="$script_dir/sql/backup_review_snapshot.sql"
[[ -f "$core_snapshot_sql" ]] || fail "missing core backup snapshot SQL"
[[ -f "$review_snapshot_sql" ]] || fail "missing review backup snapshot SQL"
case "$requested_backup" in
  "$repo_root"|"$repo_root"/*)
    fail "backup directory must stay outside the repository"
    ;;
esac
[[ -d "$requested_backup" ]] || fail "backup directory does not exist"
backup=$(cd "$requested_backup" && pwd -P)
case "$backup" in
  "$repo_root"|"$repo_root"/*)
    fail "backup directory must stay outside the repository"
    ;;
esac

required_files=(
  manifest.txt
  core.dump
  review.dump
  core-source-counts.txt
  core-restored-counts.txt
  review-source-counts.txt
  review-restored-counts.txt
  images.txt
)
for filename in "${required_files[@]}"; do
  [[ -f "$backup/$filename" && ! -L "$backup/$filename" ]] || fail "missing regular backup file: $filename"
done

manifest_value() {
  local key=$1
  local count value
  count=$(awk -F= -v key="$key" '$1 == key { count++ } END { print count + 0 }' "$backup/manifest.txt")
  [[ "$count" == "1" ]] || fail "manifest must contain exactly one $key entry"
  value=$(awk -F= -v key="$key" '$1 == key { print substr($0, length($1) + 2) }' "$backup/manifest.txt")
  [[ -n "$value" ]] || fail "manifest value is empty: $key"
  printf '%s' "$value"
}

[[ "$(manifest_value format)" == "inkwords-local-backup.v2" ]] || fail "unsupported backup manifest format"
[[ "$(manifest_value core_file)" == "core.dump" ]] || fail "manifest core file must be core.dump"
[[ "$(manifest_value review_file)" == "review.dump" ]] || fail "manifest review file must be review.dump"
[[ "$(manifest_value images_file)" == "images.txt" ]] || fail "manifest images file must be images.txt"
[[ "$(manifest_value core_restore_check)" == "passed" ]] || fail "core restore check did not pass"
[[ "$(manifest_value review_restore_check)" == "passed" ]] || fail "review restore check did not pass"

core_expected_sha=$(manifest_value core_sha256)
review_expected_sha=$(manifest_value review_sha256)
[[ "$core_expected_sha" =~ ^[a-f0-9]{64}$ ]] || fail "invalid core dump SHA-256"
[[ "$review_expected_sha" =~ ^[a-f0-9]{64}$ ]] || fail "invalid review dump SHA-256"
core_actual_sha=$(shasum -a 256 "$backup/core.dump" | awk '{print $1}')
review_actual_sha=$(shasum -a 256 "$backup/review.dump" | awk '{print $1}')
[[ "$core_actual_sha" == "$core_expected_sha" ]] || fail "core dump SHA-256 does not match the manifest"
[[ "$review_actual_sha" == "$review_expected_sha" ]] || fail "review dump SHA-256 does not match the manifest"
cmp -s "$backup/core-source-counts.txt" "$backup/core-restored-counts.txt" || fail "core restore-check counts do not match"
cmp -s "$backup/review-source-counts.txt" "$backup/review-restored-counts.txt" || fail "review restore-check counts do not match"

env_file="$repo_root/backend/.env"
[[ -f "$env_file" ]] || fail "missing local environment file: $env_file"
command -v docker >/dev/null 2>&1 || fail "docker is required"
docker compose version >/dev/null 2>&1 || fail "docker compose is required"

cd "$repo_root"
compose=(docker compose --env-file "$env_file")
"${compose[@]}" config >/dev/null
running_services=$("${compose[@]}" ps --status running --services)
db_running=false
writers=(core-api llm-stream parser-service export-service review-service course-runner)
running_writers=()
while IFS= read -r running; do
  if [[ "$running" == "db" ]]; then
    db_running=true
  fi
  for writer in "${writers[@]}"; do
    if [[ "$running" == "$writer" ]]; then
      running_writers+=("$writer")
      break
    fi
  done
done <<<"$running_services"
[[ "$db_running" == true ]] || fail "database service is not running"
if [[ ${#running_writers[@]} -gt 0 ]]; then
  fail "writer services must be stopped before cleanup preflight: ${running_writers[*]}"
fi
"${compose[@]}" exec -T db pg_restore --list <"$backup/core.dump" >/dev/null || fail "core dump is not a readable PostgreSQL archive"
"${compose[@]}" exec -T db pg_restore --list <"$backup/review.dump" >/dev/null || fail "review dump is not a readable PostgreSQL archive"

core_backup_counts=$(tr -d '\r\n' <"$backup/core-source-counts.txt")
review_backup_counts=$(tr -d '\r\n' <"$backup/review-source-counts.txt")
core_current_counts=$("${compose[@]}" exec -T db sh -lc 'psql -X -v ON_ERROR_STOP=1 -U "$POSTGRES_USER" -d "$POSTGRES_DB" -At' <"$core_snapshot_sql" | tr -d '\r\n')
review_current_counts=$("${compose[@]}" exec -T db sh -lc 'psql -X -v ON_ERROR_STOP=1 -U "$POSTGRES_USER" -d inkwords_review_db -At' <"$review_snapshot_sql" | tr -d '\r\n')
[[ "$core_current_counts" == "$core_backup_counts" ]] || fail "core cleanup-critical counts changed after the verified backup"
[[ "$review_current_counts" == "$review_backup_counts" ]] || fail "review cleanup-critical counts changed after the verified backup"

core_readiness=$("${compose[@]}" exec -T db sh -lc 'psql -X -v ON_ERROR_STOP=1 -U "$POSTGRES_USER" -d "$POSTGRES_DB" -Atc "SELECT concat_ws(chr(124), (SELECT COUNT(*) FROM local_workspaces), (SELECT COUNT(*) FROM job_tasks WHERE workspace_id IS NULL), (SELECT COUNT(*) FROM blogs WHERE workspace_id IS NULL), (SELECT COUNT(*) FROM o_auth_tokens), (SELECT COUNT(*) FROM project_courses), (SELECT COUNT(*) FROM user_prompt_settings), (SELECT MAX(version_id) FROM inkwords_core_schema_migrations WHERE is_applied))"' | tr -d '\r\n')
IFS='|' read -r workspace_count task_without_workspace blog_without_workspace oauth_count project_course_count prompt_settings_count core_version <<<"$core_readiness"
[[ "$workspace_count" == "1" ]] || fail "expected exactly one installation workspace"
[[ "$task_without_workspace" == "0" ]] || fail "tasks without workspace ownership remain"
[[ "$blog_without_workspace" == "0" ]] || fail "blogs without workspace ownership remain"
[[ "$oauth_count" == "0" ]] || fail "OAuth rows remain"
[[ "$project_course_count" == "0" ]] || fail "ProjectCourse rows remain"
[[ "$prompt_settings_count" == "0" ]] || fail "user prompt setting rows remain"
[[ "$core_version" == "23" ]] || fail "core schema must be exactly migration 23 before cleanup"

review_readiness=$("${compose[@]}" exec -T db sh -lc 'psql -X -v ON_ERROR_STOP=1 -U "$POSTGRES_USER" -d inkwords_review_db -Atc "SELECT concat_ws(chr(124), (SELECT COUNT(*) FROM review_sessions WHERE workspace_id IS NULL), (SELECT MAX(version_id) FROM inkwords_review_schema_migrations WHERE is_applied))"' | tr -d '\r\n')
IFS='|' read -r review_without_workspace review_version <<<"$review_readiness"
[[ "$review_without_workspace" == "0" ]] || fail "review sessions without workspace ownership remain"
[[ "$review_version" == "21" ]] || fail "review schema must be exactly migration 21 before cleanup"

printf 'legacy-cleanup-preflight: ready; verified backup and cleanup-critical counts match\n'
printf 'legacy-cleanup-preflight: core schema=%s review schema=%s workspace=%s\n' "$core_version" "$review_version" "$workspace_count"
