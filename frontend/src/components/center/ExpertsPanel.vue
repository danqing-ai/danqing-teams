<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { StreamEvent } from '@/types/mission'

const { t } = useI18n()

const props = defineProps<{
  streamEvents: StreamEvent[]
}>()

interface Expert {
  turnId: string
  agentId: string
  agentName: string
  goal: string
  status: 'running' | 'completed' | 'failed' | 'cancelled'
  stepsUsed: number
}

function asRecord(v: unknown): Record<string, unknown> | null {
  if (v && typeof v === 'object' && !Array.isArray(v)) return v as Record<string, unknown>
  return null
}

const experts = computed<Expert[]>(() => {
  const map: Record<string, Expert> = {}

  for (const ev of props.streamEvents) {
    if (ev.type === 'delegate.started') {
      const payload = asRecord(ev.payload)
      if (!payload) continue
      const childTurnId = String(payload.childTurnId ?? '')
      if (!childTurnId) continue
      if (!map[childTurnId]) {
        map[childTurnId] = {
          turnId: childTurnId,
          agentId: String(payload.agentId ?? ''),
          agentName: String(payload.agentId ?? 'AI'),
          goal: String(payload.goal ?? ''),
          status: 'running',
          stepsUsed: 0,
        }
      } else {
        // Update in case started arrived after other events
        map[childTurnId].agentId = String(payload.agentId ?? map[childTurnId].agentId)
        map[childTurnId].agentName = String(payload.agentId ?? map[childTurnId].agentName)
        map[childTurnId].goal = String(payload.goal ?? map[childTurnId].goal)
      }
      continue
    }

    if (ev.type === 'delegate.completed') {
      const payload = asRecord(ev.payload)
      if (!payload) continue
      // Find the expert by agentId (delegate.completed doesn't carry childTurnId)
      const agentId = String(payload.agentId ?? '')
      for (const expert of Object.values(map)) {
        if (expert.agentId === agentId && expert.status === 'running') {
          expert.status = (String(payload.status ?? '') as Expert['status']) || 'completed'
          break
        }
      }
      continue
    }

    // Count tool steps per child turn
    if (ev.type === 'tool.completed') {
      // tool.completed events for child turns have their own turnId
      const turnId = ev.turnId
      if (turnId && map[turnId]) {
        map[turnId].stepsUsed++
      }
    }
  }

  return Object.values(map).sort((a, b) => a.turnId.localeCompare(b.turnId))
})

const stats = computed(() => {
  const total = experts.value.length
  const running = experts.value.filter((e) => e.status === 'running').length
  const completed = experts.value.filter((e) => e.status === 'completed').length
  const failed = experts.value.filter((e) => e.status === 'failed').length
  return { total, running, completed, failed }
})

function statusLabel(status: Expert['status']): string {
  switch (status) {
    case 'running': return t('sessions.running')
    case 'completed': return t('sessions.completed')
    case 'failed': return t('sessions.failed')
    case 'cancelled': return t('sessions.cancelled')
  }
}

function statusClass(status: Expert['status']): string {
  switch (status) {
    case 'running': return 'expert--running'
    case 'completed': return 'expert--completed'
    case 'failed': return 'expert--failed'
    case 'cancelled': return 'expert--cancelled'
  }
}
</script>

<template>
  <aside class="experts-panel">
    <div v-if="!experts.length" class="experts-panel__empty">
      <p>{{ t('sessions.noExperts') }}</p>
      <p class="experts-panel__hint">{{ t('sessions.noExpertsHint') }}</p>
    </div>

    <div v-else class="experts-panel__list">
      <div
        v-for="expert in experts"
        :key="expert.turnId"
        class="expert"
        :class="statusClass(expert.status)"
      >
        <div class="expert__head">
          <div class="expert__identity">
            <span class="expert__avatar">{{ expert.agentName.charAt(0).toUpperCase() }}</span>
            <span class="expert__name">{{ expert.agentName }}</span>
          </div>
          <span class="expert__status-badge">{{ statusLabel(expert.status) }}</span>
        </div>
        <div class="expert__goal" :title="expert.goal">{{ expert.goal }}</div>
        <div class="expert__meta">
          <span class="expert__steps">{{ expert.stepsUsed }} steps</span>
        </div>
      </div>
    </div>

    <div v-if="experts.length" class="experts-panel__footer">
      <span class="experts-panel__stats">
        {{ stats.completed }}/{{ stats.total }}
        <span v-if="stats.running" class="experts-panel__running">· {{ stats.running }} running</span>
      </span>
    </div>
  </aside>
