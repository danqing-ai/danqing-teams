<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useTasksStore } from '@/stores/tasks'
import { isTerminalTask } from '@/utils/task-list'

const tasks = useTasksStore()
const spinKey = ref(0)

watch(
  () => tasks.taskRefreshing,
  (v, prev) => {
    if (v && !prev) spinKey.value += 1
  },
)

const intervalSec = computed(() => Math.round(tasks.currentPollInterval / 1000))
const countdownSec = computed(() => Math.max(0, Math.ceil(tasks.pollCountdownMs / 1000)))
const isIdlePoll = computed(() => isTerminalTask(tasks.currentTask?.status ?? ''))

const progressOffset = computed(() => {
  const circumference = 2 * Math.PI * 10
  return circumference * (1 - tasks.pollProgress)
})

const showCountdown = computed(
  () => tasks.isPolling && !tasks.taskRefreshing && countdownSec.value > 0,
)

const tooltip = computed(() => {
  if (tasks.taskRefreshing) return '正在同步任务流'
  if (!tasks.isPolling) return '自动刷新已暂停 · 点击立即刷新'
  const pace = isIdlePoll.value ? '慢速' : '实时'
  return `${pace} · 每 ${intervalSec.value} 秒自动刷新 · 点击立即刷新`
})

async function onRefresh() {
  await tasks.refreshTaskNow()
}
</script>

<template>
  <button
    type="button"
    class="task-goal-refresh"
    :class="{
      'is-live': tasks.isPolling && !isIdlePoll,
      'is-syncing': tasks.taskRefreshing,
    }"
    :disabled="tasks.taskRefreshing || tasks.taskSwitching"
    :title="tooltip"
    :aria-label="tooltip"
    @click="onRefresh"
  >
    <svg class="task-goal-refresh__ring" viewBox="0 0 24 24" aria-hidden="true">
      <circle
        class="task-goal-refresh__track"
        cx="12"
        cy="12"
        r="10"
        fill="none"
      />
      <circle
        class="task-goal-refresh__progress"
        cx="12"
        cy="12"
        r="10"
        fill="none"
        stroke-linecap="round"
        :stroke-dasharray="62.83"
        :stroke-dashoffset="progressOffset"
        transform="rotate(-90 12 12)"
      />
    </svg>

    <svg
      v-if="tasks.taskRefreshing"
      :key="spinKey"
      class="task-goal-refresh__glyph is-spinning"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      stroke-width="2"
      stroke-linecap="round"
      stroke-linejoin="round"
      aria-hidden="true"
    >
      <path d="M21 12a9 9 0 1 1-2.64-6.36" />
      <path d="M21 3v6h-6" />
    </svg>
    <span v-else-if="showCountdown" class="task-goal-refresh__count" aria-hidden="true">
      {{ countdownSec }}
    </span>
    <svg
      v-else
      class="task-goal-refresh__glyph"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      stroke-width="2"
      stroke-linecap="round"
      stroke-linejoin="round"
      aria-hidden="true"
    >
      <path d="M21 12a9 9 0 1 1-2.64-6.36" />
      <path d="M21 3v6h-6" />
    </svg>
  </button>
</template>
