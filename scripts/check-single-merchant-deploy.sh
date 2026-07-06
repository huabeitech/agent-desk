#!/usr/bin/env bash
set -euo pipefail

config_file="${1:-docker/agent-desk.yaml}"
compose_file="${2:-docker-compose.yml}"
env_file="${AGENT_DESK_ENV_FILE:-.env.production}"

errors=0
warnings=0

fail() {
  errors=$((errors + 1))
  printf 'FAIL: %s\n' "$1"
}

warn() {
  warnings=$((warnings + 1))
  printf 'WARN: %s\n' "$1"
}

ok() {
  printf 'OK: %s\n' "$1"
}

yaml_section_value() {
  local section="$1"
  local key="$2"
  awk -v section="$section" -v key="$key" '
    /^[A-Za-z0-9_]+:/ {
      current = $1
      sub(":", "", current)
      next
    }
    current == section && $0 ~ "^[[:space:]]+" key ":" {
      line = $0
      sub("^[^:]+:[[:space:]]*", "", line)
      sub("[[:space:]]+#.*$", "", line)
      gsub(/^[ \t"]+|[ \t"]+$/, "", line)
      print line
      exit
    }
  ' "$config_file"
}

yaml_nested_value() {
  local first="$1"
  local second="$2"
  local key="$3"
  awk -v first="$first" -v second="$second" -v key="$key" '
    /^[A-Za-z0-9_]+:/ {
      current = $1
      sub(":", "", current)
      nested = ""
      next
    }
    current == first && $0 ~ "^[[:space:]]{2}" second ":" {
      nested = second
      next
    }
    current == first && nested == second && $0 ~ "^[[:space:]]{4}" key ":" {
      line = $0
      sub("^[^:]+:[[:space:]]*", "", line)
      sub("[[:space:]]+#.*$", "", line)
      gsub(/^[ \t"]+|[ \t"]+$/, "", line)
      print line
      exit
    }
  ' "$config_file"
}

is_blank_or_placeholder() {
  local value
  value="$(printf '%s' "${1:-}" | tr '[:upper:]' '[:lower:]' | xargs)"
  case "$value" in
    ""|changeme|change-me|replace-me|replace-with-a-random-secret|please-change|your-secret|secret)
      return 0
      ;;
  esac
  return 1
}

env_file_value() {
  local key="$1"
  [ -f "$env_file" ] || return 0
  awk -v key="$key" '
    $0 ~ "^[[:space:]]*#" { next }
    $0 ~ "^[[:space:]]*" key "=" {
      line=$0
      sub("^[[:space:]]*" key "=", "", line)
      sub("[[:space:]]+#.*$", "", line)
      gsub(/^[ \t'\''"]+|[ \t'\''"]+$/, "", line)
      print line
      exit
    }
  ' "$env_file"
}

check_env_file_value() {
  local key="$1"
  local label="$2"
  local value
  value="$(env_file_value "$key")"
  if is_blank_or_placeholder "$value"; then
    fail "${env_file} 中 ${label}（${key}）为空或仍是占位值"
  else
    ok "${env_file} 中 ${label} 已填写"
  fi
}

effective_env_value() {
  local key="$1"
  local value="${!key:-}"
  if [ -n "$value" ]; then
    printf '%s' "$value"
    return
  fi
  env_file_value "$key"
}

if [ ! -f "$config_file" ]; then
  fail "配置文件不存在：$config_file"
else
  ok "找到配置文件：$config_file"
fi

if [ -f "$compose_file" ]; then
  ok "找到 compose 文件：$compose_file"
else
  warn "未找到 compose 文件：$compose_file；仅检查配置文件"
fi

if [ -f "$env_file" ]; then
 ok "找到环境变量文件：$env_file"
  check_env_file_value AGENT_DESK_BOOTSTRAP_ADMIN_PASSWORD "首次管理员密码"
  check_env_file_value AGENT_DESK_CUSTOMERSESSION_SECRET "客户聊天密钥"
  if [ -f "$compose_file" ] && grep -Eq 'AGENT_DESK_MYSQL_PASSWORD' "$compose_file"; then
    check_env_file_value AGENT_DESK_MYSQL_PASSWORD "MySQL 应用密码"
    check_env_file_value AGENT_DESK_MYSQL_ROOT_PASSWORD "MySQL root 密码"
  fi
else
  warn "未找到环境变量文件：${env_file}；建议从 .env.example 复制并为每个商家单独填写"
fi

