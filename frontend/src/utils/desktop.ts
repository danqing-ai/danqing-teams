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

/** Overlay title bar + transparent window styles (macOS Tauri only). */
export function installTauriMacosShell(): void {
  if (!isTauriRuntime()) return;
  const platform = navigator.platform.toLowerCase();
  const ua = navigator.userAgent.toLowerCase();
  if (!platform.includes('mac') && !ua.includes('mac')) return;
  document.documentElement.classList.add('dq-tauri-macos');
}
