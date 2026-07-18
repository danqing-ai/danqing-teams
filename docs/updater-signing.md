# Desktop updater signing

Generate once:

```bash
cd desktop
npx tauri signer generate -w src-tauri/keys/updater.key -p ""
```

- Put the **public** key into `desktop/src-tauri/tauri.conf.json` → `plugins.updater.pubkey`
- Store the **private** key in GitHub Actions secrets:
  - `TAURI_SIGNING_PRIVATE_KEY` — full contents of `updater.key`
  - `TAURI_SIGNING_PRIVATE_KEY_PASSWORD` — password (empty if none)
- Optional China mirror secrets:
  - `UPDATE_MIRROR_BASE_URL` — public HTTPS base (e.g. `https://releases.danqing.ai/danqing-teams`); must match the first `plugins.updater.endpoints` entry host/path
  - `UPDATE_MIRROR_S3_URI` — S3/OSS URI for `aws s3 sync` (e.g. `s3://bucket/danqing-teams`)
  - or `UPDATE_MIRROR_RCLONE_REMOTE` — rclone destination
  - `UPDATE_MIRROR_AWS_ACCESS_KEY_ID` / `UPDATE_MIRROR_AWS_SECRET_ACCESS_KEY` / `UPDATE_MIRROR_AWS_REGION` / `UPDATE_MIRROR_AWS_ENDPOINT_URL` as needed for S3-compatible storage

App endpoints (in `tauri.conf.json`) try the China mirror first, then GitHub Releases `latest.json`.

Local pack with updater artifacts:

```bash
export TAURI_SIGNING_PRIVATE_KEY="$(cat desktop/src-tauri/keys/updater.key)"
export TAURI_SIGNING_PRIVATE_KEY_PASSWORD=""
make pack-macos-desktop
```

Private keys under `desktop/src-tauri/keys/` are gitignored.
