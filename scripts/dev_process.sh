# Dev process helpers — source after out_paths.sh (requires DQ_PROJECT, DQ_ROOT, DQ_RUN_DIR)

if [[ -n "${_DQ_DEV_PROCESS_LOADED:-}" ]]; then
  return 0 2>/dev/null || true
fi
_DQ_DEV_PROCESS_LOADED=1

dq_dev_marker() {
  printf '%s@%s' "$DQ_PROJECT" "$DQ_ROOT"
}

_dq_run_in_new_session() {
  local wrapper="$1"
  local log="$2"
  if command -v setsid >/dev/null 2>&1; then
    setsid "$wrapper" >>"$log" 2>&1 &
  else
    perl -MPOSIX -e 'setsid() or die $!; exec @ARGV' "$wrapper" >>"$log" 2>&1 &
  fi
  echo $!
}

_dq_kill_pgid() {
  local pid="$1"
  local signal="$2"
  [[ -n "$pid" ]] || return 1
  kill -0 "$pid" 2>/dev/null || return 1
  local pgid
  pgid="$(ps -o pgid= -p "$pid" 2>/dev/null | tr -d ' ' || true)"
  if [[ -n "$pgid" && "$pgid" -gt 0 ]]; then
    kill "-$signal" "-${pgid}" 2>/dev/null && return 0
  fi
  kill "-$signal" "$pid" 2>/dev/null
}

_dq_collect_descendants() {
  local root_pid="$1"
  local seen="$2"
  [[ -n "$root_pid" ]] || return 0
  [[ "$seen" == *"|${root_pid}|"* ]] && return 0
  seen="${seen}|${root_pid}|"
  printf '%s\n' "$root_pid"
  local child
  while read -r child; do
    [[ -z "$child" ]] && continue
    _dq_collect_descendants "$child" "$seen"
  done < <(pgrep -P "$root_pid" 2>/dev/null || true)
}

_dq_kill_tree() {
  local root_pid="$1"
  local signal="$2"
  local list=() i
  while read -r p; do
    [[ -n "$p" ]] && list+=("$p")
  done < <(_dq_collect_descendants "$root_pid" "|")
  (( ${#list[@]} == 0 )) && return 0
  for (( i=${#list[@]}-1; i>=0; i-- )); do
    kill "-$signal" "${list[$i]}" 2>/dev/null || true
  done
}

_dq_pids_by_env() {
  ps axeww 2>/dev/null \
    | grep -F "DQ_DEV=${DQ_PROJECT}" \
    | grep -F "DQ_DEV_ROOT=${DQ_ROOT}" \
    | awk '{print $1}' \
    | grep -E '^[0-9]+$' || true
}

_dq_kill_marked() {
  local signal="$1"
  local pid
  while read -r pid; do
    [[ -z "$pid" || "$pid" == "$$" ]] && continue
    kill "-$signal" "$pid" 2>/dev/null || true
  done
}

dq_dev_start() {
  local role="$1"
  local workdir="$2"
  shift 2

  local marker log pidfile wrapper_dir wrapper
  marker="$(dq_dev_marker)"
  log="$DQ_RUN_DIR/${role}.log"
  pidfile="$DQ_RUN_DIR/${role}.pid"
  wrapper_dir="$DQ_RUN_DIR/wrappers"
  wrapper="$wrapper_dir/${role}.sh"

  mkdir -p "$wrapper_dir"
  {
    printf '%s\n' '#!/usr/bin/env bash'
    printf '%s\n' "export DQ_DEV='${DQ_PROJECT}'"
    printf '%s\n' "export DQ_DEV_ROLE='${role}'"
    printf '%s\n' "export DQ_DEV_ROOT='${DQ_ROOT}'"
    printf '%s\n' "export DQ_DEV_MARKER='${marker}'"
    printf '%s\n' "cd '${workdir}'"
    printf '%s\n' 'set -a'
    [[ -n "${DQ_DEV_ENV:-}" ]] && printf '%s\n' "$DQ_DEV_ENV"
    printf '%s\n' 'set +a'
    printf 'exec '
    printf '%q ' "$@"
    printf '\n'
  } >"$wrapper"
  chmod +x "$wrapper"

  : >"$log"
  local pid
  pid="$(_dq_run_in_new_session "$wrapper" "$log")"
  echo "$pid" >"$pidfile"
  echo "$marker" >"$DQ_RUN_DIR/project.marker"
}

dq_dev_stop_all() {
  local stopped=0
  local root_pattern tracked=() role pf pid pids

  root_pattern="DQ_DEV_ROOT=${DQ_ROOT}"

  for role in backend frontend; do
    pf="$DQ_RUN_DIR/${role}.pid"
    [[ -f "$pf" ]] || continue
    pid="$(tr -d '[:space:]' <"$pf" || true)"
    rm -f "$pf"
    [[ -n "$pid" ]] || continue
    tracked+=("$pid")
    if _dq_kill_pgid "$pid" TERM; then
      echo "Stopped $role (pid $pid)"
      stopped=1
    fi
    _dq_kill_tree "$pid" TERM && stopped=1
  done

  sleep 1

  pids="$(pgrep -f "$root_pattern" 2>/dev/null || true)"
  if [[ -n "$pids" ]]; then
    while read -r pid; do
      [[ -z "$pid" || "$pid" == "$$" ]] && continue
      kill -TERM "$pid" 2>/dev/null && stopped=1 || true
    done <<<"$pids"
  fi

  pids="$(pgrep -f "${DQ_RUN_DIR}/wrappers/" 2>/dev/null || true)"
  if [[ -n "$pids" ]]; then
    while read -r pid; do
      [[ -z "$pid" || "$pid" == "$$" ]] && continue
      kill -TERM "$pid" 2>/dev/null && stopped=1 || true
    done <<<"$pids"
  fi

  pids="$(_dq_pids_by_env)"
  if [[ -n "$pids" ]]; then
    while read -r pid; do
      [[ -z "$pid" || "$pid" == "$$" ]] && continue
      kill -TERM "$pid" 2>/dev/null && stopped=1 || true
    done <<<"$pids"
  fi

  sleep 1

  local survivors=() seen="|"
  while read -r pid; do
    [[ -z "$pid" || "$pid" == "$$" ]] && continue
    survivors+=("$pid")
  done < <(pgrep -f "$root_pattern" 2>/dev/null || true)
  while read -r pid; do
    [[ -z "$pid" || "$pid" == "$$" ]] && continue
    survivors+=("$pid")
  done < <(pgrep -f "${DQ_RUN_DIR}/wrappers/" 2>/dev/null || true)
  while read -r pid; do
    [[ -z "$pid" || "$pid" == "$$" ]] && continue
    survivors+=("$pid")
  done < <(_dq_pids_by_env)

  if ((${#tracked[@]} > 0)); then
  for pid in "${tracked[@]}"; do
    survivors+=("$pid")
  done
  fi

  if ((${#survivors[@]} > 0)); then
    for pid in "${survivors[@]}"; do
      while read -r expanded; do
        [[ -z "$expanded" ]] && continue
        [[ "$seen" == *"|${expanded}|"* ]] && continue
        seen="${seen}${expanded}|"
        kill -0 "$expanded" 2>/dev/null || continue
        kill -KILL "$expanded" 2>/dev/null && stopped=1 || true
      done < <(_dq_collect_descendants "$pid" "|")
    done
  fi

  rm -f "$DQ_RUN_DIR/project.marker"

  if [[ "$stopped" -eq 0 ]]; then
    echo "No dev processes found for ${DQ_PROJECT} (${DQ_ROOT})"
  fi
}
