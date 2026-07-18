<script setup lang="ts">
import { computed, nextTick, ref, watch, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useSessionsStore } from '@/stores/sessions'
import { useProjectsStore } from '@/stores/projects'
import { useLLMStore } from '@/stores/llm'
import { useWorkspaceUiStore } from '@/stores/workspaceUi'
import ComposerAttachmentTray from '@/components/composer/ComposerAttachmentTray.vue'
import { toast } from '@/utils/feedback'
import type { LLMModel } from '@/types/mission'
import type { ElementAttachment } from '@/types/element-attachment'
import {
  buildComposerUserInput,
  createComposerAttachmentId,
  MAX_IMAGE_ATTACHMENT_BYTES,
  toApiImageAttachments,
  type ComposerAttachment,
  type ElementComposerAttachment,
} from '@/types/composer-attachment'

const { t } = useI18n()
const content = ref('')
const attachments = ref<ComposerAttachment[]>([])
const editingId = ref<string | null>(null)
const editingAnnotation = ref('')
const inputWrap = ref<HTMLElement | null>(null)
const fileInputRef = ref<HTMLInputElement | null>(null)
const dragOver = ref(false)
const sessions = useSessionsStore()
const projects = useProjectsStore()
const llm = useLLMStore()
const workspaceUi = useWorkspaceUiStore()

const isMac =
  typeof navigator !== 'undefined' &&
  /(Mac|iPhone|iPad|iPod)/i.test(navigator.platform || navigator.userAgent || '')

const sendShortcut = computed(() => (isMac ? '⌘↵' : 'Ctrl+↵'))

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
  sessions.composingNew ? t('composer.placeholderNew') : t('composer.placeholderContinue'),
)

const hasPendingApproval = computed(() => workspaceUi.pendingApprovals > 0)

const canSend = computed(
  () =>
    Boolean(content.value.trim() || attachments.value.length) &&
    !sessions.loading &&
    !hasPendingApproval.value,
)

const isTurnRunning = computed(
  () => !sessions.composingNew && sessions.runningTurnId !== null,
)

const showAgentSelect = computed(
  () => (sessions.composingNew || sessions.currentSessionId) && primaryAgents.value.length > 0,
)

/** Few primary agents → segmented toggle; many → dropdown. */
const useAgentSegmented = computed(() => primaryAgents.value.length > 0 && primaryAgents.value.length <= 4)

const agentOptions = computed(() =>
  primaryAgents.value.map((a) => ({ label: a.name, value: a.id })),
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
  const wrapped: ElementComposerAttachment = {
    id: att.id,
    kind: 'element',
    data: att,
  }
  attachments.value = [...attachments.value, wrapped]
  focusInput()
}

function removeAttachment(id: string) {
  attachments.value = attachments.value.filter((a) => a.id !== id)
  if (editingId.value === id) {
    editingId.value = null
    editingAnnotation.value = ''
  }
}

function startEditAnnotation(att: ElementComposerAttachment) {
  editingId.value = att.id
  editingAnnotation.value = att.data.annotation
}

