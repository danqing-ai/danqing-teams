<script setup lang="ts">
import { computed } from 'vue'
import type { StreamEvent } from '@/types/mission'

const props = defineProps<{
  streamEvents: StreamEvent[]
}>()

interface TodoItem {
  content: string
  status: 'pending' | 'in_progress' | 'completed' | 'cancelled'
  priority: 'high' | 'medium' | 'low'
}

const todos = computed<TodoItem[]>(() => {
  for (let i = props.streamEvents.length - 1; i >= 0; i--) {
    const ev = props.streamEvents[i]
    if (ev.type !== 'tool.running') continue
    const p = ev.payload as Record<string, unknown> | null
    if (p?.name !== 'todowrite') continue
    const input = p?.input as Record<string, unknown> | null
    if (!input?.todos) continue
    const items = input.todos as unknown[]
    if (!Array.isArray(items) || items.length === 0) continue
    return items.map((item: any) => ({
      content: String(item.content ?? ''),
      status: String(item.status ?? 'pending') as TodoItem['status'],
      priority: String(item.priority ?? 'medium') as TodoItem['priority'],
    }))
  }
  return []
})

const groups = computed(() => {
  const items = todos.value
  const inProgress = items.filter((t) => t.status === 'in_progress')
  const pending = items.filter((t) => t.status === 'pending')
  const cancelled = items.filter((t) => t.status === 'cancelled')
  const completed = items.filter((t) => t.status === 'completed')
  return [
    { key: 'in_progress', label: '进行中', items: inProgress },
    { key: 'pending', label: '待处理', items: pending },
    { key: 'cancelled', label: '已取消', items: cancelled },
    { key: 'completed', label: '已完成', items: completed },
  ].filter((g) => g.items.length > 0)
})

const stats = computed(() => {
  const items = todos.value
  return {
    total: items.length,
    inProgress: items.filter((t) => t.status === 'in_progress').length,
    pending: items.filter((t) => t.status === 'pending').length,
    completed: items.filter((t) => t.status === 'completed').length,
    cancelled: items.filter((t) => t.status === 'cancelled').length,
  }
})

function statusIcon(status: TodoItem['status']): string {
  switch (status) {
    case 'completed': return '[x]'
    case 'in_progress': return '[>]'
    case 'cancelled': return '[-]'
    default: return '[ ]'
  }
}

function priorityColor(priority: TodoItem['priority']): string {
  switch (priority) {
    case 'high': return '#e03e2d'
    case 'medium': return '#d4a017'
    case 'low': return '#2d8a56'
  }
}
</script>

<template>
  <aside class="plan-panel">
    <div class="plan-panel__head">
      <span v-if="todos.length" class="plan-panel__stats">{{ stats.completed }}/{{ stats.total }}</span>
    </div>

    <div v-if="!todos.length" class="plan-panel__empty">
      <p>暂无计划</p>
      <p class="plan-panel__hint">Agent 会在复杂任务时自动生成计划</p>
    </div>

    <div v-else class="plan-panel__list">
      <div
        v-for="group in groups"
        :key="group.key"
        class="plan-panel__group"
      >
        <div class="plan-panel__group-label">{{ group.label }}</div>
        <div
          v-for="(item, idx) in group.items"
          :key="idx"
          class="plan-panel__item"
          :class="{ 'is-done': item.status === 'completed', 'is-cancelled': item.status === 'cancelled' }"
        >
          <span
            class="plan-panel__item-icon"
            :style="{ color: priorityColor(item.priority) }"
          >
            {{ statusIcon(item.status) }}
          </span>
          <span class="plan-panel__item-text">{{ item.content }}</span>
        </div>
      </div>
    </div>

    <div v-if="todos.length" class="plan-panel__progress">
      <div class="plan-panel__progress-track">
        <div
          class="plan-panel__progress-bar"
          :style="{ width: stats.total ? (stats.completed / stats.total * 100) + '%' : '0%' }"
        />
      </div>
    </div>
  </aside>
</template>

<style scoped>
.plan-panel {
  display: flex;
  flex-direction: column;
  height: 100%;
  min-height: 0;
  overflow: hidden;
  background: var(--teams-glass-bg, rgba(255, 255, 255, 0.04));
  font-size: 13px;
}

.plan-panel__head {
  flex-shrink: 0;
  display: flex;
  align-items: center;
  justify-content: flex-end;
  padding: 6px 12px 4px;
  border-bottom: 1px solid var(--dq-separator-light, rgba(0, 0, 0, 0.06));
}

.plan-panel__stats {
  font-size: 11px;
  font-weight: 500;
  color: var(--dq-label-secondary);
}

.plan-panel__empty {
  padding: 24px 16px;
  text-align: center;
  color: var(--dq-label-tertiary);
}

.plan-panel__hint {
  margin-top: 6px;
  font-size: 11px;
  color: var(--dq-label-quaternary);
  line-height: 1.5;
}

.plan-panel__list {
  flex: 1;
  overflow-y: auto;
  padding: 6px 0;
}

.plan-panel__group {
  padding: 2px 0;
}

.plan-panel__group-label {
  padding: 6px 16px 2px;
  font-size: 10px;
  font-weight: 600;
  letter-spacing: 0.06em;
  text-transform: uppercase;
  color: var(--dq-label-quaternary);
}

.plan-panel__item {
  display: flex;
  align-items: flex-start;
  gap: 6px;
  padding: 5px 16px;
  line-height: 1.45;
  color: var(--dq-label-primary);
}

.plan-panel__item.is-done {
  color: var(--dq-label-tertiary);
}

.plan-panel__item.is-done .plan-panel__item-text {
  text-decoration: line-through;
}

.plan-panel__item.is-cancelled {
  color: var(--dq-label-tertiary);
  opacity: 0.6;
}

.plan-panel__item.is-cancelled .plan-panel__item-text {
  text-decoration: line-through;
}

.plan-panel__item-icon {
  flex-shrink: 0;
  font-family: monospace;
  font-size: 12px;
  line-height: 1.45;
}

.plan-panel__item-text {
  word-break: break-word;
}

.plan-panel__progress {
  flex-shrink: 0;
  padding: 8px 16px;
  border-top: 1px solid var(--dq-separator-light, rgba(0, 0, 0, 0.06));
}

.plan-panel__progress-track {
  height: 3px;
  border-radius: 2px;
  background: var(--dq-bg-container-hover, rgba(0, 0, 0, 0.06));
  overflow: hidden;
}

.plan-panel__progress-bar {
  height: 100%;
  border-radius: 2px;
  background: var(--dq-accent);
  transition: width 0.3s ease;
}
</style>
