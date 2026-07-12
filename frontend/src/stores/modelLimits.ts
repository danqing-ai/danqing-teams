import { defineStore } from 'pinia'
import { ref } from 'vue'
import { fetchJSON, asArray } from '@/api/client'
import { toast } from '@/utils/feedback'
import type { ModelLimit } from '@/types/mission'

export const useModelLimitsStore = defineStore('modelLimits', () => {
  const limits = ref<ModelLimit[]>([])
  const loading = ref(false)
  const saving = ref(false)

  async function load() {
    loading.value = true
    try {
      const data = await fetchJSON<ModelLimit[]>('/model-limits')
      limits.value = asArray(data)
    } catch {
      limits.value = []
    } finally {
      loading.value = false
    }
  }

  async function save(all: ModelLimit[]) {
    saving.value = true
    try {
      const data = await fetchJSON<ModelLimit[]>('/model-limits', {
        method: 'PUT',
        body: JSON.stringify(all),
      })
      limits.value = asArray(data)
      toast.success('模型参数已保存')
    } catch (e) {
      toast.error(e instanceof Error ? e.message : '保存失败')
      throw e
    } finally {
      saving.value = false
    }
  }

  return { limits, loading, saving, load, save }
})