function saveEditAnnotation() {
  const id = editingId.value
  if (!id) return
  attachments.value = attachments.value.map((a) => {
    if (a.id !== id || a.kind !== 'element') return a
    return { ...a, data: { ...a.data, annotation: editingAnnotation.value.trim() } }
  })
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

function openFilePicker() {
  fileInputRef.value?.click()
}

function onFilePicked(e: Event) {
  const input = e.target as HTMLInputElement
  const files = Array.from(input.files ?? [])
  input.value = ''
  for (const f of files) addLocalFile(f)
}

function addLocalFile(file: File) {
  if (file.type.startsWith('image/')) {
    addImageFile(file)
    return
  }
  // Non-image: placeholder chip until upload pipeline exists
  attachments.value = [
    ...attachments.value,
    {
      id: createComposerAttachmentId('file'),
      kind: 'file',
      name: file.name,
      mime: file.type || 'application/octet-stream',
      size: file.size,
      placeholder: true,
    },
  ]
  toast.info(t('composer.attachFilePlaceholder'))
  focusInput()
}

async function send() {
  if (hasPendingApproval.value) {
    toast.warning(t('sessions.pendingApprovalHint'))
    return
  }
  const text = buildComposerUserInput(content.value, attachments.value)
  const imageAtts = toApiImageAttachments(attachments.value)
  if ((!text.trim() && !imageAtts.length) || sessions.loading) return

  if (sessions.composingNew) {
    if (!sessions.selectedProjectId) {
      toast.warning(t('composer.needProject'))
      return
    }
    try {
      await sessions.createSession(text, sessions.selectedProjectId, imageAtts)
      clearComposer()
      focusInput()
    } catch (e) {
      toast.error(e instanceof Error ? e.message : t('composer.sendFailed'))
    }
    return
  }

  try {
    await sessions.sendTurn(text, imageAtts)
    clearComposer()
    focusInput()
  } catch (e) {
    toast.error(e instanceof Error ? e.message : t('composer.sendFailed'))
  }
}

async function stop() {
  if (!sessions.runningTurnId) return
  try {
    await sessions.cancelTurn(sessions.runningTurnId)
  } catch (e) {
    toast.error(e instanceof Error ? e.message : t('composer.cancelFailed'))
  }
}

function onKeydown(e: KeyboardEvent) {
  if ((e.metaKey || e.ctrlKey) && e.key === 'Enter') {
    e.preventDefault()
    if (isTurnRunning.value) void stop()
    else void send()
    return
  }
  if (e.key === 'Enter' && !e.shiftKey) {
    e.preventDefault()
    if (isTurnRunning.value) void stop()
    else void send()
  }
}

function addImageFile(file: File) {
  if (!file.type.startsWith('image/')) return
  if (file.size > MAX_IMAGE_ATTACHMENT_BYTES) {
    toast.warning(t('composer.attachImageTooLarge', { max: '10 MB' }))
    return
  }
  const reader = new FileReader()
  reader.onload = () => {
    const dataUrl = String(reader.result ?? '')
    attachments.value = [
      ...attachments.value,
      {
        id: createComposerAttachmentId('img'),
        kind: 'image',
        name: file.name || `paste-${Date.now()}.png`,
        mime: file.type || 'image/png',
        size: file.size,
        dataUrl,
      },
    ]
    focusInput()
  }
  reader.readAsDataURL(file)
}

function onPaste(e: ClipboardEvent) {
  const items = e.clipboardData?.items
  if (!items) return
  const images = Array.from(items).filter((i) => i.type.startsWith('image/'))
  if (!images.length) return
  e.preventDefault()
  for (const item of images) {
    const file = item.getAsFile()
    if (file) addImageFile(file)
  }
  toast.success(t('composer.pasteImageAdded'))
}

function onDrop(e: DragEvent) {
  e.preventDefault()
  dragOver.value = false
  const files = Array.from(e.dataTransfer?.files ?? [])
  if (!files.length) return
  for (const f of files) addLocalFile(f)
}

defineExpose({ focusInput, appendContent, addElementAttachment })
</script>

<template>
  <div
    class="composer-float"
    :class="{
      'is-dragover': dragOver,
      'is-blocked': hasPendingApproval,
      'is-compose': sessions.composingNew,
    }"
    role="form"
    aria-label="Session composer"
    @dragover.prevent="dragOver = true"
    @dragleave.prevent="dragOver = false"
    @drop="onDrop"
  >
    <!-- Upper card: input + model/effort/send -->
    <div class="composer-float__card">
      <div v-if="dragOver" class="composer-float__drop">{{ t('composer.dropHint') }}</div>

      <div v-if="hasPendingApproval" class="composer-float__banner composer-float__banner--warn">
        {{ t('sessions.pendingApprovalHint') }}
      </div>
      <div v-else-if="isTurnRunning" class="composer-float__banner composer-float__banner--run">
        <span class="composer-float__run-dot" />
        {{ t('composer.running') }}
      </div>

      <ComposerAttachmentTray
        :attachments="attachments"
        :editing-id="editingId"
        :editing-annotation="editingAnnotation"
        @remove="removeAttachment"
        @edit-start="startEditAnnotation"
        @edit-save="saveEditAnnotation"
        @edit-cancel="cancelEditAnnotation"
        @update:editing-annotation="editingAnnotation = $event"
      />

      <div ref="inputWrap" class="composer-float__body">
        <DqInput
          v-model="content"
          type="textarea"
          :rows="sessions.composingNew ? 3 : 2"
          class="composer-float__input"
          :placeholder="placeholder"
          @keydown="onKeydown"
          @paste="onPaste"
        />
      </div>

      <input
        ref="fileInputRef"
        type="file"
        class="composer-float__file-input"
        multiple
        accept="image/*,.pdf,.txt,.md,.json,.csv,.png,.jpg,.jpeg,.webp,.gif"
        @change="onFilePicked"
      />

      <div class="composer-float__footer">
        <div class="composer-float__footer-leading">
          <button
            type="button"
            class="composer-tool-btn"
            :title="t('composer.attachFile')"
            :aria-label="t('composer.attachFile')"
            @click="openFilePicker"
          >
            <svg viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
              <path d="M12 5v14" />
              <path d="M5 12h14" />
            </svg>
          </button>
          <span
            v-if="llm.modelsLoaded && !availableModels.length"
            class="composer-meta-chip composer-meta-chip--warn"
          >
            {{ t('composer.needLlm') }}
          </span>
        </div>

        <div class="composer-float__footer-trailing">
          <div
            v-if="llm.modelsLoaded && availableModels.length"
            class="composer-select composer-select--model"
          >
            <DqSelect
              v-model="selectedBaseModelId"
              size="small"
              :aria-label="t('composer.selectModel')"
            >
              <DqOption
                v-for="model in availableModels"
                :key="model.id"
                :value="model.id"
                :label="model.label"
              />
            </DqSelect>
          </div>

          <div
            v-if="llm.modelsLoaded && availableEfforts.length > 1"
            class="composer-select composer-select--effort"
          >
            <DqSelect
              v-model="sessions.selectedEffort"
              size="small"
              :aria-label="t('composer.selectEffort')"
            >
              <DqOption v-for="e in availableEfforts" :key="e" :value="e" :label="e" />
            </DqSelect>
          </div>

          <button
            v-if="isTurnRunning"
            type="button"
            class="composer-send composer-send--stop"
            :aria-label="t('composer.stop')"
            @click="stop"
          >
            <svg viewBox="0 0 24 24" width="14" height="14" fill="currentColor" aria-hidden="true">
              <rect x="5" y="5" width="14" height="14" rx="2" />
            </svg>
            <span>{{ t('composer.stop') }}</span>
          </button>
          <button
            v-else
            type="button"
            class="composer-send"
            :disabled="!canSend"
            :aria-label="t('composer.send')"
            @click="send"
          >
            <span>{{ t('composer.send') }}</span>
            <kbd class="composer-send__kbd">{{ sendShortcut }}</kbd>
          </button>
        </div>
      </div>
    </div>

    <!-- Lower tray: project + agent context -->
    <div
      v-if="sessions.composingNew || showAgentSelect"
      class="composer-float__tray"
    >
      <div class="composer-float__tray-leading">
        <div
          v-if="sessions.composingNew"
          class="composer-select composer-select--project"
        >
          <DqSelect
            v-model="sessions.selectedProjectId"
            size="small"
            :aria-label="t('composer.selectProject')"
            :placeholder="t('composer.selectProject')"
          >
            <DqOption
              v-for="p in projects.sortedProjects"
              :key="p.id"
              :value="p.id"
              :label="p.name"
            />
          </DqSelect>
        </div>

        <DqSegmented
          v-if="showAgentSelect && useAgentSegmented"
          v-model="sessions.selectedAgentId"
          class="composer-agent-seg composer-agent-seg--compact"
          :options="agentOptions"
          :aria-label="t('composer.selectAgent')"
        />
        <div
          v-else-if="showAgentSelect"
          class="composer-select composer-select--agent"
        >
          <DqSelect
            v-model="sessions.selectedAgentId"
            size="small"
            :aria-label="t('composer.selectAgent')"
          >
            <DqOption
              v-for="a in primaryAgents"
              :key="a.id"
              :value="a.id"
              :label="a.name"
            />
          </DqSelect>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.composer-float {
  position: relative;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.composer-float.is-blocked {
  opacity: 0.94;
}

.composer-float.is-dragover .composer-float__card {
  border-color: var(--dq-accent);
  background: color-mix(in srgb, var(--dq-accent) 6%, var(--dq-glass-popover-bg));
}

.composer-float__card {
  position: relative;
}

.composer-float__drop {
  position: absolute;
  inset: 0;
  z-index: 2;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: inherit;
  background: color-mix(in srgb, var(--dq-accent) 12%, var(--dq-glass-popover-bg));
  color: var(--dq-accent);
  font-weight: 600;
  pointer-events: none;
}

.composer-float__banner {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 14px;
  font-size: var(--dq-font-size-caption);
  font-weight: 600;
  border-bottom: 1px solid var(--dq-separator-light);
}

.composer-float__banner--warn {
  color: var(--dq-system-orange);
  background: color-mix(in srgb, var(--dq-system-orange) 8%, transparent);
}

.composer-float__banner--run {
  color: var(--dq-accent);
  background: color-mix(in srgb, var(--dq-accent) 8%, transparent);
}

.composer-float__run-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: currentColor;
  animation: pulse 1.2s ease-in-out infinite;
}

