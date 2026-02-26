#!/usr/bin/env bash
# Офлайн-выгрузка зависимостей в локальный GOPROXY (file://.cache/gomod)
set -euo pipefail

BASE_URL=${BASE_URL:-https://proxy.golang.org}
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
MODCACHE="${ROOT}/.cache/gomod"

mods=(
  "github.com/shirou/gopsutil/v3@v3.24.1"
  "github.com/google/go-cmp@v0.6.0"
  "github.com/tklauser/go-sysconf@v0.3.12"
  "github.com/tklauser/numcpus@v0.6.1"
  "github.com/lufia/plan9stats@v0.0.0-20211012122336-39d0f177ccd0"
  "github.com/power-devops/perfstat@v0.0.0-20210106213030-5aafc221ea8c"
  "github.com/shoenig/go-m1cpu@v0.1.6"
  "golang.org/x/sys@v0.18.0"
  "github.com/stretchr/testify@v1.8.4"
  "github.com/yusufpapurcu/wmi@v1.2.3"
  "github.com/cpuguy83/go-md2man/v2@v2.0.4"
  "github.com/russross/blackfriday/v2@v2.1.0"
  "github.com/shurcooL/sanitized_anchor_name@v1.0.0"
  "github.com/spf13/cobra@v1.8.1"
  "github.com/spf13/pflag@v1.0.5"
  "github.com/inconshreveable/mousetrap@v1.1.0"
  "github.com/mattn/go-sqlite3@v1.14.22"
  "gopkg.in/yaml.v3@v3.0.1"
)

fetch_one() {
  local modver="$1"
  local mod="${modver%@*}"
  local ver="${modver##*@}"
  local mod_proxy
  mod_proxy="$(printf '%s' "${mod}" | perl -pe 's/([A-Z])/"!".lc($1)/ge')"
  local dir="${MODCACHE}/${mod_proxy}/@v"
  mkdir -p "${dir}"
  for ext in zip mod info; do
    local url="${BASE_URL}/${mod_proxy}/@v/${ver}.${ext}"
    local dst="${dir}/${ver}.${ext}"
    if [[ -f "${dst}" ]]; then
      echo "skip ${modver}.${ext} (exists)"
      continue
    fi
    echo "fetch ${url}"
    env -i curl -fsS --retry 3 --retry-delay 1 -L "${url}" -o "${dst}"
  done
  echo "${ver}" > "${dir}/list"
}

for m in "${mods[@]}"; do
  fetch_one "${m}"
done

echo "Done. Set GOPROXY=file://${MODCACHE},off GOSUMDB=off"
