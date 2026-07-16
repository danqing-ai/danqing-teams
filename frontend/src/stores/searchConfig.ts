import { defineStore } from 'pinia'
import { ref } from 'vue'
import { fetchJSON } from '@/api/client'
import { toast } from '@/utils/feedback'
import type { SearchConfig, UpsertSearchConfigRequest, SearchProvider, ConfigFile, UpdateConfigFileRequest } from '@/types/mission'

export const useSearchConfigStore = defineStore('searchConfig', () => {
  const config = ref<SearchConfig | null>(null)
  const loading = ref(false)
  const saving = ref(false)

  const providerOptions: { value: SearchProvider; label: string }[] = [
    { value: 'duckduckgo', label: 'DuckDuckGo (HTML)' },
    { value: 'bing', label: 'Bing (HTML)' },
    { value: 'brave', label: 'Brave Search' },
    { value: 'tavily', label: 'Tavily' },
    { value: 'bocha', label: 'Bocha' },
    { value: 'metaso', label: 'Metaso' },
    { value: 'searxng', label: 'SearXNG' },
    { value: 'baidu', label: 'Baidu AI Search' },
    { value: 'volcengine', label: 'Volcengine Ark' },
    { value: 'sofya', label: 'Sofya' },
  ]

  async function loadConfig() {
    loading.value = true
    try {
      const cfg = await fetchJSON<ConfigFile>('/config')
      config.value = cfg.search
    } catch {
      config.value = null
    } finally {
      loading.value = false
    }
  }

  async function saveConfig(payload: UpsertSearchConfigRequest) {
    saving.value = true
    try {
      const req: UpdateConfigFileRequest = { search: payload }
      const cfg = await fetchJSON<ConfigFile>('/config', {
        method: 'PUT',
        body: JSON.stringify(req),
      })
      config.value = cfg.search
      toast.success('搜索配置已保存')
    } catch (e) {
      toast.error(e instanceof Error ? e.message : '保存失败')
      throw e
    } finally {
      saving.value = false
    }
  }

  function providerLabel(p: SearchProvider) {
    return providerOptions.find((o) => o.value === p)?.label ?? p
  }

  return {
    config,
    loading,
    saving,
    providerOptions,
    loadConfig,
    saveConfig,
    providerLabel,
  }
})
