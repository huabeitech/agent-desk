#!/usr/bin/env bash
set -euo pipefail

backup_dir=""
compose_file="docker-compose.yml"
project_dir="$(pwd)"
dry_run=0
confirmed=0
restore_config=0
skip_mysql=0
skip_data=0

usage() {
  cat <<'EOF'
Usage:
  scripts/restore-single-merchant.sh --backup-dir backups/20260101-120000 [options]

Options:
  --backup-dir DIR       Backup directory created by scripts/backup-single-merchant.sh
  --compose FILE         Compose file to use for MySQL restore (default: docker-compose.yml)
  --project-dir DIR      Project directory to restore into (default: current directory)
  --restore-config       Restore config/config.yaml and docker/ snapshots when present
  --skip-mysql           Do not import mysql.sql
  --skip-data            Do not restore data.tar.gz
  --dry-run              Print planned actions without changing files or database
  --confirm              Required for non-dry-run restore

Environment:
  AGENT_DESK_MYSQL_PASSWORD       MySQL app user password for docker compose mysql import

Safety notes:
  - Stop the app before restoring local ./data to avoid partially written files.
  - Restoring MySQL imports into database cs_ai_agent as user cs_ai_agent.
  - --restore-config can overwrite local config and docker deployment snapshots.
EOF
}

while [ "$#" -gt 0 ]; do
  case "$1" in
    --backup-dir)
      backup_dir="${2:?missing --backup-dir value}"
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
    --restore-config)
      restore_config=1
      shift
      ;;
    --skip-mysql)
      skip_mysql=1
      shift
      ;;
    --skip-data)
      skip_data=1
      shift
      ;;
    --dry-run)
      dry_run=1
      shift
      ;;
    --confirm)
      confirmed=1
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

if [ -z "$backup_dir" ]; then
  echo "--backup-dir is required" >&2
  usage >&2
  exit 2
fi

cd "$project_dir"

if [ ! -d "$backup_dir" ]; then
  echo "Backup directory not found: $backup_dir" >&2
  exit 1
fi

if [ "$dry_run" != "1" ] && [ "$confirmed" != "1" ]; then
  echo "Refusing to restore without --confirm. Re-run with --dry-run first." >&2
  exit 1
fi

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

restore_mysql() {
  if [ "$skip_mysql" = "1" ]; then
    note "Skipping MySQL restore by request"
    return
  fi
  if [ ! -f "$backup_dir/mysql.sql" ]; then
    note "No mysql.sql found; skipping MySQL restore"
    return
  fi
  if ! has_compose_service mysql; then
    note "Compose service mysql not found; skipping MySQL restore"
    return
  fi
  if ! compose_service_running mysql; then
    echo "MySQL compose service is not running; start it before restoring mysql.sql" >&2
    exit 1
  fi
  if [ -z "${AGENT_DESK_MYSQL_PASSWORD:-}" ]; then
    echo "AGENT_DESK_MYSQL_PASSWORD is required to restore mysql.sql" >&2
    exit 1
  fi
  note "Restoring MySQL database from $backup_dir/mysql.sql"
  if [ "$dry_run" = "1" ]; then
    note "[dry-run] docker compose -f $compose_file exec -T mysql mysql -ucs_ai_agent -p******** cs_ai_agent < $backup_dir/mysql.sql"
  else
    docker compose -f "$compose_file" exec -T mysql \
      mysql -ucs_ai_agent -p"$AGENT_DESK_MYSQL_PASSWORD" cs_ai_agent < "$backup_dir/mysql.sql"
  fi
}

restore_data() {
  if [ "$skip_data" = "1" ]; then
    note "Skipping local data restore by request"
    return
  fi
  if [ ! -f "$backup_dir/data.tar.gz" ]; then
    note "No data.tar.gz found; skipping local data restore"
    return
  fi
  note "Restoring local data directory from $backup_dir/data.tar.gz"
  if [ -e data ]; then
    run mv data "data.before-restore-$(date +%Y%m%d-%H%M%S)"
  fi
  run tar -xzf "$backup_dir/data.tar.gz"
}

restore_config_snapshots() {
  if [ "$restore_config" != "1" ]; then
    note "Skipping config snapshots; pass --restore-config to restore them"
    return
  fi
  if [ -f "$backup_dir/config/config.yaml" ]; then
    note "Restoring config/config.yaml"
    run mkdir -p config
    if [ -f config/config.yaml ]; then
      run cp config/config.yaml "config/config.yaml.before-restore-$(date +%Y%m%d-%H%M%S)"
    fi
    run cp "$backup_dir/config/config.yaml" config/config.yaml
  fi
  if [ -f "$backup_dir/docker-config.tar.gz" ]; then
    note "Restoring docker config directory"
    if [ -d docker ]; then
      run mv docker "docker.before-restore-$(date +%Y%m%d-%H%M%S)"
    fi
    run tar -xzf "$backup_dir/docker-config.tar.gz"
  fi
  if [ -f "$backup_dir/$(basename "$compose_file")" ]; then
    note "Restoring compose file snapshot: $(basename "$compose_file")"
    if [ -f "$compose_file" ]; then
      run cp "$compose_file" "$compose_file.before-restore-$(date +%Y%m%d-%H%M%S)"
    fi
    run cp "$backup_dir/$(basename "$compose_file")" "$compose_file"
  fi
}

note "Restore source: $backup_dir"
note "Project dir: $(pwd)"
note "Compose file: $compose_file"

restore_mysql
restore_data
restore_config_snapshots

if [ "$dry_run" = "1" ]; then
  note "Dry run complete. No data was changed."
else
  note "Restore complete. Run health checks and acceptance tests before reopening traffic."
fi
