import { computed, ref } from 'vue'
import type { Update } from '@tauri-apps/plugin-updater'
import { fetchJSON } from '@/api/client'
import { isTauriRuntime } from '@/utils/desktop'

export type UpdaterStatus =
  | 'idle'
  | 'checking'
  | 'upToDate'
  | 'available'
  | 'downloading'
  | 'installing'
  | 'error'

const appVersion = ref('')
const status = ref<UpdaterStatus>('idle')
const availableVersion = ref('')
const updateNotes = ref('')
const errorMessage = ref('')
const downloadPercent = ref<number | null>(null)
const lastCheckedAt = ref<number | null>(null)

let pendingUpdate: Update | null = null
let initPromise: Promise<void> | null = null
let silentCheckStarted = false

async function resolveAppVersion(): Promise<string> {
  if (isTauriRuntime()) {
    try {
      const { getVersion } = await import('@tauri-apps/api/app')
      return await getVersion()
    } catch {
      /* fall through */
    }
  }
  try {
    const data = await fetchJSON<{ version: string }>('/version')
    return data.version || 'dev'
  } catch {
    return 'dev'
  }
}

async function ensureVersion(): Promise<string> {
  if (!appVersion.value) {
    appVersion.value = await resolveAppVersion()
  }
  return appVersion.value
}

async function closePending(): Promise<void> {
  if (!pendingUpdate) return
  try {
    await pendingUpdate.close()
  } catch {
    /* ignore */
  }
  pendingUpdate = null
}

/** Load displayed version once (Tauri getVersion, else /api/v1/version). */
export async function initAppVersion(): Promise<string> {
  if (!initPromise) {
    initPromise = ensureVersion().then(() => undefined)
  }
  await initPromise
  return appVersion.value
}

/**
 * Check for desktop updates. Silent mode does not set error status for network failures
 * (keeps sidebar quiet when GitHub/mirror is unreachable).
 */
export async function checkForUpdates(options?: { silent?: boolean }): Promise<boolean> {
  const silent = options?.silent ?? false
  await ensureVersion()

  if (!isTauriRuntime()) {
    status.value = 'idle'
    availableVersion.value = ''
    updateNotes.value = ''
    errorMessage.value = ''
    return false
  }

  status.value = 'checking'
  errorMessage.value = ''
  downloadPercent.value = null

  try {
    const { check } = await import('@tauri-apps/plugin-updater')
    await closePending()
    const update = await check()
    lastCheckedAt.value = Date.now()
    if (!update) {
      status.value = 'upToDate'
      availableVersion.value = ''
      updateNotes.value = ''
      return false
    }
    pendingUpdate = update
    availableVersion.value = update.version
    updateNotes.value = update.body ?? ''
    status.value = 'available'
    return true
  } catch (e) {
    const msg = e instanceof Error ? e.message : String(e)
    if (silent) {
      status.value = 'idle'
      errorMessage.value = ''
    } else {
      status.value = 'error'
      errorMessage.value = msg
    }
    return false
  }
}

/** Background check used on app start (desktop only). */
export function startSilentUpdateCheck(delayMs = 2500): void {
  if (!isTauriRuntime() || silentCheckStarted) return
  silentCheckStarted = true
  window.setTimeout(() => {
    void checkForUpdates({ silent: true })
  }, delayMs)
}

export async function downloadAndInstallUpdate(): Promise<void> {
  if (!isTauriRuntime()) {
    throw new Error('Updates are only available in the desktop app')
  }
  if (!pendingUpdate) {
    const found = await checkForUpdates()
    if (!found || !pendingUpdate) {
      throw new Error('No update available')
    }
  }

  const update = pendingUpdate
  status.value = 'downloading'
  errorMessage.value = ''
  downloadPercent.value = 0
  let contentLength = 0
  let downloaded = 0

  try {
    await update.downloadAndInstall((event) => {
      if (event.event === 'Started') {
        contentLength = event.data.contentLength ?? 0
        downloaded = 0
        downloadPercent.value = contentLength > 0 ? 0 : null
      } else if (event.event === 'Progress') {
        downloaded += event.data.chunkLength
        if (contentLength > 0) {
          downloadPercent.value = Math.min(99, Math.round((downloaded / contentLength) * 100))
        }
      } else if (event.event === 'Finished') {
        downloadPercent.value = 100
        status.value = 'installing'
      }
    })
    status.value = 'installing'
    const { relaunch } = await import('@tauri-apps/plugin-process')
    await relaunch()
  } catch (e) {
    const msg = e instanceof Error ? e.message : String(e)
    status.value = 'error'
    errorMessage.value = msg
    throw e
  }
}

export function useAppUpdater() {
  const hasUpdate = computed(() => status.value === 'available' || status.value === 'downloading' || status.value === 'installing')
  const isBusy = computed(() =>
    status.value === 'checking' || status.value === 'downloading' || status.value === 'installing',
  )
  const canCheck = computed(() => isTauriRuntime() && !isBusy.value)
  const canInstall = computed(() => isTauriRuntime() && status.value === 'available' && !!pendingUpdate)

  return {
    appVersion,
    status,
    availableVersion,
    updateNotes,
    errorMessage,
    downloadPercent,
    lastCheckedAt,
    hasUpdate,
    isBusy,
    canCheck,
    canInstall,
    isDesktop: isTauriRuntime,
    initAppVersion,
    checkForUpdates,
    downloadAndInstallUpdate,
    startSilentUpdateCheck,
  }
}
