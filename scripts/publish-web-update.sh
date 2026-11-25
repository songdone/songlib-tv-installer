#!/usr/bin/env bash
set -euo pipefail

# Local-only helper to build, sign (Ed25519), and publish a SongLib TV Installer
# frontend hot-update snapshot to Cloudflare R2.
#
# Never stores secrets in the repository:
#   - Offline Ed25519 key:  SONGLIB_WEB_SIGNING_KEY_FILE (default: ~/secure-keys/songlib-web-signing.pem)
#   - R2 credentials file:  SONGLIB_R2_ENV_FILE          (default: ~/.config/songlib-r2.env, chmod 600)
#
# Modes:
#   1. Sign + publish a new UI release:
#      ./scripts/publish-web-update.sh \
#        --version 1.2.0-ui.2 --sequence 2 \
#        --min-backend 1.2.0 --backend-id <64-hex-backend-id> \
#        --prefix tv-installer/v1.2.0 [--notes-file notes.txt]
#
#   2. Upload an already-signed release directory:
#      ./scripts/publish-web-update.sh \
#        --release-dir build/web-release-v1.2.0-ui.1 \
#        --prefix tv-installer/v1.2.0

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

R2_ENV_FILE="${SONGLIB_R2_ENV_FILE:-${HOME}/.config/songlib-r2.env}"
if [[ -f "${R2_ENV_FILE}" ]]; then
  # Warn if credentials file is readable by group/others.
  perms="$(stat -f '%Lp' "${R2_ENV_FILE}" 2>/dev/null || stat -c '%a' "${R2_ENV_FILE}" 2>/dev/null || true)"
  if [[ -n "${perms}" && "${perms}" != "600" && "${perms}" != "400" ]]; then
    echo "ERROR: ${R2_ENV_FILE} has permissions ${perms}; please run: chmod 600 ${R2_ENV_FILE}" >&2
    exit 1
  fi
  set -a
  # shellcheck disable=SC1090
  source "${R2_ENV_FILE}"
  set +a
fi

BUCKET="${R2_BUCKET:-songlib-update}"
KEY_ID="${SONGLIB_WEB_UPDATE_KEY_ID:-songlib-web-2026}"
SIGNING_KEY="${SONGLIB_WEB_SIGNING_KEY_FILE:-${HOME}/secure-keys/songlib-web-signing.pem}"
PREFIX=""
RELEASE_DIR=""
VERSION=""
SEQUENCE=""
MIN_BACKEND=""
BACKEND_ID=""
NOTES_FILE=""
CONCURRENCY="8"
SKIP_BUILD="0"
SKIP_UPLOAD="0"
DRY_RUN="0"

