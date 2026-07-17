<script setup lang="ts">
import { computed, nextTick, ref, watch, onMounted } from 'vue'
import { useSessionsStore } from '@/stores/sessions'
import { useProjectsStore } from '@/stores/projects'
import { useLLMStore } from '@/stores/llm'
import { toast } from '@/utils/feedback'
import type { LLMModel } from '@/types/mission'
import {
  buildUserInputWithAttachments,
  chipLabel,
  chipTooltip,
  type ElementAttachment,
} from '@/types/element-attachment'

const content = ref('')
const attachments = ref<ElementAttachment[]>([])
const editingId = ref<string | null>(null)
const editingAnnotation = ref('')
const inputWrap = ref<HTMLElement | null>(null)
const sessions = useSessionsStore()
const projects = useProjectsStore()
const llm = useLLMStore()

const availableModels = computed(() => {
  return llm.models.map((m) => ({ id: m.id, label: m.id, model: m }))
})

const selectedModel = computed<LLMModel | undefined>(() => {
  const parts = sessions.selectedModelId.split('/')
  const baseId = parts.length >= 2 ? `${parts[0]}/${parts[1]}` : sessions.selectedModelId
  return llm.models.find((m) => m.id === baseId)
})

const selectedBaseModelId = computed({
  get: () => {
    const parts = sessions.selectedModelId.split('/')
    return parts.length >= 2 ? `${parts[0]}/${parts[1]}` : sessions.selectedModelId
  },
  set: (v: string) => {
    const newModel = llm.models.find((m) => m.id === v)
    const efforts = newModel?.availableEfforts ?? []
    let effort = sessions.selectedEffort
    if (effort && efforts.length > 0 && !efforts.includes(effort)) {
      effort = efforts[0]
      sessions.selectedEffort = effort
    }
    sessions.selectedModelId = effort && effort !== 'off' ? `${v}/${effort}` : v
  },
})

const availableEfforts = computed<string[]>(() => {
  return selectedModel.value?.availableEfforts ?? []
})

// When effort changes, update the full modelId
watch(() => sessions.selectedEffort, (effort) => {
  const parts = sessions.selectedModelId.split('/')
  if (parts.length >= 2) {
    const base = `${parts[0]}/${parts[1]}`
    sessions.selectedModelId = effort && effort !== 'off' ? `${base}/${effort}` : base
  }
})

const primaryAgents = computed(() => {
  const list = sessions.agents.filter((a) => a.mode !== 'subagent')
  const order = ['default', 'team', 'planner']
  return [...list].sort((a, b) => {
    const ai = order.indexOf(a.id)
    const bi = order.indexOf(b.id)
    const ao = ai === -1 ? order.length : ai
    const bo = bi === -1 ? order.length : bi
    if (ao !== bo) return ao - bo
    return a.name.localeCompare(b.name, 'zh-CN')
  })
})

const placeholder = computed(() =>
  sessions.composingNew ? '输入会话目标…' : '继续输入…',
)

const canSend = computed(
  () => Boolean(content.value.trim() || attachments.value.length) && !sessions.loading,
)

onMounted(async () => {
  const oldIds = new Set(llm.models.map((m) => m.id))
  await llm.loadModels()
  sessions.syncModelSelection(llm.models, oldIds)
})

watch(
  () => llm.models,
  (newModels, oldModels) => {
    const oldIds = new Set((oldModels ?? []).map((m) => m.id))
    sessions.syncModelSelection(newModels, oldIds)
  },
)

function focusInput() {
  void nextTick(() => {
    inputWrap.value?.querySelector('textarea')?.focus()
  })
}

function appendContent(text: string) {
  if (!text) return
  content.value = (content.value ? content.value + '\n' : '') + text
  focusInput()
}

function addElementAttachment(att: ElementAttachment) {
  attachments.value = [...attachments.value, att]
  focusInput()
}

function removeAttachment(id: string) {
  attachments.value = attachments.value.filter((a) => a.id !== id)
  if (editingId.value === id) {
    editingId.value = null
    editingAnnotation.value = ''
  }
}

function startEditAnnotation(att: ElementAttachment) {
  editingId.value = att.id
  editingAnnotation.value = att.annotation
}

function saveEditAnnotation() {
  const id = editingId.value
  if (!id) return
  attachments.value = attachments.value.map((a) =>
    a.id === id ? { ...a, annotation: editingAnnotation.value.trim() } : a,
  )
  editingId.value = null
  editingAnnotation.value = ''
}

function cancelEditAnnotation() {
  editingId.value = null
  editingAnnotation.value = ''
}

