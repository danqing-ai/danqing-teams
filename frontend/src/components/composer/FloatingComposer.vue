<script setup lang="ts">
import { computed, nextTick, ref, watch, onMounted } from 'vue'
import { useSessionsStore } from '@/stores/sessions'
import { useProjectsStore } from '@/stores/projects'
import { useLLMStore } from '@/stores/llm'
import { toast } from '@/utils/feedback'

const content = ref('')
const inputWrap = ref<HTMLElement | null>(null)
const sessions = useSessionsStore()
const projects = useProjectsStore()
const llm = useLLMStore()

const availableModels = computed(() => {
  return llm.models.map((m) => ({ id: m.id, label: m.id }))
})

const selectedModelLabel = computed(
  () => availableModels.value.find((m) => m.id === sessions.selectedModelId)?.label ?? sessions.selectedModelId,
)

const primaryAgents = computed(() =>
  sessions.agents.filter((a) => a.mode !== 'subagent' && a.id !== 'default'),
)

const selectedAgentLabel = computed(() => {
  if (sessions.selectedAgentId) {
    return sessions.agents.find((a) => a.id === sessions.selectedAgentId)?.name ?? sessions.selectedAgentId
  }
  return 'Default'
})

const selectedProject = computed(() =>
  projects.projects.find((p) => p.id === sessions.selectedProjectId) ?? null,
)

const placeholder = computed(() =>
  sessions.composingNew ? '输入会话目标…' : '继续输入…',
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

watch(
  () => sessions.composingNew,
  (v) => {
    if (v) {
      content.value = ''
      if (!sessions.selectedProjectId && projects.projects.length) {
        sessions.selectedProjectId = projects.projects[0].id
      }
      focusInput()
    }
  },
)

async function send() {
  const text = content.value.trim()
  if (!text || sessions.loading) return

  if (sessions.composingNew) {
    if (!sessions.selectedProjectId) {
      toast.warning('请选择项目')
      return
    }
    try {
      await sessions.createSession(text, sessions.selectedProjectId)
      content.value = ''
      focusInput()
    } catch (e) {
      toast.error(e instanceof Error ? e.message : '发送失败')
    }
    return
  }

  try {
    await sessions.sendTurn(text)
    content.value = ''
    focusInput()
  } catch (e) {
    toast.error(e instanceof Error ? e.message : '发送失败')
  }
}

function onKeydown(e: KeyboardEvent) {
  if ((e.metaKey || e.ctrlKey) && e.key === 'Enter') {
    e.preventDefault()
    void send()
    return
  }
  if (e.key === 'Enter' && !e.shiftKey) {
    e.preventDefault()
    void send()
  }
}

defineExpose({ focusInput })
</script>

<template>
  <div class="composer-float" role="form" aria-label="Session composer">
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
        <label v-if="sessions.composingNew" class="composer-chip composer-chip--select">
          <span class="composer-chip__label">{{ selectedProject?.name || '选择项目' }}</span>
          <select v-model="sessions.selectedProjectId" class="composer-chip__select" aria-label="选择项目">
            <option v-for="p in projects.sortedProjects" :key="p.id" :value="p.id">
              {{ p.name }}
            </option>
          </select>
        </label>

        <label v-if="llm.modelsLoaded && !availableModels.length" class="composer-chip">
          <span class="composer-chip__label composer-chip__label--accent">请先配置 LLM 提供商</span>
        </label>
        <label v-else class="composer-chip composer-chip--select">
          <span class="composer-chip__label">{{ selectedModelLabel }}</span>
          <select v-model="sessions.selectedModelId" class="composer-chip__select" aria-label="选择模型">
            <option v-for="model in availableModels" :key="model.id" :value="model.id">
              {{ model.label }}
            </option>
          </select>
        </label>

        <label v-if="(sessions.composingNew || sessions.currentSessionId) && primaryAgents.length" class="composer-chip composer-chip--select">
          <span class="composer-chip__label">{{ selectedAgentLabel }}</span>
          <select v-model="sessions.selectedAgentId" class="composer-chip__select" aria-label="选择 Agent">
            <option :value="null">Default</option>
            <option v-for="a in primaryAgents" :key="a.id" :value="a.id">
              {{ a.name }}
            </option>
          </select>
        </label>

      </div>

      <div class="composer-float__actions-right">
        <button
          type="button"
          class="composer-float__send"
          :disabled="sessions.loading || !content.trim()"
          aria-label="发送"
          @click="send"
        >
          <svg viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
            <path d="M12 19V5M5 12l7-7 7 7" />
          </svg>
        </button>
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
  flex-wrap: wrap;
  min-width: 0;
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

.composer-chip.is-on {
  background: color-mix(in srgb, var(--dq-accent) 18%, transparent);
  border-color: color-mix(in srgb, var(--dq-accent) 28%, transparent);
  color: var(--dq-accent);
}

.composer-chip--select::after {
  content: '';
  position: absolute;
  right: 9px;
  top: 50%;
  width: 0;
  height: 0;
  margin-top: -1px;
  border-left: 3.5px solid transparent;
  border-right: 3.5px solid transparent;
  border-top: 4px solid currentColor;
  opacity: 0.7;
  pointer-events: none;
}

.composer-chip__label--accent {
  color: var(--dq-accent);
}

.composer-chip__select {
  position: absolute;
  inset: 0;
  opacity: 0;
  cursor: pointer;
}

.composer-chip__label {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  pointer-events: none;
}

.composer-chip--select {
  padding-right: 22px;
}

.composer-float__send {
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  width: 32px;
  height: 32px;
  padding: 0;
  border: none;
  border-radius: 50%;
  background: var(--dq-accent);
  color: var(--dq-color-white);
  cursor: pointer;
  transition: opacity 0.12s ease, transform 0.12s ease;
}

.composer-float__send:hover:not(:disabled) {
  filter: brightness(1.06);
}

.composer-float__send:active:not(:disabled) {
  transform: scale(0.96);
}

.composer-float__send:disabled {
  opacity: 0.35;
  cursor: not-allowed;
}
</style>
