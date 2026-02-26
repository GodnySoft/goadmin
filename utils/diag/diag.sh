#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'USAGE'
Usage: diag.sh [--output DIR] [--textfile-dir DIR] [--quiet]

Collects host diagnostics and prepares Prometheus textfile metrics.

Options:
  --output DIR        Output directory for logs (default: ./out)
  --textfile-dir DIR  Prometheus textfile collector dir (optional)
  --quiet             Reduce console output
  -h, --help          Show this help

Environment variables:
  OUTPUT_DIR          Same as --output
  PROM_TEXTFILE_DIR   Same as --textfile-dir
USAGE
}

OUTPUT_DIR="${OUTPUT_DIR:-./out}"
PROM_TEXTFILE_DIR="${PROM_TEXTFILE_DIR:-}"
QUIET=0

while [[ $# -gt 0 ]]; do
  case "$1" in
    --output)
      OUTPUT_DIR="$2"; shift 2 ;;
    --textfile-dir)
      PROM_TEXTFILE_DIR="$2"; shift 2 ;;
    --quiet)
      QUIET=1; shift ;;
    -h|--help)
      usage; exit 0 ;;
    *)
      echo "Unknown arg: $1" >&2
      usage >&2
      exit 2
      ;;
  esac
done

log() {
  if [[ "$QUIET" -eq 0 ]]; then
    echo "$@"
  fi
}

have() { command -v "$1" >/dev/null 2>&1; }

TS="$(date +%Y%m%d_%H%M%S)"
RUN_DIR="${OUTPUT_DIR%/}/run-${TS}"
LATEST_LINK="${OUTPUT_DIR%/}/latest"

mkdir -p "$RUN_DIR"
ln -sfn "$(basename "$RUN_DIR")" "$LATEST_LINK"

log "Output: $RUN_DIR"

# Helpers
write_cmd() {
  local name="$1"; shift
  local file="$RUN_DIR/$name"
  {
    echo "# cmd: $*"
    echo "# ts: $(date -Is)"
    echo
    "$@" || true
  } > "$file" 2>&1
}

write_file() {
  local name="$1"; shift
  local file="$RUN_DIR/$name"
  "$@" > "$file" 2>&1 || true
}

# Core system info
write_cmd "uname.txt" uname -a

if [[ -f /etc/os-release ]]; then
  write_file "os-release.txt" cat /etc/os-release
fi

write_cmd "uptime.txt" uptime
write_cmd "date.txt" date -Is

# CPU / Memory
if have lscpu; then
  write_cmd "cpu.txt" lscpu
else
  write_cmd "cpu.txt" cat /proc/cpuinfo
fi

if have free; then
  write_cmd "memory.txt" free -h
else
  write_cmd "memory.txt" cat /proc/meminfo
fi

# Disks / FS
write_cmd "lsblk.txt" lsblk -a
write_cmd "df.txt" df -hT
write_cmd "mount.txt" mount

# Network
write_cmd "ip_addr.txt" ip -br addr
write_cmd "ip_link.txt" ip -br link
write_cmd "ip_route.txt" ip route
write_cmd "resolv.conf" cat /etc/resolv.conf

if have ss; then
  write_cmd "sockets.txt" ss -tulpen
fi

# Processes
if have ps; then
  write_cmd "ps_top.txt" ps auxfww --sort=-%cpu | head -n 200
  write_cmd "ps_mem.txt" ps auxfww --sort=-%mem | head -n 200
fi

# systemd
if have systemctl; then
  write_cmd "systemd_failed_units.txt" systemctl --failed
  write_cmd "systemd_timers.txt" systemctl list-timers --all
  write_cmd "systemd_services.txt" systemctl list-units --type=service --all
fi

# Logs (safe subset)
if have journalctl; then
  write_cmd "journal_last_1h.txt" journalctl -S -1h --no-pager
  write_cmd "journal_last_boot_errors.txt" journalctl -b -p err --no-pager
fi

# Package info (best effort)
if have apt; then
  write_cmd "apt_updates.txt" bash -c "apt list --upgradable 2>/dev/null"
fi

# Security baseline
if have ufw; then
  write_cmd "ufw_status.txt" ufw status verbose
fi

if have aa-status; then
  write_cmd "apparmor_status.txt" aa-status
fi

# Prometheus metrics (textfile collector format)
START_TS=$(date +%s)

