#!/usr/bin/env bash
set -euo pipefail

output_root="backups"
compose_file="docker-compose.yml"
project_dir="$(pwd)"
dry_run=0

usage() {
  cat <<'EOF'
Usage:
  scripts/backup-single-merchant.sh [--output backups] [--compose docker-compose.yml] [--project-dir .] [--dry-run]

Creates a timestamped backup directory containing:
  - MySQL dump when a mysql compose service is present and running
  - local data directory snapshot when ./data exists
  - docker/config yaml snapshots useful for recovery

Environment:
  AGENT_DESK_MYSQL_PASSWORD       MySQL app user password, used for docker compose mysql dump
  AGENT_DESK_BACKUP_TIMESTAMP     Optional timestamp override for repeatable automation
EOF
}

while [ "$#" -gt 0 ]; do
  case "$1" in
    --output)
      output_root="${2:?missing --output value}"
      shift 2
      ;;
    --compose)
      compose_file="${2:?missing --compose value}"
      shift 2
      ;;
    --project-dir)
      project_dir="${2:?missing --project-dir value}"
      shift 2
      ;;
    --dry-run)
      dry_run=1
      shift
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "Unknown argument: $1" >&2
      usage >&2
      exit 2
      ;;
  esac
done

cd "$project_dir"

timestamp="${AGENT_DESK_BACKUP_TIMESTAMP:-$(date +%Y%m%d-%H%M%S)}"
backup_dir="$output_root/$timestamp"

run() {
  if [ "$dry_run" = "1" ]; then
    printf '[dry-run] %q' "$1"
    shift
    for arg in "$@"; do
      printf ' %q' "$arg"
    done
    printf '\n'
    return 0
  fi
  "$@"
}

note() {
  printf '%s\n' "$1"
}

has_compose_service() {
  local service="$1"
  if [ "$dry_run" = "1" ]; then
    [ -f "$compose_file" ] && grep -Eq "^[[:space:]]{2}${service}:" "$compose_file"
    return
  fi
  [ -f "$compose_file" ] && docker compose -f "$compose_file" config --services 2>/dev/null | grep -qx "$service"
}

compose_service_running() {
  local service="$1"
  if [ "$dry_run" = "1" ]; then
    return 0
  fi
  [ -n "$(docker compose -f "$compose_file" ps -q "$service" 2>/dev/null || true)" ]
}

run mkdir -p "$backup_dir"

if [ "$dry_run" != "1" ]; then
  {
    printf 'timestamp=%s\n' "$timestamp"
    printf 'project_dir=%s\n' "$(pwd)"
    printf 'compose_file=%s\n' "$compose_file"
    printf 'created_at=%s\n' "$(date -u +%Y-%m-%dT%H:%M:%SZ)"
  } > "$backup_dir/BACKUP-MANIFEST.txt"
fi

if [ -f "$compose_file" ]; then
  note "Backing up compose file: $compose_file"
  run cp "$compose_file" "$backup_dir/$(basename "$compose_file")"
fi

if [ -d docker ]; then
  note "Backing up docker config directory"
  run tar -czf "$backup_dir/docker-config.tar.gz" docker
fi

if [ -f config/config.yaml ]; then
  note "Backing up config/config.yaml"
  run mkdir -p "$backup_dir/config"
  run cp config/config.yaml "$backup_dir/config/config.yaml"
fi

if has_compose_service mysql; then
  if compose_service_running mysql; then
    if [ "$dry_run" != "1" ] && [ -z "${AGENT_DESK_MYSQL_PASSWORD:-}" ]; then
      echo "AGENT_DESK_MYSQL_PASSWORD is required to dump the mysql compose service" >&2
      exit 1
    fi
    note "Dumping MySQL database from compose service mysql"
    if [ "$dry_run" = "1" ]; then
      note "[dry-run] docker compose -f $compose_file exec -T mysql mysqldump -ucs_ai_agent -p******** cs_ai_agent > $backup_dir/mysql.sql"
    else
      docker compose -f "$compose_file" exec -T mysql \
        mysqldump -ucs_ai_agent -p"$AGENT_DESK_MYSQL_PASSWORD" cs_ai_agent > "$backup_dir/mysql.sql"
    fi
  else
    note "MySQL compose service exists but is not running; skipping mysql dump"
  fi
fi

if [ -d data ]; then
  note "Backing up local data directory"
  run tar -czf "$backup_dir/data.tar.gz" data
else
  note "No local ./data directory found; named Docker volumes require provider-level volume backup if not mounted locally"
fi

if [ "$dry_run" = "1" ]; then
  note "Dry run complete. Planned backup directory: $backup_dir"
else
  note "Backup complete: $backup_dir"
fi