usage() {
  cat <<'EOF'
Usage:
  Sign & upload new frontend snapshot:
    ./scripts/publish-web-update.sh \
      --version <ui-version> --sequence <N> \
      --min-backend <semver> --backend-id <sha256-hex> \
      --prefix <r2-prefix> [options]

  Upload an existing signed release directory:
    ./scripts/publish-web-update.sh \
      --release-dir <dir> --prefix <r2-prefix> [options]

Options:
  --release-dir <dir>     Existing signed release directory (skips build & signing)
  --prefix <prefix>       R2 object key prefix, e.g. tv-installer/v1.2.0 (required unless --skip-upload)
  --bucket <name>         R2 bucket name (default: $R2_BUCKET or songlib-update)
  --version <ver>         UI release version, e.g. 1.2.0-ui.2
  --sequence <N>          Monotonic positive integer sequence
  --min-backend <ver>     Minimum backend semver, e.g. 1.2.0
  --backend-id <hex>      Exact backend_id returned by /api/web-update
  --key-id <id>           Public key ID (default: songlib-web-2026)
  --signing-key <path>    Ed25519 PKCS8 PEM file (default: ~/secure-keys/songlib-web-signing.pem)
  --notes-file <path>     Optional release notes UTF-8 text file
  --concurrency <N>       Parallel uploads for files/* (default: 8)
  --skip-build            Reuse existing web/static/dist without running npm build
  --skip-upload           Only build and sign locally without uploading to R2
  --dry-run               Verify snapshot and print planned R2 uploads without network calls
  -h, --help              Show this help
EOF
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --release-dir) RELEASE_DIR="$2"; shift 2 ;;
    --prefix) PREFIX="$2"; shift 2 ;;
    --bucket) BUCKET="$2"; shift 2 ;;
    --version) VERSION="$2"; shift 2 ;;
    --sequence) SEQUENCE="$2"; shift 2 ;;
    --min-backend) MIN_BACKEND="$2"; shift 2 ;;
    --backend-id) BACKEND_ID="$2"; shift 2 ;;
    --key-id) KEY_ID="$2"; shift 2 ;;
    --signing-key) SIGNING_KEY="$2"; shift 2 ;;
    --notes-file) NOTES_FILE="$2"; shift 2 ;;
    --concurrency) CONCURRENCY="$2"; shift 2 ;;
    --skip-build) SKIP_BUILD="1"; shift ;;
    --skip-upload) SKIP_UPLOAD="1"; shift ;;
    --dry-run) DRY_RUN="1"; shift ;;
    -h|--help) usage; exit 0 ;;
    *) echo "Unknown option: $1" >&2; usage >&2; exit 2 ;;
  esac
done

if [[ -z "${RELEASE_DIR}" ]]; then
  if [[ -z "${VERSION}" || -z "${SEQUENCE}" || -z "${MIN_BACKEND}" || -z "${BACKEND_ID}" ]]; then
    echo "ERROR: Either --release-dir OR (--version, --sequence, --min-backend, --backend-id) is required." >&2
    exit 2
  fi
  if [[ ! -f "${SIGNING_KEY}" ]]; then
    echo "ERROR: Signing key not found at ${SIGNING_KEY}" >&2
    exit 1
  fi

  if [[ "${SKIP_BUILD}" != "1" ]]; then
    echo "[build] Building frontend in web/static..."
    (cd "${REPO_ROOT}/web/static" && npm run build)
  fi

  RELEASE_DIR="${REPO_ROOT}/build/web-release-v${VERSION}"
  echo "[sign] Signing snapshot into ${RELEASE_DIR}..."
  sign_cmd=(
    go run ./cmd/songlib-web-release
    -web "${REPO_ROOT}/web/static/dist"
    -out "${RELEASE_DIR}"
    -version "${VERSION}"
    -sequence "${SEQUENCE}"
    -min-backend "${MIN_BACKEND}"
    -backend-id "${BACKEND_ID}"
    -key-id "${KEY_ID}"
    -signing-key "${SIGNING_KEY}"
  )
  if [[ -n "${NOTES_FILE}" ]]; then
    sign_cmd+=(-notes-file "${NOTES_FILE}")
  fi
  (cd "${REPO_ROOT}" && "${sign_cmd[@]}")
fi

if [[ "${SKIP_UPLOAD}" == "1" ]]; then
  echo "[done] Local signed snapshot ready at ${RELEASE_DIR} (--skip-upload)."
  exit 0
fi

if [[ -z "${PREFIX}" ]]; then
  echo "ERROR: --prefix is required for R2 upload (e.g. --prefix tv-installer/v1.2.0)." >&2
  exit 2
fi

upload_cmd=(
  python3 "${SCRIPT_DIR}/r2_upload.py"
  --release-dir "${RELEASE_DIR}"
  --bucket "${BUCKET}"
  --prefix "${PREFIX}"
  --concurrency "${CONCURRENCY}"
)
if [[ "${DRY_RUN}" == "1" ]]; then
  upload_cmd+=(--dry-run)
fi

"${upload_cmd[@]}"

if [[ "${DRY_RUN}" != "1" ]]; then
  echo "[feed] https://update.6uos.de/${PREFIX#/}/manifest.json"
fi