</template>

<style scoped>
.experts-panel {
  display: flex;
  flex-direction: column;
  height: 100%;
  min-height: 0;
  overflow: hidden;
  background: var(--teams-glass-bg, rgba(255, 255, 255, 0.04));
  font-size: 13px;
}

.experts-panel__empty {
  padding: 32px 16px;
  text-align: center;
  color: var(--dq-label-tertiary);
}

.experts-panel__hint {
  margin-top: 8px;
  font-size: 11px;
  color: var(--dq-label-quaternary);
  line-height: 1.5;
}

.experts-panel__list {
  flex: 1;
  overflow-y: auto;
  padding: 10px;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.expert {
  padding: 12px;
  border-radius: 10px;
  background: color-mix(in srgb, var(--dq-label-primary) 3%, transparent);
  border: 1px solid color-mix(in srgb, var(--dq-label-primary) 6%, transparent);
  transition: border-color 0.15s ease, background 0.15s ease;
}

.expert:hover {
  border-color: color-mix(in srgb, var(--dq-label-primary) 12%, transparent);
  background: color-mix(in srgb, var(--dq-label-primary) 5%, transparent);
}

.expert__head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}

.expert__identity {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
}

.expert__avatar {
  flex-shrink: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  width: 24px;
  height: 24px;
  border-radius: 6px;
  font-size: 11px;
  font-weight: 700;
  color: var(--dq-bg-page);
  background: var(--dq-accent);
  text-transform: uppercase;
}

.expert--completed .expert__avatar {
  background: var(--dq-success);
}

.expert--failed .expert__avatar {
  background: var(--dq-danger, #ff453a);
}

.expert--cancelled .expert__avatar {
  background: var(--dq-label-tertiary);
}

.expert__name {
  font-weight: 600;
  font-size: 13px;
  color: var(--dq-label-primary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.expert__status-badge {
  flex-shrink: 0;
  font-size: 10px;
  font-weight: 600;
  padding: 2px 8px;
  border-radius: 10px;
  background: color-mix(in srgb, var(--dq-label-primary) 8%, transparent);
  color: var(--dq-label-secondary);
  letter-spacing: 0.02em;
}

.expert--running .expert__status-badge {
  background: color-mix(in srgb, var(--dq-accent) 15%, transparent);
  color: var(--dq-accent);
}

.expert--completed .expert__status-badge {
  background: color-mix(in srgb, var(--dq-success) 15%, transparent);
  color: var(--dq-success);
}

.expert--failed .expert__status-badge {
  background: color-mix(in srgb, var(--dq-danger, #ff453a) 15%, transparent);
  color: var(--dq-danger, #ff453a);
}

.expert--cancelled .expert__status-badge {
  background: color-mix(in srgb, var(--dq-label-tertiary) 15%, transparent);
  color: var(--dq-label-tertiary);
}

.expert__goal {
  margin-top: 8px;
  font-size: 11px;
  color: var(--dq-label-secondary);
  line-height: 1.5;
  word-break: break-word;
  display: -webkit-box;
  -webkit-line-clamp: 4;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.expert__meta {
  margin-top: 8px;
  display: flex;
  align-items: center;
  gap: 8px;
}

.expert__steps {
  font-size: 10px;
  color: var(--dq-label-tertiary);
  font-family: ui-monospace, SFMono-Regular, 'SF Mono', Menlo, monospace;
  background: color-mix(in srgb, var(--dq-label-primary) 5%, transparent);
  padding: 2px 6px;
  border-radius: 4px;
}

.experts-panel__footer {
  flex-shrink: 0;
  padding: 10px 12px;
  border-top: 1px solid var(--dq-separator-light, rgba(0, 0, 0, 0.06));
}

.experts-panel__stats {
  font-size: 11px;
  font-weight: 600;
  color: var(--dq-label-secondary);
}

.experts-panel__running {
  color: var(--dq-accent);
}
</style>