if [ -f "$config_file" ]; then
  customer_secret="$(effective_env_value AGENT_DESK_CUSTOMERSESSION_SECRET)"
  if [ -z "$customer_secret" ]; then
    customer_secret="$(yaml_section_value customerSession secret)"
  fi
  if is_blank_or_placeholder "$customer_secret"; then
    fail "customerSession.secret 仍为空或占位值；请设置 AGENT_DESK_CUSTOMERSESSION_SECRET 或写入独立随机值"
  else
    ok "customerSession.secret 已配置"
  fi

  db_type="$(yaml_section_value db type)"
  db_dsn="$(effective_env_value AGENT_DESK_DB_DSN)"
  mysql_password="$(effective_env_value AGENT_DESK_MYSQL_PASSWORD)"
  if [ -z "$db_dsn" ]; then
    db_dsn="$(yaml_section_value db dsn)"
  fi
  if is_blank_or_placeholder "$db_dsn"; then
    fail "db.dsn 为空；请配置商家独立数据库"
  elif printf '%s' "$db_dsn" | grep -Eq 'cs_ai_agent_password|root_password|ChangeMe123'; then
    if [ -f "$compose_file" ] && grep -q 'AGENT_DESK_DB_DSN' "$compose_file" && ! is_blank_or_placeholder "$mysql_password"; then
      ok "compose 会用 AGENT_DESK_MYSQL_PASSWORD 覆盖演示 DSN"
    else
      fail "db.dsn 仍包含演示密码；请为该商家设置独立数据库密码"
    fi
  else
    ok "数据库 DSN 未发现默认演示密码"
  fi

  if [ "$db_type" = "sqlite" ]; then
    warn "当前使用 SQLite；小型单店可用，正式高并发或多人后台建议改 MySQL"
  elif [ "$db_type" = "mysql" ]; then
    ok "数据库类型为 MySQL"
  else
    warn "数据库类型为 ${db_type:-未设置}；请确认运行环境支持"
  fi

  vector_type="$(yaml_section_value vectorDB type)"
  if [ "$vector_type" = "qdrant" ] || [ "$vector_type" = "lancedb" ]; then
    ok "向量库类型为 $vector_type"
  else
    fail "vectorDB.type 未配置为 qdrant 或 lancedb"
  fi

  if grep -Eq 'http://(127\.0\.0\.1|localhost):8083' "$config_file"; then
    warn "CORS 仍包含 localhost；正式域名上线前请改为商家后台域名和嵌入站点域名"
  fi

  webhook_enabled="$(effective_env_value AGENT_DESK_NOTIFY_WEBHOOK_ENABLED)"
  webhook_url="$(effective_env_value AGENT_DESK_NOTIFY_WEBHOOK_URL)"
  if [ -z "$webhook_enabled" ]; then
    webhook_enabled="$(yaml_nested_value notify webhook enabled)"
  fi
  if [ -z "$webhook_url" ]; then
    webhook_url="$(yaml_nested_value notify webhook url)"
  fi
  wxwork_notify_enabled="$(awk '
    $0 ~ /^wxWork:/ { current="wxWork"; nested=""; next }
    current == "wxWork" && $0 ~ /^[A-Za-z0-9_]+:/ { current=""; nested=""; next }
    current == "wxWork" && $0 ~ /^[[:space:]]{2}notify:/ { nested="notify"; next }
    current == "wxWork" && nested == "notify" && $0 ~ /^[[:space:]]{4}enabled:/ {
      line=$0; sub("^[^:]+:[[:space:]]*", "", line); gsub(/^[ \t"]+|[ \t"]+$/, "", line); print line; exit
    }
  ' "$config_file")"
  if [ "$webhook_enabled" = "true" ] && ! is_blank_or_placeholder "$webhook_url"; then
    ok "Webhook 外部通知已配置"
  elif [ "$wxwork_notify_enabled" = "true" ]; then
    ok "企业微信应用通知已启用"
  else
    warn "未启用 Webhook 或企业微信通知；高意向线索和转人工只能依赖站内通知"
  fi
fi

bootstrap_password="$(effective_env_value AGENT_DESK_BOOTSTRAP_ADMIN_PASSWORD)"
if is_blank_or_placeholder "$bootstrap_password"; then
  fail "未设置 AGENT_DESK_BOOTSTRAP_ADMIN_PASSWORD；首次初始化会使用默认 admin 密码"
else
  ok "首次管理员密码环境变量已设置"
fi

if [ -f "$compose_file" ]; then
  mysql_password="$(effective_env_value AGENT_DESK_MYSQL_PASSWORD)"
  mysql_root_password="$(effective_env_value AGENT_DESK_MYSQL_ROOT_PASSWORD)"
  if grep -Eq 'cs_ai_agent_password|cs_ai_agent_root_password' "$compose_file" && (is_blank_or_placeholder "$mysql_password" || is_blank_or_placeholder "$mysql_root_password"); then
    fail "compose 仍会回退到演示 MySQL 密码；请设置 AGENT_DESK_MYSQL_PASSWORD 和 AGENT_DESK_MYSQL_ROOT_PASSWORD"
  else
    ok "compose 数据库密码未使用默认回退值，或已由环境变量覆盖"
  fi

  if ! grep -q 'AGENT_DESK_BOOTSTRAP_ADMIN_PASSWORD' "$compose_file"; then
    warn "compose 未透传 AGENT_DESK_BOOTSTRAP_ADMIN_PASSWORD；容器首次初始化可能仍使用默认密码"
  fi
  if ! grep -q 'AGENT_DESK_CUSTOMERSESSION_SECRET' "$compose_file"; then
    warn "compose 未透传 AGENT_DESK_CUSTOMERSESSION_SECRET；容器内 customerSession.secret 可能为空"
  fi
fi

printf '\n检查完成：%d 个失败，%d 个警告。\n' "$errors" "$warnings"
if [ "$errors" -gt 0 ]; then
  exit 1
fi
