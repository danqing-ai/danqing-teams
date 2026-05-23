import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import { asArray, fetchJSON } from '@/api/client'
import { buildStreamFeed, type StreamFeedItem } from '@/utils/stream-actors'
import { isTerminalTask } from '@/utils/task-list'
import { useTeamsStore } from './teams'
import { useTodosStore } from './todos'
import { useWorkspaceStore } from './workspace'
import type {
  TeamTask,
  TeamMessage,
  TimelineEvent,
  ApprovalRequest,
  SendTeamMessageResponse,
} from '@/types'

export type SendMessageOptions = {
  /** 强制开新任务线程（忽略当前选中任务） */
  forceNew?: boolean
  /** 显式指定任务 ID（须属于当前 Team） */
  taskId?: string
}

const ACTIVE_POLL_MS = 2000
const IDLE_POLL_MS = 8000

export { ACTIVE_POLL_MS, IDLE_POLL_MS }

function taskIsTerminal(status: string) {
  return isTerminalTask(status)
}

export const useTasksStore = defineStore('tasks', () => {
  const tasks = ref<TeamTask[]>([])
  const messages = ref<TeamMessage[]>([])
  const currentTaskId = ref('')
  const composingNew = ref(false)
  const composeDraft = ref('')
  const composeReturnTaskId = ref('')
  const taskSwitching = ref(false)
  const taskRefreshing = ref(false)
  const isPolling = ref(false)
  const pollCountdownMs = ref(0)
  const lastRefreshedAt = ref<number | null>(null)
  const timeline = ref<TimelineEvent[]>([])
  const pendingApprovals = ref<ApprovalRequest[]>([])
  let pollHandle: ReturnType<typeof setTimeout> | null = null
  let countdownHandle: ReturnType<typeof setInterval> | null = null
  let pollDeadline = 0

  const currentTask = computed(() =>
    tasks.value.find((t) => t.id === currentTaskId.value),
  )

  const currentTaskApprovals = computed(() =>
    pendingApprovals.value.filter((a) => a.taskId === currentTaskId.value),
  )

  /** 合并 messages + timeline + 待审批，按时间整理后的任务事件流 */
  const taskFeed = computed<StreamFeedItem[]>(() =>
    buildStreamFeed(
      messages.value,
      timeline.value,
      currentTaskApprovals.value.map((a) => ({ id: a.id, summary: a.summary })),
    ),
  )

  function pollIntervalMs() {
    const status = currentTask.value?.status ?? ''
    return taskIsTerminal(status) ? IDLE_POLL_MS : ACTIVE_POLL_MS
  }

  const currentPollInterval = computed(() =>
    taskIsTerminal(currentTask.value?.status ?? '') ? 0 : pollIntervalMs(),
  )

  const currentTaskTerminal = computed(() =>
    taskIsTerminal(currentTask.value?.status ?? ''),
  )

  const pollProgress = computed(() => {
    const total = currentPollInterval.value
    if (!isPolling.value || total <= 0) return 0
    return Math.min(1, Math.max(0, 1 - pollCountdownMs.value / total))
  })

  function shouldAutoRefresh(status?: string) {
    return !taskIsTerminal(status ?? currentTask.value?.status ?? '')
  }

  function stopCountdown() {
    if (countdownHandle != null) {
      clearInterval(countdownHandle)
      countdownHandle = null
    }
    pollCountdownMs.value = 0
  }

  function startCountdown(ms: number) {
    stopCountdown()
    if (ms <= 0) return
    pollDeadline = Date.now() + ms
    pollCountdownMs.value = ms
    countdownHandle = setInterval(() => {
      pollCountdownMs.value = Math.max(0, pollDeadline - Date.now())
    }, 200)
  }

  function stopPolling() {
    if (pollHandle != null) {
      clearTimeout(pollHandle)
      pollHandle = null
    }
    stopCountdown()
    isPolling.value = false
  }

  async function refreshCurrentTask() {
    const teams = useTeamsStore()
    if (!teams.currentTeamId || !currentTaskId.value) return

    const taskId = currentTaskId.value
    taskRefreshing.value = true
    stopCountdown()
    try {
      await Promise.all([
        loadMessages(),
        loadTimeline(),
        loadApprovals(),
        loadTasks(),
        useTodosStore().load(taskId),
        useWorkspaceStore().load(taskId),
      ])
      lastRefreshedAt.value = Date.now()
    } finally {
      taskRefreshing.value = false
    }
  }

  async function refreshTaskNow() {
    if (!currentTaskId.value || taskRefreshing.value) return
    if (!shouldAutoRefresh()) return
    stopPolling()
    await refreshCurrentTask()
    schedulePolling()
  }

  function resetInspector() {
    useTodosStore().reset()
    useWorkspaceStore().reset()
  }

  function schedulePolling() {
    stopPolling()
    if (!currentTaskId.value) return
    if (!shouldAutoRefresh()) return

    isPolling.value = true

    const queueNext = () => {
      if (!currentTaskId.value || !shouldAutoRefresh()) {
        stopPolling()
        return
      }
      const interval = pollIntervalMs()
      startCountdown(interval)
      pollHandle = setTimeout(() => {
        void tick()
      }, interval)
    }

    const tick = async () => {
      if (!currentTaskId.value) return
      await refreshCurrentTask()
      if (!currentTaskId.value || !shouldAutoRefresh()) {
        stopPolling()
        return
      }
      queueNext()
    }

    queueNext()
  }

  async function loadTasks() {
    const teams = useTeamsStore()
    if (!teams.currentTeamId) return
    tasks.value = asArray(
      await fetchJSON<TeamTask[] | null>(`/teams/${teams.currentTeamId}/tasks`),
    )
  }

  async function loadMessages() {
    const teams = useTeamsStore()
    if (!teams.currentTeamId || !currentTaskId.value) return
    messages.value = asArray(
      await fetchJSON<TeamMessage[] | null>(
        `/teams/${teams.currentTeamId}/tasks/${currentTaskId.value}/messages`,
      ),
    )
  }

  async function sendMessage(content: string, options?: SendMessageOptions) {
    const teams = useTeamsStore()
    if (!teams.currentTeamId) {
      throw new Error('请先选择 Team')
    }

    const body: { content: string; taskId?: string } = { content: content.trim() }
    if (!body.content) {
      throw new Error('消息不能为空')
    }

    if (!options?.forceNew && !composingNew.value) {
      const tid = options?.taskId ?? currentTaskId.value
      if (tid) {
        const task = tasks.value.find((t) => t.id === tid && t.teamId === teams.currentTeamId)
        if (task) {
          if (isTerminalTask(task.status)) {
            throw new Error('此任务已结束，无法继续发送消息')
          }
          body.taskId = tid
        }
      }
    }

    const resp = await fetchJSON<SendTeamMessageResponse>(
      `/teams/${teams.currentTeamId}/messages`,
      {
        method: 'POST',
        body: JSON.stringify(body),
      },
    )

    if (!resp?.task?.id) {
      throw new Error('服务端返回无效')
    }
    const list = asArray(tasks.value)
    const idx = list.findIndex((t) => t.id === resp.task.id)
    if (idx >= 0) {
      list[idx] = resp.task
      tasks.value = list
    } else {
      tasks.value = [resp.task, ...list]
    }

    const isNewThread =
      options?.forceNew ||
      composingNew.value ||
      !currentTaskId.value ||
      currentTaskId.value !== resp.task.id

    if (isNewThread) {
      composingNew.value = false
      composeDraft.value = ''
      composeReturnTaskId.value = ''
      selectTask(resp.task.id)
    } else {
      const msgs = asArray(messages.value)
      const exists = msgs.some((m) => m.id === resp.message.id)
      if (!exists) {
        messages.value = [...msgs, resp.message]
      }
      await refreshCurrentTask()
    }
    return resp
  }

  function clearForTeamSwitch() {
    stopPolling()
    composingNew.value = false
    composeDraft.value = ''
    composeReturnTaskId.value = ''
    currentTaskId.value = ''
    messages.value = []
    timeline.value = []
    pendingApprovals.value = []
    taskSwitching.value = false
    taskRefreshing.value = false
    isPolling.value = false
    pollCountdownMs.value = 0
    lastRefreshedAt.value = null
    resetInspector()
  }

  function startNewTask() {
    composeReturnTaskId.value = currentTaskId.value
    composeDraft.value = ''
    stopPolling()
    composingNew.value = true
    currentTaskId.value = ''
    messages.value = []
    timeline.value = []
    pendingApprovals.value = []
    taskSwitching.value = false
    resetInspector()
  }

  function cancelCompose() {
    if (!composingNew.value) return
    composingNew.value = false
    composeDraft.value = ''
    const prev = composeReturnTaskId.value
    composeReturnTaskId.value = ''
    if (prev) {
      selectTask(prev)
      return
    }
    stopPolling()
    currentTaskId.value = ''
    messages.value = []
    timeline.value = []
    pendingApprovals.value = []
    resetInspector()
  }

  /** @deprecated 使用 sendMessage */
  async function submit(content: string) {
    return sendMessage(content)
  }

  async function loadTimeline() {
    const teams = useTeamsStore()
    if (!teams.currentTeamId || !currentTaskId.value) return
    timeline.value = asArray(
      await fetchJSON<TimelineEvent[] | null>(
        `/teams/${teams.currentTeamId}/tasks/${currentTaskId.value}/timeline`,
      ),
    )
  }

  async function loadApprovals() {
    const teams = useTeamsStore()
    if (!teams.currentTeamId) return
    pendingApprovals.value = asArray(
      await fetchJSON<ApprovalRequest[] | null>(`/teams/${teams.currentTeamId}/approvals`),
    )
  }

  function setComposeDraft(text: string) {
    if (!composingNew.value) return
    composeDraft.value = text
  }

  function selectTask(id: string) {
    composingNew.value = false
    composeDraft.value = ''
    composeReturnTaskId.value = ''
    stopPolling()
    if (currentTaskId.value !== id) {
      messages.value = []
      timeline.value = []
    }
    currentTaskId.value = id
    taskSwitching.value = true
    void (async () => {
      try {
        await refreshCurrentTask()
      } finally {
        taskSwitching.value = false
      }
      schedulePolling()
    })()
  }

  async function approve(approvalId: string, approve: boolean) {
    const teams = useTeamsStore()
    const path = approve ? 'approve' : 'reject'
    await fetchJSON(`/teams/${teams.currentTeamId}/approvals/${approvalId}/${path}`, {
      method: 'POST',
      body: JSON.stringify({}),
    })
    await refreshCurrentTask()
  }

  async function cancelTask() {
    const teams = useTeamsStore()
    if (!teams.currentTeamId || !currentTaskId.value) {
      throw new Error('未选择任务')
    }
    await fetchJSON(
      `/teams/${teams.currentTeamId}/tasks/${currentTaskId.value}/cancel`,
      {
        method: 'POST',
        body: JSON.stringify({}),
      },
    )
    await refreshCurrentTask()
    stopPolling()
  }

  return {
    tasks,
    messages,
    currentTaskId,
    composingNew,
    composeDraft,
    taskSwitching,
    taskRefreshing,
    isPolling,
    pollCountdownMs,
    pollProgress,
    currentPollInterval,
    lastRefreshedAt,
    currentTaskTerminal,
    currentTask,
    timeline,
    taskFeed,
    pendingApprovals,
    loadTasks,
    loadMessages,
    sendMessage,
    submit,
    loadTimeline,
    loadApprovals,
    refreshCurrentTask,
    refreshTaskNow,
    selectTask,
    startNewTask,
    setComposeDraft,
    cancelCompose,
    approve,
    cancelTask,
    clearForTeamSwitch,
    stopPolling,
  }
})