.composer-float__body {
  padding: 12px 14px 4px;
}

.composer-float__body :deep(.dq-input),
.composer-float__body :deep(textarea) {
  border: none !important;
  background: transparent !important;
  box-shadow: none !important;
  resize: none;
}

.composer-float__file-input {
  display: none;
}

.composer-float__footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  padding: 6px 10px 10px;
  min-width: 0;
}

.composer-float__footer-leading,
.composer-float__footer-trailing {
  display: flex;
  align-items: center;
  gap: 4px;
  min-width: 0;
}

.composer-float__footer-trailing {
  flex: 1 1 auto;
  justify-content: flex-end;
  flex-shrink: 1;
  gap: 8px;
  min-width: 0;
  max-width: 100%;
}

.composer-float__tray {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  padding: 0 4px 2px;
  min-width: 0;
}

.composer-float__tray-leading {
  display: flex;
  flex-wrap: nowrap;
  align-items: center;
  gap: 4px;
  min-width: 0;
  overflow-x: auto;
  scrollbar-width: none;
}

.composer-float__tray-leading::-webkit-scrollbar {
  display: none;
}

.composer-agent-seg--compact {
  width: auto;
  flex-shrink: 0;
  border: none;
  background: transparent;
  padding: 0;
}

.composer-agent-seg--compact :deep(.dq-segmented__item) {
  padding: 4px 8px;
  font-size: var(--dq-font-size-caption);
  font-weight: 500;
  color: var(--dq-label-tertiary);
  border-radius: 6px;
}

