<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'

export interface ToolCardPayload {
  name: string
  description?: string
  status: string
  inputStr?: string
  output?: string
  error?: string
  stepNum?: number
}

const props = defineProps<{
  card: ToolCardPayload
  expanded: boolean
  awaitingApproval?: boolean
  awaitingLabel?: string
  childLinkLabel?: string
  showChildLink?: boolean
}>()

const emit = defineEmits<{
  toggle: []
  openChild: []
}>()

const { t } = useI18n()

function truncateText(s: string, max = 200): string {
  if (s.length <= max) return s
  return s.slice(0, max) + '…'
}

function toolInputFields(inputStr?: string): { key: string; value: string }[] | null {
  if (!inputStr) return null
  try {
    const obj = JSON.parse(inputStr) as unknown
    if (!obj || typeof obj !== 'object' || Array.isArray(obj)) return null
    return Object.entries(obj as Record<string, unknown>).map(([key, value]) => ({
      key,
      value: typeof value === 'string' ? value : JSON.stringify(value),
    }))
  } catch {
    return null
  }
}

const inputSummary = computed(() => {
  const fields = toolInputFields(props.card.inputStr)
  if (fields?.length) {
    const first = fields[0]
    return `${first.key}: ${truncateText(first.value, 80)}`
  }
  if (props.card.inputStr) return truncateText(props.card.inputStr, 100)
  return props.card.description || ''
})

const outputSummary = computed(() => {
  if (props.card.error) return truncateText(props.card.error, 100)
  if (props.card.output) return truncateText(props.card.output, 100)
  return ''
})

const fields = computed(() => toolInputFields(props.card.inputStr))
</script>

<template>
  <div
    class="tool-card"
    :class="{
      'is-expanded': expanded,
      'is-running': card.status === 'running' && !awaitingApproval,
      'is-awaiting-approval': awaitingApproval,
      'is-completed': card.status === 'completed',
      'is-error': card.status === 'error',
      'is-cancelled': card.status === 'cancelled',
    }"
  >
    <div class="tool-card__header" @click="emit('toggle')">
      <div class="tool-card__meta">
        <span class="tool-card__name">{{ card.name }}</span>
        <span v-if="!expanded && inputSummary" class="tool-card__desc">{{ inputSummary }}</span>
      </div>
      <div class="tool-card__actions">
        <span v-if="awaitingApproval" class="tool-card__status-badge is-awaiting">
          <span class="tool-card__spinner" />
          <span>{{ awaitingLabel || t('sessions.awaitingApproval') }}</span>
        </span>
        <span v-else-if="card.status === 'running'" class="tool-card__status-badge is-running">
          <span class="tool-card__spinner" />
          <span>{{ t('sessions.running') }}</span>
        </span>
        <span v-else-if="card.status === 'cancelled'" class="tool-card__status-badge is-cancelled">
          {{ t('sessions.cancelled') }}
        </span>
        <span v-else-if="card.status === 'error'" class="tool-card__status-badge is-error">
          {{ t('sessions.failed') }}
        </span>
        <span v-else-if="card.status === 'completed'" class="tool-card__status-badge is-done">
          {{ t('sessions.completed') }}
        </span>
        <span
          v-if="showChildLink"
          class="tool-card__link"
          :class="{ 'is-awaiting': awaitingApproval }"
          @click.stop="emit('openChild')"
        >{{ childLinkLabel || t('sessions.viewSubagent') }}</span>
        <svg
          class="tool-card__chevron"
          :class="{ 'is-open': expanded }"
          viewBox="0 0 24 24"
          width="14"
          height="14"
          fill="none"
          stroke="currentColor"
          stroke-width="2"
          stroke-linecap="round"
          stroke-linejoin="round"
        >
          <polyline points="6 9 12 15 18 9" />
        </svg>
      </div>
    </div>

    <div v-if="!expanded && outputSummary && card.status !== 'running'" class="tool-card__preview">
      {{ outputSummary }}
    </div>

    <div v-show="expanded" class="tool-card__body">
      <div v-if="card.inputStr" class="tool-card__section">
        <span class="tool-card__section-label">{{ t('sessions.toolInput') }}</span>
        <div v-if="fields" class="tool-card__fields">
          <div v-for="field in fields" :key="field.key" class="tool-card__field">
            <span class="tool-card__field-key">{{ field.key }}</span>
            <span class="tool-card__field-val" :title="field.value">{{ truncateText(field.value) }}</span>
          </div>
        </div>
        <pre v-else class="tool-card__code">{{ card.inputStr }}</pre>
      </div>
      <div v-if="card.output" class="tool-card__section">
        <span class="tool-card__section-label">{{ t('sessions.toolOutput') }}</span>
        <pre class="tool-card__code">{{ card.output }}</pre>
      </div>
      <div v-if="card.error" class="tool-card__section tool-card__section--error">
        <span class="tool-card__section-label">{{ t('sessions.toolError') }}</span>
        <pre class="tool-card__code">{{ card.error }}</pre>
      </div>
    </div>
  </div>