function clearComposer() {
  content.value = ''
  attachments.value = []
  editingId.value = null
  editingAnnotation.value = ''
}

watch(
  () => sessions.composingNew,
  (v) => {
    if (v) {
      clearComposer()
      if (!sessions.selectedProjectId && projects.sortedProjects.length) {
        sessions.selectedProjectId = projects.sortedProjects[0].id
      }
      focusInput()
    }
  },
)

const isTurnRunning = computed(
  () => !sessions.composingNew && sessions.runningTurnId !== null,
)

async function send() {
  const text = buildUserInputWithAttachments(content.value, attachments.value)
  if (!text || sessions.loading) return

  if (sessions.composingNew) {
    if (!sessions.selectedProjectId) {
      toast.warning('请选择项目')
      return
    }
    try {
      await sessions.createSession(text, sessions.selectedProjectId)
      clearComposer()
      focusInput()
    } catch (e) {
      toast.error(e instanceof Error ? e.message : '发送失败')
    }
    return
  }

  try {
    await sessions.sendTurn(text)
    clearComposer()
    focusInput()
  } catch (e) {
    toast.error(e instanceof Error ? e.message : '发送失败')
  }
}

async function stop() {
  if (!sessions.runningTurnId) return
  try {
    await sessions.cancelTurn(sessions.runningTurnId)
  } catch (e) {
    toast.error(e instanceof Error ? e.message : '取消失败')
  }
}

function onKeydown(e: KeyboardEvent) {
  if ((e.metaKey || e.ctrlKey) && e.key === 'Enter') {
    e.preventDefault()
    if (isTurnRunning.value) {
      void stop()
    } else {
      void send()
    }
    return
  }
  if (e.key === 'Enter' && !e.shiftKey) {
    e.preventDefault()
    if (isTurnRunning.value) {
      void stop()
    } else {
      void send()
    }
  }
}

defineExpose({ focusInput, appendContent, addElementAttachment })
</script>