.composer-agent-seg--compact :deep(.dq-segmented__item.is-active) {
  color: var(--dq-label-primary);
  background: color-mix(in srgb, var(--dq-label-primary) 8%, transparent);
  box-shadow: none;
}

.composer-meta-chip {
  display: inline-flex;
  align-items: center;
  height: 28px;
  padding: 0 8px;
  border-radius: 6px;
  border: none;
  font-size: var(--dq-font-size-caption);
  white-space: nowrap;
}

.composer-meta-chip--warn {
  color: var(--dq-system-orange);
  background: color-mix(in srgb, var(--dq-system-orange) 10%, transparent);
}

.composer-select {
  flex: 0 1 auto;
  width: auto !important;
  min-width: 0;
  max-width: 100%;
  display: block;
  overflow: hidden;
}

.composer-select :deep(.dq-select) {
  display: block;
  width: 100%;
  min-width: 0;
  max-width: 100%;
}

.composer-select :deep(.dq-select__trigger) {
  width: 100%;
  max-width: 100%;
  min-width: 0;
  min-height: 28px;
  padding: 2px 6px;
  font-size: var(--dq-font-size-caption);
  overflow: hidden;
}

.composer-select :deep(.dq-select__value) {
  flex: 1 1 auto;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.composer-select--project {
  flex: 0 1 160px;
  max-width: 200px;
}

.composer-select--model {
  flex: 1 1 100px;
  min-width: 64px;
  max-width: 180px;
}

.composer-select--effort {
  flex: 0 0 auto;
  width: max-content;
  min-width: 48px;
  max-width: 72px;
  overflow: visible;
}

.composer-select--effort :deep(.dq-select),
.composer-select--effort :deep(.dq-select__trigger) {
  width: auto;
}

.composer-select--agent {
  flex: 0 1 120px;
  max-width: 160px;
}

.composer-float__footer-trailing .composer-send {
  flex-shrink: 0;
}

.composer-tool-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  border: none;
  border-radius: 8px;
  background: transparent;
  color: var(--dq-label-tertiary);
  cursor: pointer;
  transition: background 0.12s ease, color 0.12s ease;
}

.composer-tool-btn:hover {
  background: color-mix(in srgb, var(--dq-label-primary) 8%, transparent);
  color: var(--dq-label-primary);
}

.composer-send {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  height: 32px;
  margin-left: 4px;
  padding: 0 12px 0 14px;
  border: none;
  border-radius: 10px;
  background: var(--dq-accent);
  color: var(--dq-color-white);
  font-size: var(--dq-font-size-footnote);
  font-weight: 650;
  cursor: pointer;
  transition: opacity 0.15s ease, transform 0.12s ease;
}

.composer-send:hover:not(:disabled) {
  filter: brightness(1.06);
}

.composer-send:active:not(:disabled) {
  transform: scale(0.98);
}

.composer-send:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}

.composer-send--stop {
  background: color-mix(in srgb, var(--dq-system-orange) 88%, #000);
}

.composer-send__kbd {
  display: inline-flex;
  align-items: center;
  height: 18px;
  padding: 0 5px;
  border-radius: 5px;
  background: color-mix(in srgb, #000 18%, transparent);
  font-family: var(--dq-font-mono, ui-monospace, monospace);
  font-size: 10px;
  font-weight: 600;
  opacity: 0.9;
}

@keyframes pulse {
  0%,
  100% {
    opacity: 1;
  }
  50% {
    opacity: 0.4;
  }
}
</style>