</template>

<style scoped>
.tool-card {
  border-radius: 10px;
  border: 1px solid color-mix(in srgb, var(--dq-label-primary) 8%, transparent);
  background: color-mix(in srgb, var(--dq-label-primary) 3%, transparent);
  overflow: hidden;
}

.tool-card.is-running,
.tool-card.is-awaiting-approval {
  border-color: color-mix(in srgb, var(--dq-accent) 35%, transparent);
}

.tool-card.is-error {
  border-color: color-mix(in srgb, var(--dq-danger) 35%, transparent);
}

.tool-card.is-cancelled {
  border-color: color-mix(in srgb, var(--dq-label-primary) 12%, transparent);
  opacity: 0.85;
}

.tool-card__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  padding: 8px 10px;
  cursor: pointer;
}

.tool-card__meta {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
  flex: 1;
}

.tool-card__name {
  flex-shrink: 0;
  font-weight: 600;
  font-size: var(--dq-font-size-footnote);
  font-family: var(--dq-font-mono, ui-monospace, monospace);
  color: var(--dq-label-primary);
}

.tool-card__desc {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: var(--dq-font-size-caption);
  color: var(--dq-label-tertiary);
}

.tool-card__actions {
  display: flex;
  align-items: center;
  gap: 6px;
  flex-shrink: 0;
}

.tool-card__status-badge {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 2px 7px;
  border-radius: 8px;
  font-size: var(--dq-font-size-caption);
  font-weight: 600;
}

.tool-card__status-badge.is-running,
.tool-card__status-badge.is-awaiting {
  background: color-mix(in srgb, var(--dq-accent) 12%, transparent);
  color: var(--dq-accent);
}

.tool-card__status-badge.is-error {
  background: color-mix(in srgb, var(--dq-danger) 12%, transparent);
  color: var(--dq-danger);
}

.tool-card__status-badge.is-cancelled {
  background: color-mix(in srgb, var(--dq-label-primary) 8%, transparent);
  color: var(--dq-label-secondary);
}

.tool-card__status-badge.is-done {
  background: color-mix(in srgb, var(--dq-success) 12%, transparent);
  color: var(--dq-success);
}

.tool-card__spinner {
  width: 8px;
  height: 8px;
  border: 1.5px solid currentColor;
  border-right-color: transparent;
  border-radius: 50%;
  animation: tool-spin 0.7s linear infinite;
}

.tool-card__link {
  font-size: var(--dq-font-size-caption);
  color: var(--dq-accent);
  cursor: pointer;
}

.tool-card__chevron {
  color: var(--dq-label-tertiary);
  transition: transform 0.15s ease;
}

.tool-card__chevron.is-open {
  transform: rotate(180deg);
}

.tool-card__preview {
  padding: 0 10px 8px;
  font-size: var(--dq-font-size-caption);
  color: var(--dq-label-tertiary);
  line-height: 1.4;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.tool-card__body {
  padding: 0 10px 10px;
  border-top: 1px solid color-mix(in srgb, var(--dq-label-primary) 6%, transparent);
}

.tool-card__section {
  margin-top: 8px;
}

.tool-card__section-label {
  display: block;
  margin-bottom: 4px;
  font-size: var(--dq-font-size-caption);
  font-weight: 600;
  color: var(--dq-label-quaternary);
  text-transform: uppercase;
  letter-spacing: 0.04em;
}

.tool-card__fields {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.tool-card__field {
  display: grid;
  grid-template-columns: 88px 1fr;
  gap: 8px;
  font-size: var(--dq-font-size-caption);
}

.tool-card__field-key {
  color: var(--dq-label-tertiary);
  font-family: var(--dq-font-mono, ui-monospace, monospace);
}

.tool-card__field-val {
  color: var(--dq-label-secondary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.tool-card__code {
  margin: 0;
  padding: 8px;
  border-radius: 6px;
  background: color-mix(in srgb, var(--dq-label-primary) 4%, transparent);
  font-size: var(--dq-font-size-caption);
  font-family: var(--dq-font-mono, ui-monospace, monospace);
  color: var(--dq-label-secondary);
  white-space: pre-wrap;
  word-break: break-word;
  max-height: 240px;
  overflow: auto;
}

.tool-card__section--error .tool-card__code {
  color: var(--dq-danger);
}

@keyframes tool-spin {
  to {
    transform: rotate(360deg);
  }
}
</style>
