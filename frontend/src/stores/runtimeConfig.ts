import { defineStore } from 'pinia'
import { ref } from 'vue'
import { fetchJSON } from '@/api/client'
import { toast } from '@/utils/feedback'
import type { ConfigFile, UpdateConfigFileRequest } from '@/types/mission'

export interface RuntimeForm {
  autoApprove: boolean
  doomLoopThreshold: number
  maxStepsDefault: number
  maxDelegationDepth: number
  recallTopK: number
  searchTopK: number
  compactionEnabled: boolean
  compactionMaxTokens: number
  compactionTriggerRatio: number
  compactionCutTokens: number
  compactionTurnInterval: number
  compactionSubInterval: number
  compactionToolTruncate: number
}

export const useRuntimeConfigStore = defineStore('runtimeConfig', () => {
  const config = ref<RuntimeForm | null>(null)
  const loading = ref(false)
  const saving = ref(false)

  async function loadConfig() {
    loading.value = true
    try {
      const cfg = await fetchJSON<ConfigFile>('/config')
      const rt = cfg.runtime
      config.value = {
        autoApprove: rt.autoApprove,
        doomLoopThreshold: rt.turn.doomLoopThreshold,
        maxStepsDefault: rt.turn.maxStepsDefault,
        maxDelegationDepth: rt.team.maxDelegationDepth,
        recallTopK: rt.memory.recallTopK,
        searchTopK: rt.knowledge.searchTopK,
        compactionEnabled: rt.compaction?.enabled ?? false,
        compactionMaxTokens: rt.compaction?.maxTokens ?? 128000,
        compactionTriggerRatio: rt.compaction?.triggerRatio ?? 0.85,
        compactionCutTokens: rt.compaction?.cutTokens ?? 16000,
        compactionTurnInterval: rt.compaction?.turnInterval ?? 6,
        compactionSubInterval: rt.compaction?.subInterval ?? 4,
        compactionToolTruncate: rt.compaction?.toolTruncate ?? 2000,
      }
    } catch {
      config.value = null
    } finally {
      loading.value = false
    }
  }

  async function saveConfig(form: RuntimeForm) {
    saving.value = true
    try {
      const runtime: ConfigFile['runtime'] = {
        autoApprove: form.autoApprove,
        turn: {
          doomLoopThreshold: form.doomLoopThreshold,
          maxStepsDefault: form.maxStepsDefault,
        },
        team: {
          maxDelegationDepth: form.maxDelegationDepth,
        },
        memory: {
          recallTopK: form.recallTopK,
        },
        knowledge: {
          searchTopK: form.searchTopK,
        },
        compaction: {
          enabled: form.compactionEnabled,
          model: '',
          maxTokens: form.compactionMaxTokens,
          triggerRatio: form.compactionTriggerRatio,
          cutTokens: form.compactionCutTokens,
          turnInterval: form.compactionTurnInterval,
          subInterval: form.compactionSubInterval,
          toolTruncate: form.compactionToolTruncate,
        },
      }
      const req: UpdateConfigFileRequest = { runtime }
      const cfg = await fetchJSON<ConfigFile>('/config', {
        method: 'PUT',
        body: JSON.stringify(req),
      })
      const rt = cfg.runtime
      config.value = {
        autoApprove: rt.autoApprove,
        doomLoopThreshold: rt.turn.doomLoopThreshold,
        maxStepsDefault: rt.turn.maxStepsDefault,
        maxDelegationDepth: rt.team.maxDelegationDepth,
        recallTopK: rt.memory.recallTopK,
        searchTopK: rt.knowledge.searchTopK,
        compactionEnabled: rt.compaction?.enabled ?? false,
        compactionMaxTokens: rt.compaction?.maxTokens ?? 128000,
        compactionTriggerRatio: rt.compaction?.triggerRatio ?? 0.85,
        compactionCutTokens: rt.compaction?.cutTokens ?? 16000,
        compactionTurnInterval: rt.compaction?.turnInterval ?? 6,
        compactionSubInterval: rt.compaction?.subInterval ?? 4,
        compactionToolTruncate: rt.compaction?.toolTruncate ?? 2000,
      }
      toast.success('运行时配置已保存')
    } catch (e) {
      toast.error(e instanceof Error ? e.message : '保存失败')
      throw e
    } finally {
      saving.value = false
    }
  }

  return {
    config,
    loading,
    saving,
    loadConfig,
    saveConfig,
  }
})
