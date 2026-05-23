<script setup lang="ts">
import { computed, nextTick, ref, watch } from 'vue'
import { useTasksStore } from '@/stores/tasks'
import StreamFeedItem from '@/components/center/StreamFeedItem.vue'
import { groupStreamByDay, withoutGoalDuplicate } from '@/utils/stream-feed'
import {
  canCancelTask,
  isTerminalTask,
  statusLabel,
  streamWaitingHint,
  tagTypeForStatus,
  terminalGoalDetail,
  terminalStatusDetail,
} from '@/utils/task-list'
import { confirm, toast } from '@/utils/feedback'
import TaskGoalRefresh from '@/components/center/TaskGoalRefresh.vue'

const tasks = useTasksStore()
const cancelling = ref(false)
const streamEl = ref<HTMLElement | null>(null)

const streamItems = computed(() => tasks.taskFeed)
const visibleStreamItems = computed(() =>
  withoutGoalDuplicate(streamItems.value, tasks.currentTask?.content),
)
const streamGroups = computed(() => groupStreamByDay(visibleStreamItems.value))

const showTerminate = computed(() => {
  const task = tasks.currentTask
  return task ? canCancelTask(task.status) : false
})

const showTaskRefresh = computed(
  () => tasks.currentTask != null && !isTerminalTask(tasks.currentTask.status),
)

const statusDetail = computed(() =>
  tasks.currentTaskTerminal
    ? terminalGoalDetail(tasks.currentTask, tasks.lastRefreshedAt)
    : terminalStatusDetail(tasks.currentTask),
)

const showLiveBadge = computed(
  () => showTaskRefresh.value && tasks.isPolling && !tasks.taskRefreshing,
)

const composePreview = computed(() => tasks.composeDraft.trim())

const streamWaitingMessage = computed(() => {
  const status = tasks.currentTask?.status ?? 'pending'
  return streamWaitingHint(status)
})

function scrollStreamToBottom(behavior: ScrollBehavior = 'smooth') {
  if (tasks.composingNew) return
  const el = streamEl.value
  if (!el) return
  el.scrollTo({ top: el.scrollHeight, behavior })
}

watch(
  () => [tasks.currentTaskId, visibleStreamItems.value.length, tasks.taskSwitching] as const,
  ([taskId, , switching]) => {
    if (!taskId || switching || tasks.composingNew) return
    void nextTick(() => scrollStreamToBottom(visibleStreamItems.value.length <= 1 ? 'auto' : 'smooth'))
  },
)

async function terminateTask() {
  if (!tasks.currentTaskId || cancelling.value) return
  try {
    await confirm(
      '确定终止当前任务？进行中的 Worker 将被停止，任务标记为失败。',
      '终止任务',
      { type: 'warning' },
    )
  } catch {
    return
  }
  cancelling.value = true
  try {
    await tasks.cancelTask()
    toast.success('任务已终止')
  } catch (e) {
    toast.error(e instanceof Error ? e.message : '终止失败')
  } finally {
    cancelling.value = false
  }
}
</script>

<template>
  <div class="task-stream-shell">
    <header v-if="tasks.composingNew" class="task-goal-bar task-goal-bar--compose">
      <div class="task-goal-bar__main">
        <p class="task-goal-bar__eyebrow">新建任务</p>
        <h1 class="task-goal-bar__title">
          {{ composePreview || '在 Composer 输入任务目标' }}
        </h1>
        <p class="task-goal-bar__detail">
          {{
            composePreview
              ? '发送后将创建任务并开始执行。'
              : '发送的第一条消息将作为任务目标，Team Controller 据此分派 Worker。'
          }}
        </p>
      </div>
    </header>

    <header v-else-if="tasks.currentTask" class="task-goal-bar" :class="{ 'task-goal-bar--terminal': tasks.currentTaskTerminal }">
      <div class="task-goal-bar__main">
        <p class="task-goal-bar__eyebrow">任务目标</p>
        <h1 class="task-goal-bar__title">{{ tasks.currentTask.content }}</h1>
        <p v-if="statusDetail" class="task-goal-bar__detail">{{ statusDetail }}</p>
      </div>
      <div class="task-goal-bar__actions">
        <div class="task-goal-bar__status">
          <span v-if="showLiveBadge" class="task-goal-bar__live" aria-hidden="true" />
          <DqTag size="small" :type="tagTypeForStatus(tasks.currentTask.status)">
            {{ statusLabel(tasks.currentTask.status) }}
          </DqTag>
        </div>
        <template v-if="showTaskRefresh">
          <span class="task-goal-bar__sep" aria-hidden="true" />
          <TaskGoalRefresh />
        </template>
        <DqButton
          v-if="showTerminate"
          size="sm"
          :disabled="cancelling"
          @click="terminateTask"
        >
          终止任务
        </DqButton>
      </div>
    </header>

    <div ref="streamEl" class="task-stream" role="main" aria-label="Team task stream">
      <div v-if="tasks.composingNew" class="task-stream__empty task-stream__empty--compose">
        <p class="task-stream__empty-title">
          {{ composePreview ? '确认任务目标' : '描述你要解决的问题' }}
        </p>
        <p class="task-stream__empty-hint">
          {{
            composePreview
              ? '按 Enter 或点击发送创建任务。'
              : '例如：「集群 ses-01 节点 CPU 负载高，请分析根因并给出止血建议」'
          }}
        </p>
      </div>

      <div v-else-if="!tasks.currentTaskId" class="task-stream__empty">
        <p class="task-stream__empty-title">Team Canvas</p>
        <p class="task-stream__empty-hint">
          从左侧选择任务，或点击「新建」开始新任务。
        </p>
      </div>

      <div v-else-if="tasks.taskSwitching" class="task-stream__empty task-stream__empty--compact">
        <p class="task-stream__empty-hint">加载任务流…</p>
      </div>

      <div v-else-if="!visibleStreamItems.length" class="task-stream__empty task-stream__empty--compact">
        <p class="task-stream__empty-hint">{{ streamWaitingMessage }}</p>
      </div>

      <div v-else class="task-stream__list">
        <template v-for="group in streamGroups" :key="group.id">
          <div v-if="group.label" class="task-stream__day" role="separator">
            <span class="task-stream__day-label">{{ group.label }}</span>
          </div>
          <StreamFeedItem v-for="item in group.items" :key="item.id" :item="item">
            <template v-if="item.approvalId" #approval-actions>
              <DqButton type="primary" size="sm" @click="tasks.approve(item.approvalId!, true)">
                批准
              </DqButton>
              <DqButton size="sm" @click="tasks.approve(item.approvalId!, false)">拒绝</DqButton>
            </template>
          </StreamFeedItem>
        </template>
        <div class="task-stream__anchor" aria-hidden="true" />
      </div>
    </div>
  </div>
</template>
