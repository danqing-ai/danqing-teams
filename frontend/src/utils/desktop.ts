/** True when UI runs inside Tauri (desktop shell). */
export function isTauriRuntime(): boolean {
  const w = window as Window & { __TAURI_INTERNALS__?: unknown; __TAURI__?: unknown };
  return Boolean(w.__TAURI_INTERNALS__ ?? w.__TAURI__);
}

/** Overlay title bar + transparent window styles (macOS Tauri only). */
export function installTauriMacosShell(): void {
  if (!isTauriRuntime()) return;
  const platform = navigator.platform.toLowerCase();
  const ua = navigator.userAgent.toLowerCase();
  if (!platform.includes('mac') && !ua.includes('mac')) return;
  document.documentElement.classList.add('dq-tauri-macos');
}
