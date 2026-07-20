<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useModelConfigStore } from '@/stores/modelLimits'
import { useSessionsStore } from '@/stores/sessions'
import { formatTokenCount, useSessionContextUsage } from '@/composables/useSessionContextUsage'

const { t } = useI18n()
const sessions = useSessionsStore()
const modelConfig = useModelConfigStore()
const showHistory = ref(false)

const {
  usedTokens,
  contextWindow,
  usageRatio,
  usageLevel,
  compactionHistory,
  sessionTotalTokens,
} = useSessionContextUsage()

onMounted(() => {
  if (!modelConfig.models.length) void modelConfig.load()
})

const percentLabel = computed(() => `${Math.round(usageRatio.value * 100)}%`)
const hasData = computed(() => usedTokens.value > 0 || compactionHistory.value.length > 0)
</script>

<template>
  <div
    v-if="sessions.currentSession"
    class="context-usage"
    :class="[`is-${usageLevel}`, { 'is-empty': !hasData }]"
  >
    <div v-if="showHistory && compactionHistory.length" class="context-usage__history">
      <div
        v-for="c in compactionHistory.slice().reverse()"
        :key="c.seq"
        class="context-usage__history-row"
      >
        {{ t('sessions.compactionRow', {
          turns: c.turnsCompacted,
          before: formatTokenCount(c.tokensBefore),
          after: formatTokenCount(c.tokensAfter),
        }) }}
      </div>
      <div class="context-usage__history-total">
        {{ t('sessions.sessionTokensTotal', { n: formatTokenCount(sessionTotalTokens) }) }}
      </div>
    </div>
    <div class="context-usage__main" :title="t('sessions.contextUsageHint')">
      <svg class="context-usage__icon" viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
        <path d="M12 2v4M12 18v4M4.93 4.93l2.83 2.83M16.24 16.24l2.83 2.83M2 12h4M18 12h4M4.93 19.07l2.83-2.83M16.24 7.76l2.83-2.83" />
        <circle cx="12" cy="12" r="3" />
      </svg>
      <div class="context-usage__track" aria-hidden="true">
        <div class="context-usage__fill" :style="{ width: `${Math.round(usageRatio * 100)}%` }" />
      </div>
      <span class="context-usage__label">
        {{ formatTokenCount(usedTokens) }} / {{ formatTokenCount(contextWindow) }}
        <span class="context-usage__pct">({{ percentLabel }})</span>
      </span>
      <button
        v-if="compactionHistory.length"
        type="button"
        class="context-usage__history-btn"
        @click="showHistory = !showHistory"
      >
        {{ t('sessions.compactionCount', { n: compactionHistory.length }) }}
      </button>
    </div>
  </div>
</template>

<style scoped>
.context-usage {
  flex-shrink: 0;
  padding: 0 4px 2px;
}

.context-usage__main {
  display: flex;
  align-items: center;
  gap: 8px;
  min-height: 28px;
}

.context-usage__icon {
  flex-shrink: 0;
  color: var(--dq-label-tertiary);
}

.context-usage.is-warn .context-usage__icon,
.context-usage.is-warn .context-usage__pct {
  color: var(--dq-system-orange);
}

.context-usage.is-critical .context-usage__icon,
.context-usage.is-critical .context-usage__pct {
  color: var(--dq-danger);
}

.context-usage__track {
  flex: 1;
  max-width: 120px;
  height: 4px;
  border-radius: 2px;
  background: color-mix(in srgb, var(--dq-label-primary) 10%, transparent);
  overflow: hidden;
}

.context-usage__fill {
  height: 100%;
  border-radius: 2px;
  background: var(--dq-accent);
  transition: width 0.25s ease;
}

.context-usage.is-warn .context-usage__fill {
  background: var(--dq-system-orange);
}

.context-usage.is-critical .context-usage__fill {
  background: var(--dq-danger);
}

.context-usage__label {
  font-size: var(--dq-font-size-caption);
  font-family: var(--dq-font-mono, ui-monospace, monospace);
  color: var(--dq-label-secondary);
  white-space: nowrap;
}

.context-usage__pct {
  color: var(--dq-label-tertiary);
}

.context-usage__history-btn {
  margin-left: auto;
  padding: 2px 8px;
  border: none;
  border-radius: 6px;
  background: color-mix(in srgb, var(--dq-label-primary) 6%, transparent);
  color: var(--dq-label-secondary);
  font-size: var(--dq-font-size-caption);
  cursor: pointer;
}

.context-usage__history-btn:hover {
  background: color-mix(in srgb, var(--dq-label-primary) 10%, transparent);
}

.context-usage__history {
  margin-bottom: 6px;
  padding: 8px 10px;
  border-radius: 8px;
  background: color-mix(in srgb, var(--dq-glass-popover-bg, var(--dq-bg-elevated)) 92%, transparent);
  border: 1px solid var(--dq-separator-light);
  backdrop-filter: blur(8px);
  font-size: var(--dq-font-size-caption);
  color: var(--dq-label-secondary);
  line-height: 1.5;
}

.context-usage__history-row + .context-usage__history-row {
  margin-top: 4px;
}

.context-usage__history-total {
  margin-top: 6px;
  padding-top: 6px;
  border-top: 1px solid var(--dq-separator-light);
  color: var(--dq-label-tertiary);
}

.context-usage.is-empty .context-usage__fill {
  width: 0 !important;
}
</style>
