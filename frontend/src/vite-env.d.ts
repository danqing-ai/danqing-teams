/// <reference types="vite/client" />

declare const __ROUTER_BASE__: string
declare const __TAURI_BUILD__: boolean

interface ImportMetaEnv {
  readonly VITE_API_BASE_URL: string
}

interface ImportMeta {
  readonly env: ImportMetaEnv
}
