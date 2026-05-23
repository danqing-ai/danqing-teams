<script setup lang="ts">
import { computed, nextTick, ref, watch } from 'vue'
import { useTasksStore } from '@/stores/tasks'
import { useTeamsStore } from '@/stores/teams'
import { useTaskActions } from '@/composables/useTaskActions'
import { composerClosedHint, isComposerLocked, terminalStatusDetail } from '@/utils/task-list'
import { toast } from '@/utils/feedback'

const content = ref('')
const loading = ref(false)
const inputWrap = ref<HTMLElement | null>(null)
const tasks = useTasksStore()
const teams = useTeamsStore()
const { openCreateTask } = useTaskActions()

const composerLocked = computed(
  () => !tasks.composingNew && isComposerLocked(tasks.currentTask),
)

const closedHint = computed(() => {
  if (!tasks.composingNew && tasks.currentTaskTerminal && terminalStatusDetail(tasks.currentTask)) {
    return null
  }
  return composerClosedHint(tasks.currentTask, tasks.timeline, tasks.messages)
})

const showTerminalActionOnly = computed(
  () => composerLocked.value && !closedHint.value,
)

const placeholder = computed(() => {
  if (tasks.composingNew) {
    return '输入任务目标，例如：集群 ses-01 节点负载高，请分析根因…'
  }
  if (tasks.currentTaskId) {
    return '在当前任务中继续输入，Team Controller 将分派或跟进 Worker…'
  }
  return '输入任务目标，发送后 Team Controller 将创建任务并分派 Worker…'
})

function focusInput() {
  void nextTick(() => {
    const el = inputWrap.value?.querySelector('textarea')
    el?.focus()
  })
}

watch(
  () => tasks.composingNew,
  (v, prev) => {
    if (v) {
      content.value = ''
      tasks.setComposeDraft('')
      focusInput()
      return
    }
    if (prev) {
      content.value = ''
      tasks.setComposeDraft('')
    }
  },
)

watch(content, (v) => {
  if (tasks.composingNew) tasks.setComposeDraft(v)
})

async function send() {
  if (composerLocked.value || !content.value.trim() || loading.value) return
  if (!teams.currentTeamId) {
    toast.warning('请先在左侧选择 Team')
    return
  }
  loading.value = true
  const creating = tasks.composingNew
  try {
    await tasks.sendMessage(content.value.trim())
    content.value = ''
    if (creating) {
      toast.success('任务已创建')
    } else {
      focusInput()
    }
  } catch (e) {
    toast.error(e instanceof Error ? e.message : '发送失败')
  } finally {
    loading.value = false
  }
}

function onKeydown(e: KeyboardEvent) {
  if (composerLocked.value) return
  const mod = e.metaKey || e.ctrlKey
  if (mod && e.key === 'Enter') {
    e.preventDefault()
    void send()
    return
  }
  if (e.key === 'Enter' && !e.shiftKey) {
    e.preventDefault()
    void send()
  }
}

function onNewTask() {
  openCreateTask()
}

defineExpose({ focusInput })
</script>

<template>
  <div
    class="composer-float"
    :class="{
      'is-locked': composerLocked,
      'is-compose': tasks.composingNew,
      'is-sending': loading,
    }"
    role="form"
    aria-label="Team task composer"
  >
    <div v-if="composerLocked" class="composer-float__status" :class="{ 'is-minimal': showTerminalActionOnly }" role="status">
      <span v-if="closedHint" class="composer-float__status-text">{{ closedHint }}</span>
      <span v-if="closedHint" class="composer-float__status-sep" aria-hidden="true">·</span>
      <button type="button" class="composer-float__status-action" @click="onNewTask">
        新建任务
      </button>
    </div>

    <template v-else>
      <div ref="inputWrap" class="composer-float__row">
        <DqInput
          v-model="content"
          type="textarea"
          :rows="tasks.composingNew ? 3 : 2"
          class="composer-float__input"
          :placeholder="placeholder"
          @keydown="onKeydown"
        />
        <DqIconButton
          type="primary"
          class="composer-float__send"
          :disabled="loading || !content.trim()"
          :label="tasks.composingNew ? '创建任务' : 'Send'"
          @click="send"
        >
          <DqIcon :size="18" aria-hidden="true">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round">
              <path d="M12 19V5M5 12l7-7 7 7" />
            </svg>
          </DqIcon>
        </DqIconButton>
      </div>
      <footer class="composer-float__footer">
        <span class="composer-float__hint">
          {{
            loading
              ? '发送中…'
              : tasks.composingNew
                ? '首条消息即任务目标，发送后开始执行。'
                : '消息将发给 Team Controller，由其分派 Worker。'
          }}
        </span>
        <span class="composer-float__keys">
          Enter / ⌘↩ 发送 · Shift+Enter 换行 · ⌘N 新建<span v-if="tasks.composingNew"> · Esc 取消</span>
        </span>
      </footer>
    </template>
  </div>
</template>

<style scoped>
.composer-float__row {
  display: flex;
  align-items: flex-end;
  gap: 10px;
}

.composer-float__send {
  flex-shrink: 0;
  margin-bottom: 2px;
}

.composer-float__row :deep(textarea.dq-input) {
  flex: 1;
  min-height: 44px;
  resize: none;
  border: none;
  background: transparent;
  box-shadow: none;
  padding: 4px 8px 4px 4px;
  font-size: 15px;
  line-height: 1.45;
  color: var(--dq-label-primary);
}

.composer-float__row :deep(textarea.dq-input::placeholder) {
  color: var(--dq-label-tertiary);
}

.composer-float__row :deep(textarea.dq-input:focus) {
  outline: none;
  box-shadow: none;
}
</style>
