#!/bin/sh
# Helpers for derive peak memory (cgroup v2 + /proc VmHWM).
peak_kb_from_cgroup() {
  p="/sys/fs/cgroup/memory.peak"
  if [ ! -r "$p" ]; then
    return 1
  fi
  val="$(cat "$p" 2>/dev/null | tr -d ' \n')"
  case "$val" in
    ""|max) return 1 ;;
  esac
  awk -v b="$val" 'BEGIN{printf "%d", b/1024}'
  return 0
}

peak_kb_from_proc() {
  if [ -r /proc/self/status ]; then
    awk '/^VmHWM:/{print $2}' /proc/self/status 2>/dev/null | head -1
  fi
}

max_kb() {
  a="${1:-0}"
  b="${2:-0}"
  if [ "$b" -gt "$a" ]; then
    echo "$b"
  else
    echo "$a"
  fi
}
