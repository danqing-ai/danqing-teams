import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { fetchJSON, asArray } from '@/api/client'
import { toast } from '@/utils/feedback'
import type { LLMProviderConfig, LLMModel, LLMModelRef, LLMProviderPreset, UpsertLLMProviderConfigRequest } from '@/types/mission'

export const useLLMStore = defineStore('llm', () => {
  const configs = ref<LLMProviderConfig[]>([])
  const models = ref<LLMModel[]>([])
  const presets = ref<LLMProviderPreset[]>([])
  const modelsLoaded = ref(false)
  const loading = ref(false)
  const saving = ref(false)

  const availableModels = computed(() => models.value)
  const modelsByProvider = computed(() => {
    const map = new Map<string, LLMModel[]>()
    for (const m of models.value) {
      const key = m.providerId
      if (!map.has(key)) map.set(key, [])
      map.get(key)!.push(m)
    }
    return map
  })

  async function loadConfigs() {
    loading.value = true
    try {
      configs.value = asArray(await fetchJSON<LLMProviderConfig[]>('/llm/configs'))
    } catch {
      configs.value = []
    } finally {
      loading.value = false
    }
  }

  async function loadPresets() {
    try {
      presets.value = await fetchJSON<LLMProviderPreset[]>('/llm/presets')
    } catch {
      presets.value = []
    }
  }

  async function loadModels() {
    try {
      models.value = asArray(await fetchJSON<LLMModel[]>('/llm/models'))
    } catch {
      models.value = []
    } finally {
      modelsLoaded.value = true
    }
  }

  async function saveConfig(payload: UpsertLLMProviderConfigRequest) {
    saving.value = true
    try {
      await fetchJSON('/llm/configs', {
        method: 'POST',
        body: JSON.stringify(payload),
      })
      await loadConfigs()
      await loadModels()
      toast.success('提供商已保存')
    } catch (e) {
      toast.error(e instanceof Error ? e.message : '保存失败')
      throw e
    } finally {
      saving.value = false
    }
  }

  async function updateConfig(id: string, payload: UpsertLLMProviderConfigRequest) {
    saving.value = true
    try {
      await fetchJSON(`/llm/configs/${encodeURIComponent(id)}`, {
        method: 'PUT',
        body: JSON.stringify(payload),
      })
      await loadConfigs()
      await loadModels()
      toast.success('提供商已更新')
    } catch (e) {
      toast.error(e instanceof Error ? e.message : '更新失败')
      throw e
    } finally {
      saving.value = false
    }
  }

  async function deleteConfig(id: string) {
    saving.value = true
    try {
      await fetchJSON(`/llm/configs/${encodeURIComponent(id)}`, { method: 'DELETE' })
      await loadConfigs()
      await loadModels()
      toast.success('提供商已删除')
    } catch (e) {
      toast.error(e instanceof Error ? e.message : '删除失败')
      throw e
    } finally {
      saving.value = false
    }
  }

  async function refreshModels(configId: string): Promise<LLMModelRef[]> {
    saving.value = true
    try {
      const res = await fetchJSON<{ models: LLMModelRef[] }>(
        `/llm/configs/${encodeURIComponent(configId)}/refresh-models`,
        { method: 'POST' },
      )
      toast.success('模型列表已刷新')
      await loadConfigs()
      await loadModels()
      return res?.models ?? []
    } catch (e) {
      toast.error(e instanceof Error ? e.message : '刷新失败')
      throw e
    } finally {
      saving.value = false
    }
  }

  async function fetchModels(payload: UpsertLLMProviderConfigRequest): Promise<LLMModelRef[]> {
    saving.value = true
    try {
      const res = await fetchJSON<{ models: LLMModelRef[] }>('/llm/configs/fetch-models', {
        method: 'POST',
        body: JSON.stringify(payload),
      })
      await loadModels()
      return res?.models ?? []
    } catch (e) {
      toast.error(e instanceof Error ? e.message : '获取模型失败')
      throw e
    } finally {
      saving.value = false
    }
  }

  async function toggleModel(configId: string, modelName: string, enabled: boolean) {
    try {
      await fetchJSON(`/llm/configs/${encodeURIComponent(configId)}/models/${encodeURIComponent(modelName)}`, {
        method: 'PATCH',
        body: JSON.stringify({ enabled }),
      })
      await loadConfigs()
      await loadModels()
    } catch (e) {
      toast.error(e instanceof Error ? e.message : '更新模型状态失败')
      throw e
    }
  }

  function ensureLoaded() {
    if (!configs.value.length && !loading.value) {
      void loadConfigs()
    }
    if (!models.value.length) {
      void loadModels()
    }
  }

  function getConfig(id: string): LLMProviderConfig | undefined {
    return configs.value.find((c) => c.id === id)
  }

  return {
    configs,
    models,
    presets,
    modelsLoaded,
    loading,
    saving,
    availableModels,
    modelsByProvider,
    loadConfigs,
    loadModels,
    loadPresets,
    saveConfig,
    updateConfig,
    deleteConfig,
    refreshModels,
    fetchModels,
    toggleModel,
    ensureLoaded,
    getConfig,
  }
})