<template>
  <div class="composer-float" role="form" aria-label="Session composer">
    <div v-if="attachments.length" class="composer-float__attachments">
      <div
        v-for="att in attachments"
        :key="att.id"
        class="composer-el-chip"
        :title="chipTooltip(att)"
      >
        <span class="composer-el-chip__icon" aria-hidden="true">
          <svg viewBox="0 0 24 24" width="12" height="12" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <circle cx="12" cy="12" r="10" />
            <line x1="22" y1="12" x2="18" y2="12" />
            <line x1="6" y1="12" x2="2" y2="12" />
            <line x1="12" y1="6" x2="12" y2="2" />
            <line x1="12" y1="22" x2="12" y2="18" />
          </svg>
        </span>
        <span class="composer-el-chip__label">{{ chipLabel(att) }}</span>
        <span v-if="att.annotation" class="composer-el-chip__dot" title="已有批注" />
        <button
          type="button"
          class="composer-el-chip__edit"
          title="编辑批注"
          aria-label="编辑批注"
          @click.stop="startEditAnnotation(att)"
        >
          <svg viewBox="0 0 24 24" width="11" height="11" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <path d="M12 20h9" />
            <path d="M16.5 3.5a2.1 2.1 0 0 1 3 3L7 19l-4 1 1-4Z" />
          </svg>
        </button>
        <button
          type="button"
          class="composer-el-chip__remove"
          title="移除"
          aria-label="移除元素附件"
          @click.stop="removeAttachment(att.id)"
        >
          ×
        </button>
      </div>
    </div>

    <div
      v-if="editingId"
      class="composer-float__edit-note"
    >
      <input
        v-model="editingAnnotation"
        class="composer-float__edit-input"
        placeholder="编辑批注…"
        @keydown.enter.prevent="saveEditAnnotation"
        @keydown.esc.prevent="cancelEditAnnotation"
      />
      <button type="button" class="composer-float__edit-btn" @click="saveEditAnnotation">保存</button>
      <button type="button" class="composer-float__edit-btn composer-float__edit-btn--ghost" @click="cancelEditAnnotation">取消</button>
    </div>

    <div ref="inputWrap" class="composer-float__body">
      <DqInput
        v-model="content"
        type="textarea"
        :rows="2"
        class="composer-float__input"
        :placeholder="placeholder"
        @keydown="onKeydown"
      />
    </div>

    <div class="composer-float__actions">
      <div class="composer-float__chips">
        <!-- Project selector -->
        <DqSelect
          v-if="sessions.composingNew"
          v-model="sessions.selectedProjectId"
          size="small"
          class="composer-chip-select"
          aria-label="选择项目"
          placeholder="选择项目"
        >
          <DqOption v-for="p in projects.sortedProjects" :key="p.id" :value="p.id" :label="p.name" />
        </DqSelect>

        <!-- No LLM warning -->
        <label v-if="llm.modelsLoaded && !availableModels.length" class="composer-chip">
          <span class="composer-chip__label composer-chip__label--accent">请先配置 LLM 提供商</span>
        </label>

        <!-- Agent selector: all primary agents (Default / Team / Planner / custom) -->
        <DqSelect
          v-if="(sessions.composingNew || sessions.currentSessionId) && primaryAgents.length"
          v-model="sessions.selectedAgentId"
          size="small"
          class="composer-chip-select"
          aria-label="选择 Agent"
        >
          <DqOption v-for="a in primaryAgents" :key="a.id" :value="a.id" :label="a.name" />
        </DqSelect>

        <!-- Model selector -->
        <DqSelect
          v-if="llm.modelsLoaded && availableModels.length"
          v-model="selectedBaseModelId"
          size="small"
          class="composer-chip-select"
          aria-label="选择模型"
        >
          <DqOption v-for="model in availableModels" :key="model.id" :value="model.id" :label="model.label" />
        </DqSelect>

        <!-- Effort selector -->
        <DqSelect
          v-if="llm.modelsLoaded && availableEfforts.length > 1"
          v-model="sessions.selectedEffort"
          size="small"
          class="composer-chip-select"
          aria-label="选择思考等级"
        >
          <DqOption v-for="e in availableEfforts" :key="e" :value="e" :label="e" />
        </DqSelect>

        <!-- Running indicator -->
        <span v-if="isTurnRunning" class="composer-chip composer-chip--running">
          <span class="composer-chip__running-dot" />
          运行中
        </span>

      </div>

      <div class="composer-float__actions-right">
        <DqIconButton
          v-if="isTurnRunning"
          class="composer-float__stop"
          aria-label="停止"
          @click="stop"
        >
          <svg viewBox="0 0 24 24" width="14" height="14" fill="currentColor" aria-hidden="true">
            <rect x="4" y="4" width="16" height="16" rx="2" />
          </svg>
        </DqIconButton>
        <DqIconButton
          v-else
          class="composer-float__send"
          :disabled="!canSend"
          aria-label="发送"
          @click="send"
        >
          <svg viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
            <path d="M12 19V5M5 12l7-7 7 7" />
          </svg>
        </DqIconButton>
      </div>
    </div>
  </div>
</template>

<style scoped>
.composer-float {
  background: var(--dq-glass-popover-bg);
  border: 1px solid var(--dq-glass-border-strong);
  border-radius: var(--dq-radius-menu);
  box-shadow: var(--dq-shadow-glass);
  -webkit-backdrop-filter: var(--dq-glass-blur-heavy);
  backdrop-filter: var(--dq-glass-blur-heavy);
  transition: border-color 0.18s ease, box-shadow 0.18s ease;
}

.composer-float:focus-within {
  border-color: var(--dq-accent);
  box-shadow:
    var(--dq-shadow-glass),
    0 0 0 1px color-mix(in srgb, var(--dq-accent) 16%, transparent);
}

.composer-float__attachments {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  padding: 10px 14px 0;
}

.composer-el-chip {
  position: relative;
  display: inline-flex;
  align-items: center;
  gap: 5px;
  max-width: 220px;
  height: 26px;
  padding: 0 4px 0 8px;
  border-radius: 8px;
  border: 1px solid color-mix(in srgb, var(--dq-accent) 28%, transparent);
  background: color-mix(in srgb, var(--dq-accent) 12%, transparent);
  color: var(--dq-label-primary);
  font-size: 12px;
  line-height: 1;
  cursor: default;
}

.composer-el-chip__icon {
  display: flex;
  color: var(--dq-accent);
  flex-shrink: 0;
}

.composer-el-chip__label {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-family: var(--dq-font-mono, ui-monospace, monospace);
  font-size: 11px;
}

.composer-el-chip__dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: var(--dq-accent);
  flex-shrink: 0;
}

.composer-el-chip__edit,
.composer-el-chip__remove {
  appearance: none;
  border: none;
  background: transparent;
  color: var(--dq-label-tertiary);
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  width: 18px;
  height: 18px;
  border-radius: 4px;
  padding: 0;
  font-size: 14px;
  line-height: 1;
}

