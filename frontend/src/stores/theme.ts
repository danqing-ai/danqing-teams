import { defineStore } from 'pinia'
import { ref, watch } from 'vue'

export type ThemeId = 'mac' | 'linear-dark' | 'china-red-dark' | 'shadcn-dark' | 'shadcn-light' | 'catppuccin' | 'tokyo-night' | 'minimal-light' | 'dracula' | 'nord-dark' | 'catppuccin-latte' | 'nord-light' | 'github-light'

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
    description: 'macOS 26 Liquid Glass 原生风格',
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
  {
    id: 'catppuccin',
    label: 'Catppuccin Mocha',
    description: '暖色柔和暗色主题，护眼舒适',
    htmlClass: 'dq-catppuccin',
    accent: '#cba6f7',
    dark: true,
  },
  {
    id: 'tokyo-night',
    label: 'Tokyo Night',
    description: '霓虹都市暗色主题，高对比度',
    htmlClass: 'dq-tokyo-night',
    accent: '#7aa2f7',
    dark: true,
  },
  {
    id: 'minimal-light',
    label: 'Minimal Light',
    description: '极简纯白亮色主题，专注编码',
    htmlClass: 'dq-minimal-light',
    accent: '#0066cc',
    dark: false,
  },
  {
    id: 'dracula',
    label: 'Dracula',
    description: '经典暗紫开发者主题',
    htmlClass: 'dq-dracula',
    accent: '#bd93f9',
    dark: true,
  },
  {
    id: 'nord-dark',
    label: 'Nord Dark',
    description: '北极蓝灰暗色主题，冷静沉稳',
    htmlClass: 'dq-nord-dark',
    accent: '#88c0d0',
    dark: true,
  },
  {
    id: 'catppuccin-latte',
    label: 'Catppuccin Latte',
    description: '暖色柔和亮色主题，护眼舒适',
    htmlClass: 'dq-catppuccin-latte',
    accent: '#1e66f5',
    dark: false,
  },
  {
    id: 'nord-light',
    label: 'Nord Light',
    description: '北极冰雪亮色主题，清新明快',
    htmlClass: 'dq-nord-light',
    accent: '#5e81ac',
    dark: false,
  },
  {
    id: 'github-light',
    label: 'GitHub Light',
    description: 'GitHub Primer 亮色主题，开发者首选',
    htmlClass: 'dq-github-light',
    accent: '#0969da',
    dark: false,
  },
]

const STORAGE_KEY = 'dq-theme'

/** All theme classes that can appear on <html> */
const ALL_THEME_CLASSES = ['dq-mac', 'dq-linear-dark', 'dq-china-red-dark', 'dq-shadcn-dark', 'dq-shadcn-light', 'dq-catppuccin', 'dq-tokyo-night', 'dq-minimal-light', 'dq-dracula', 'dq-nord-dark', 'dq-catppuccin-latte', 'dq-nord-light', 'dq-github-light']

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
