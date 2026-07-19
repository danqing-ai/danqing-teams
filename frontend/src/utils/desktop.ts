/** True when UI runs inside Tauri (desktop shell). */
export function isTauriRuntime(): boolean {
  const w = window as Window & { __TAURI_INTERNALS__?: unknown; __TAURI__?: unknown };
  return Boolean(w.__TAURI_INTERNALS__ ?? w.__TAURI__);
}

/**
 * Resolve the API base URL for backend requests.
 * - VITE_API_BASE_URL (build-time) takes highest priority.
 * - In Tauri desktop runtime the webview may load from a custom protocol
 *   (e.g. tauri://localhost) where relative URLs cannot reach the Go backend,
 *   so we fall back to the absolute localhost address.
 * - Otherwise return empty string (same-origin, proxied by Vite dev server).
 */
export function apiBaseUrl(): string {
  return import.meta.env.VITE_API_BASE_URL ?? (isTauriRuntime() ? 'http://127.0.0.1:7801' : '');
}

/**
 * Wait until the local Go backend accepts HTTP (desktop first-launch race).
 * Sidecar spawn ≠ ready: migrate/SQLite can delay listen on first open.
 * Non-desktop runtimes return true immediately.
 */
export async function waitForBackend(opts?: {
  timeoutMs?: number
  intervalMs?: number
}): Promise<boolean> {
  if (!isTauriRuntime()) return true

  const timeoutMs = opts?.timeoutMs ?? 45_000
  const intervalMs = opts?.intervalMs ?? 250
  const url = `${apiBaseUrl()}/api/v1/version`
  const deadline = Date.now() + timeoutMs

  while (Date.now() < deadline) {
    try {
      const res = await fetch(url, { method: 'GET', cache: 'no-store' })
      if (res.ok) return true
    } catch {
      /* connection refused / not listening yet */
    }
    await new Promise((r) => setTimeout(r, intervalMs))
  }
  return false
}

/** Overlay title bar + transparent window styles (macOS Tauri only). */
export function installTauriMacosShell(): void {
  if (!isTauriRuntime()) return;
  const platform = navigator.platform.toLowerCase();
  const ua = navigator.userAgent.toLowerCase();
  if (!platform.includes('mac') && !ua.includes('mac')) return;
  document.documentElement.classList.add('dq-tauri-macos');
}

export type SaveBlobResult =
  | { ok: true; path?: string; method: 'dialog' | 'picker' | 'download' }
  | { ok: false; cancelled: true }

/**
 * Save a Blob with a location prompt when possible.
 * - Tauri desktop: native Save dialog, then write via sidecar command.
 * - Chromium: File System Access `showSaveFilePicker`.
 * - Otherwise: browser default download (usually ~/Downloads), no folder picker.
 */
export async function saveBlobAs(
  blob: Blob,
  fileName: string,
  opts?: { filters?: Array<{ name: string; extensions: string[] }> },
): Promise<SaveBlobResult> {
  if (isTauriRuntime()) {
    const { save } = await import('@tauri-apps/plugin-dialog')
    const { invoke } = await import('@tauri-apps/api/core')
    const path = await save({
      defaultPath: fileName,
      filters: opts?.filters ?? [{ name: 'Zip', extensions: ['zip'] }],
    })
    if (!path) return { ok: false, cancelled: true }
    const buf = new Uint8Array(await blob.arrayBuffer())
    await invoke('write_file_bytes', { path, contents: Array.from(buf) })
    return { ok: true, path, method: 'dialog' }
  }

  const w = window as Window & {
    showSaveFilePicker?: (options?: {
      suggestedName?: string
      types?: Array<{ description?: string; accept: Record<string, string[]> }>
    }) => Promise<FileSystemFileHandle>
  }
  if (typeof w.showSaveFilePicker === 'function') {
    try {
      const handle = await w.showSaveFilePicker({
        suggestedName: fileName,
        types: [
          {
            description: opts?.filters?.[0]?.name ?? 'Zip',
            accept: { 'application/zip': ['.zip'] },
          },
        ],
      })
      const writable = await handle.createWritable()
      await writable.write(blob)
      await writable.close()
      return { ok: true, path: handle.name, method: 'picker' }
    } catch (e) {
      // User cancelled, or API denied — don't fall through to silent download.
      if (e instanceof DOMException && e.name === 'AbortError') {
        return { ok: false, cancelled: true }
      }
      throw e
    }
  }

  const objectUrl = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = objectUrl
  a.download = fileName
  a.style.display = 'none'
  document.body.appendChild(a)
  a.click()
  a.remove()
  window.setTimeout(() => URL.revokeObjectURL(objectUrl), 1000)
  return { ok: true, method: 'download' }
}
