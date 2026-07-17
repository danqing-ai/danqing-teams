import { defineStore } from 'pinia'
import { ref } from 'vue'
import { fetchJSON } from '@/api/client'
import { toast } from '@/utils/feedback'
import type { ConfigFile, UpdateConfigFileRequest, SandboxStatus } from '@/types/mission'

export interface RuntimeForm {
  autoApprove: boolean
  sandboxEnabled: boolean
  sandboxMode: 'read-only' | 'workspace-write' | 'danger-full-access'
  sandboxNetwork: 'deny' | 'allow' | 'allowlist'
  sandboxBackend: string
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

function formFromRuntime(rt: ConfigFile['runtime']): RuntimeForm {
  const sb = rt.sandbox
  return {
    autoApprove: rt.autoApprove,
    sandboxEnabled: sb?.enabled ?? true,
    sandboxMode: sb?.mode ?? 'workspace-write',
    sandboxNetwork: sb?.network ?? 'deny',
    sandboxBackend: sb?.backend ?? '',
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
}

export const useRuntimeConfigStore = defineStore('runtimeConfig', () => {
  const config = ref<RuntimeForm | null>(null)
  const sandboxStatus = ref<SandboxStatus | null>(null)
  const loading = ref(false)
  const saving = ref(false)

  async function loadSandboxStatus() {
    try {
      sandboxStatus.value = await fetchJSON<SandboxStatus>('/sandbox/status')
    } catch {
      sandboxStatus.value = null
    }
  }

  async function loadConfig() {
    loading.value = true
    try {
      const cfg = await fetchJSON<ConfigFile>('/config')
      config.value = formFromRuntime(cfg.runtime)
      await loadSandboxStatus()
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
        sandbox: {
          enabled: form.sandboxEnabled,
          mode: form.sandboxMode,
          network: form.sandboxNetwork,
          backend: form.sandboxBackend || undefined,
        },
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
      config.value = formFromRuntime(cfg.runtime)
      await loadSandboxStatus()
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
    sandboxStatus,
    loading,
    saving,
    loadConfig,
    loadSandboxStatus,
    saveConfig,
  }
})
