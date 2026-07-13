import { defineStore } from 'pinia'
import { ref, watch } from 'vue'

export type ThemeId = 'mac' | 'linear-dark' | 'china-red-dark' | 'shadcn-dark' | 'shadcn-light'

export interface ThemeOption {
  id: ThemeId
  label: string
  description: string
  /** Class added to <html> to activate the theme */
  htmlClass: string | null
  /** Accent color for preview swatch */
  accent: string
  /** Whether this is a dark theme (controls the `dark` class on <html>) */
  dark: boolean
}

export const THEME_OPTIONS: ThemeOption[] = [
  {
    id: 'mac',
    label: 'macOS',
    description: 'macOS 原生风格',
    htmlClass: 'dq-mac',
    accent: '#0a84ff',
    dark: true,
  },
  {
    id: 'linear-dark',
    label: 'Linear Dark',
    description: 'Linear / Figma 风格深色生产力主题',
    htmlClass: 'dq-linear-dark',
    accent: '#6370d2',
    dark: true,
  },
  {
    id: 'china-red-dark',
    label: 'China Red Dark',
    description: '中国红深色主题',
    htmlClass: 'dq-china-red-dark',
    accent: '#C93756',
    dark: true,
  },
  {
    id: 'shadcn-dark',
    label: 'shadcn/ui Dark',
    description: 'shadcn/ui 风格 zinc 深色主题',
    htmlClass: 'dq-shadcn-dark',
    accent: '#fafafa',
    dark: true,
  },
  {
    id: 'shadcn-light',
    label: 'shadcn/ui Light',
    description: 'shadcn/ui 风格暖白亮色主题',
    htmlClass: 'dq-shadcn-light',
    accent: '#18181b',
    dark: false,
  },
]

const STORAGE_KEY = 'dq-theme'

/** All theme classes that can appear on <html> */
const ALL_THEME_CLASSES = ['dq-mac', 'dq-linear-dark', 'dq-china-red-dark', 'dq-shadcn-dark', 'dq-shadcn-light']

function applyTheme(id: ThemeId) {
  const option = THEME_OPTIONS.find((o) => o.id === id)
  if (!option) return

  const el = document.documentElement
  // Remove all theme classes
  ALL_THEME_CLASSES.forEach((cls) => el.classList.remove(cls))
  // Toggle dark mode
  el.classList.toggle('dark', option.dark)
  // Add the new theme class
  el.classList.add(option.htmlClass)
}

function getStoredTheme(): ThemeId {
  try {
    const stored = localStorage.getItem(STORAGE_KEY)
    if (stored && THEME_OPTIONS.some((o) => o.id === stored)) {
      return stored as ThemeId
    }
  } catch {
    // ignore
  }
  return 'shadcn-light'
}

export const useThemeStore = defineStore('theme', () => {
  const currentTheme = ref<ThemeId>(getStoredTheme())

  function setTheme(id: ThemeId) {
    currentTheme.value = id
    applyTheme(id)
    try {
      localStorage.setItem(STORAGE_KEY, id)
    } catch {
      // ignore
    }
  }

  function init() {
    applyTheme(currentTheme.value)
  }

  // Reactively watch and apply
  watch(currentTheme, (id) => {
    applyTheme(id)
    try {
      localStorage.setItem(STORAGE_KEY, id)
    } catch {
      // ignore
    }
  })

  return { currentTheme, setTheme, init }
})