HOSTNAME=$(hostname 2>/dev/null || echo unknown)
OS_ID="unknown"
OS_VER="unknown"
if [[ -f /etc/os-release ]]; then
  OS_ID=$(grep -E '^ID=' /etc/os-release | head -n1 | cut -d= -f2 | tr -d '"')
  OS_VER=$(grep -E '^VERSION_ID=' /etc/os-release | head -n1 | cut -d= -f2 | tr -d '"')
fi
KERNEL=$(uname -r || echo unknown)

FAILED_UNITS=0
if have systemctl; then
  FAILED_UNITS=$(systemctl --failed --no-legend 2>/dev/null | grep -c . || true)
fi

DURATION=$(( $(date +%s) - START_TS ))

PROM_FILE="$RUN_DIR/diag.prom"
cat > "$PROM_FILE" <<'PROMEOF'
# HELP host_diag_info Host diagnostic info
# TYPE host_diag_info gauge
host_diag_info{hostname="HOSTNAME",os_id="OS_ID",os_version="OS_VER",kernel="KERNEL"} 1
# HELP host_diag_failed_systemd_units Number of failed systemd units
# TYPE host_diag_failed_systemd_units gauge
host_diag_failed_systemd_units FAILED_UNITS
# HELP host_diag_last_run_timestamp_seconds Last run time of diagnostics
# TYPE host_diag_last_run_timestamp_seconds gauge
host_diag_last_run_timestamp_seconds LAST_RUN_TS
# HELP host_diag_duration_seconds Duration of diagnostics run
# TYPE host_diag_duration_seconds gauge
host_diag_duration_seconds DURATION
PROMEOF

# Replace placeholders safely
sed -i \
  -e "s/HOSTNAME/${HOSTNAME}/g" \
  -e "s/OS_ID/${OS_ID}/g" \
  -e "s/OS_VER/${OS_VER}/g" \
  -e "s/KERNEL/${KERNEL}/g" \
  -e "s/FAILED_UNITS/${FAILED_UNITS}/g" \
  -e "s/LAST_RUN_TS/$(date +%s)/g" \
  -e "s/DURATION/${DURATION}/g" \
  "$PROM_FILE"

if [[ -n "$PROM_TEXTFILE_DIR" ]]; then
  mkdir -p "$PROM_TEXTFILE_DIR"
  tmpfile="${PROM_TEXTFILE_DIR%/}/.diag.prom.$$"
  cp "$PROM_FILE" "$tmpfile"
  mv "$tmpfile" "${PROM_TEXTFILE_DIR%/}/diag.prom"
  log "Prometheus textfile written: ${PROM_TEXTFILE_DIR%/}/diag.prom"
fi

# Consolidated report for sysadmins
REPORT_FILE="$RUN_DIR/report.txt"
{
  echo "Host Diagnostics Report"
  echo "Generated: $(date -Is)"
  echo "Host: ${HOSTNAME}"
  echo "Output: ${RUN_DIR}"
  echo
} > "$REPORT_FILE"

append_report() {
  local file="$1"
  echo "===== ${file##*/} =====" >> "$REPORT_FILE"
  cat "$file" >> "$REPORT_FILE"
  echo >> "$REPORT_FILE"
}

for f in \
  "$RUN_DIR/uname.txt" \
  "$RUN_DIR/os-release.txt" \
  "$RUN_DIR/uptime.txt" \
  "$RUN_DIR/date.txt" \
  "$RUN_DIR/cpu.txt" \
  "$RUN_DIR/memory.txt" \
  "$RUN_DIR/lsblk.txt" \
  "$RUN_DIR/df.txt" \
  "$RUN_DIR/mount.txt" \
  "$RUN_DIR/ip_addr.txt" \
  "$RUN_DIR/ip_link.txt" \
  "$RUN_DIR/ip_route.txt" \
  "$RUN_DIR/resolv.conf" \
  "$RUN_DIR/sockets.txt" \
  "$RUN_DIR/ps_top.txt" \
  "$RUN_DIR/ps_mem.txt" \
  "$RUN_DIR/systemd_failed_units.txt" \
  "$RUN_DIR/systemd_timers.txt" \
  "$RUN_DIR/systemd_services.txt" \
  "$RUN_DIR/journal_last_1h.txt" \
  "$RUN_DIR/journal_last_boot_errors.txt" \
  "$RUN_DIR/apt_updates.txt" \
  "$RUN_DIR/ufw_status.txt" \
  "$RUN_DIR/apparmor_status.txt" \
  "$RUN_DIR/diag.prom"; do
  if [[ -f "$f" ]]; then
    append_report "$f"
  fi
done

log "Done. Logs in: $RUN_DIR"
