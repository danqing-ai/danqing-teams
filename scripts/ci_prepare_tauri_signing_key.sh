#!/usr/bin/env bash
# Normalize TAURI_SIGNING_PRIVATE_KEY for CI and write TAURI_SIGNING_PRIVATE_KEY_PATH.
# Usage (GitHub Actions):
#   RAW_KEY: ${{ secrets.TAURI_SIGNING_PRIVATE_KEY }}
#   bash scripts/ci_prepare_tauri_signing_key.sh
set -euo pipefail

OUT_DIR="${RUNNER_TEMP:-${TMPDIR:-/tmp}}"
KEY_PATH="${OUT_DIR}/tauri-updater.key"

if [[ -z "${RAW_KEY:-}" && -z "${TAURI_SIGNING_PRIVATE_KEY:-}" ]]; then
  echo "No TAURI_SIGNING_PRIVATE_KEY secret — updater artifacts will be skipped"
  if [[ -n "${GITHUB_ENV:-}" ]]; then
    echo "HAS_TAURI_SIGNING_KEY=false" >> "$GITHUB_ENV"
  fi
  exit 0
fi

# Prefer RAW_KEY (from step env) so we can avoid leaving a mangled KEY in later steps
KEY_CONTENT="${RAW_KEY:-$TAURI_SIGNING_PRIVATE_KEY}"

# Strip CR, surrounding quotes/whitespace (common when pasting into GitHub Secrets UI)
KEY_CONTENT="$(printf '%s' "$KEY_CONTENT" | tr -d '\r' | sed -e 's/^[[:space:]]*//' -e 's/[[:space:]]*$//' -e 's/^"//' -e 's/"$//' -e "s/^'//" -e "s/'$//")"

if [[ -z "$KEY_CONTENT" ]]; then
  echo "::error::TAURI_SIGNING_PRIVATE_KEY is empty after normalization"
  exit 1
fi

printf '%s' "$KEY_CONTENT" > "$KEY_PATH"
chmod 600 "$KEY_PATH"

# Validate: must be base64 that decodes to a minisign/rsign secret key header
DECODED="$(mktemp)"
trap 'rm -f "$DECODED"' EXIT
if ! base64 --decode < "$KEY_PATH" > "$DECODED" 2>/dev/null \
  && ! base64 -d < "$KEY_PATH" > "$DECODED" 2>/dev/null \
  && ! base64 -D < "$KEY_PATH" > "$DECODED" 2>/dev/null; then
  echo "::error::TAURI_SIGNING_PRIVATE_KEY is not valid base64 (Invalid padding usually means truncated/corrupt paste)."
  echo "::error::Re-set the repository secret from the local key file:"
  echo "::error::  gh secret set TAURI_SIGNING_PRIVATE_KEY < desktop/src-tauri/keys/updater.key"
  echo "::error::Do NOT paste the .pub file. Do NOT wrap in quotes. Use the full one-line .key contents."
  exit 1
fi

FIRST_LINE="$(head -n 1 "$DECODED" || true)"
case "$FIRST_LINE" in
  "untrusted comment: rsign encrypted secret key"|"untrusted comment: minisign encrypted secret key")
    ;;
  *)
    echo "::error::Decoded key does not look like a Tauri/minisign private key."
    echo "::error::First line was: ${FIRST_LINE:-<empty>}"
    echo "::error::Expected: 'untrusted comment: rsign encrypted secret key'"
    echo "::error::Re-set with: gh secret set TAURI_SIGNING_PRIVATE_KEY < desktop/src-tauri/keys/updater.key"
    exit 1
    ;;
esac

echo "Tauri signing key OK ($(wc -c < "$KEY_PATH" | tr -d ' ') bytes) → $KEY_PATH"

if [[ -n "${GITHUB_ENV:-}" ]]; then
  {
    echo "HAS_TAURI_SIGNING_KEY=true"
    echo "TAURI_SIGNING_PRIVATE_KEY_PATH=$KEY_PATH"
  } >> "$GITHUB_ENV"
fi
# Important: do NOT set TAURI_SIGNING_PRIVATE_KEY to an empty string.
# Tauri treats an empty KEY as present and fails with:
#   incorrect updater private key password: Missing comment in secret key