.composer-el-chip__edit:hover,
.composer-el-chip__remove:hover {
  color: var(--dq-label-primary);
  background: color-mix(in srgb, var(--dq-label-primary) 10%, transparent);
}

.composer-float__edit-note {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px 14px 0;
}

.composer-float__edit-input {
  flex: 1;
  min-width: 0;
  height: 28px;
  padding: 0 8px;
  border-radius: 6px;
  border: 1px solid var(--dq-glass-border-strong);
  background: color-mix(in srgb, var(--dq-bg-base) 50%, transparent);
  color: var(--dq-label-primary);
  font-size: 12px;
  font-family: inherit;
}

.composer-float__edit-input:focus {
  outline: none;
  border-color: var(--dq-accent);
}

.composer-float__edit-btn {
  appearance: none;
  border: 1px solid transparent;
  border-radius: 6px;
  height: 28px;
  padding: 0 10px;
  font-size: 12px;
  cursor: pointer;
  background: var(--dq-accent);
  color: #fff;
  font-family: inherit;
}

.composer-float__edit-btn--ghost {
  background: transparent;
  border-color: var(--dq-glass-border-strong);
  color: var(--dq-label-secondary);
}

.composer-float__body :deep(textarea.dq-input) {
  display: block;
  width: 100%;
  min-height: 56px;
  max-height: 180px;
  resize: none;
  border: none;
  background: transparent;
  box-shadow: none;
  padding: 16px 18px 10px;
  font-size: var(--dq-font-size-secondary);
  line-height: 1.55;
  color: var(--dq-label-primary);
}

.composer-float__body :deep(textarea.dq-input::placeholder) {
  color: var(--dq-label-quaternary);
}

.composer-float__body :deep(textarea.dq-input:focus) {
  outline: none;
  box-shadow: none;
}

.composer-float__actions {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 0 10px 10px 12px;
}

.composer-float__chips {
  display: flex;
  align-items: center;
  gap: 8px;
  flex: 1 1 0;
  flex-wrap: nowrap;
  min-width: 0;
  overflow: hidden;
}

.composer-float__actions-right {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-shrink: 0;
}

.composer-chip {
  position: relative;
  display: inline-flex;
  align-items: center;
  gap: 5px;
  height: 28px;
  padding: 0 10px;
  border: 1px solid color-mix(in srgb, var(--dq-label-primary) 10%, transparent);
  border-radius: 999px;
  background: color-mix(in srgb, var(--dq-label-primary) 8%, transparent);
  color: var(--dq-label-secondary);
  font-size: var(--dq-font-size-footnote);
  font-weight: 500;
  line-height: 1;
  cursor: pointer;
  transition: background 0.12s ease, color 0.12s ease, border-color 0.12s ease;
}

.composer-chip:hover {
  color: var(--dq-label-primary);
  background: color-mix(in srgb, var(--dq-label-primary) 12%, transparent);
  border-color: color-mix(in srgb, var(--dq-label-primary) 16%, transparent);
}

.composer-chip__label--accent {
  color: var(--dq-accent);
}

.composer-chip__label {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  pointer-events: none;
}

.composer-chip--running {
  flex-shrink: 0;
  border-color: color-mix(in srgb, var(--dq-accent) 25%, transparent);
  background: color-mix(in srgb, var(--dq-accent) 12%, transparent);
  color: var(--dq-accent);
  cursor: default;
  padding-right: 10px;
}

.composer-chip__running-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: var(--dq-accent);
  flex-shrink: 0;
  animation: composer-pulse 1.2s ease-in-out infinite;
}

@keyframes composer-pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.3; }
}

.composer-float__send {
  flex-shrink: 0;
  width: 32px !important;
  height: 32px !important;
  border-radius: 50% !important;
  background: var(--dq-accent) !important;
  color: var(--dq-color-white) !important;
}

.composer-float__send:hover:not(:disabled) {
  filter: brightness(1.06);
}

.composer-float__stop {
  flex-shrink: 0;
  width: 32px !important;
  height: 32px !important;
  border-radius: 50% !important;
  background: var(--dq-danger) !important;
  color: var(--dq-color-white) !important;
}

.composer-float__stop:hover {
  filter: brightness(1.06);
}

.composer-chip-select {
  flex: 0 1 auto;
  min-width: 0;
  max-width: 160px;
}

.composer-chip-select :deep(.dq-select__trigger) {
  max-width: 100%;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  background: color-mix(in srgb, var(--dq-label-primary) 8%, transparent);
  border-color: color-mix(in srgb, var(--dq-label-primary) 10%, transparent);
}
</style>
