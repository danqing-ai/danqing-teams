<script setup lang="ts">
import { computed, nextTick, ref, watch } from 'vue'
import { useTasksStore } from '@/stores/tasks'
import { useTaskActions } from '@/composables/useTaskActions'
import {
  formatTaskTime,
  groupTasks,
  isActiveTask,
  statusLabel,
  tagTypeForStatus,
  taskTitle,
  type TaskFilter,
} from '@/utils/task-list'

const tasks = useTasksStore()
const { focusComposer } = useTaskActions()

const filter = ref<TaskFilter>('all')
const listScroll = ref<HTMLElement | null>(null)

const filters: { id: TaskFilter; label: string }[] = [
  { id: 'all', label: '全部' },
  { id: 'active', label: '进行中' },
  { id: 'approval', label: '待审批' },
  { id: 'done', label: '已完成' },
]

const groups = computed(() => groupTasks(tasks.tasks, filter.value))

const isFilteredEmpty = computed(
  () => tasks.tasks.length > 0 && groups.value.length === 0,
)

const draftPreview = computed(() => {
  const draft = tasks.composeDraft.trim()
  return draft ? taskTitle(draft, 32) : '待输入目标'
})

watch(
  () => [tasks.currentTaskId, tasks.composingNew] as const,
  () => {
    void nextTick(() => {
      listScroll.value
        ?.querySelector('.task-list__item.is-active')
        ?.scrollIntoView({ block: 'nearest', behavior: 'smooth' })
    })
  },
)

function select(id: string) {
  tasks.selectTask(id)
}

function openCreate() {
  tasks.startNewTask()
}

function focusDraftComposer() {
  focusComposer()
}

defineExpose({ openCreate })
</script>

<template>
  <div class="task-list">
    <div class="task-list__toolbar">
      <div class="task-list__filters" role="tablist" aria-label="任务筛选">
        <button
          v-for="f in filters"
          :key="f.id"
          type="button"
          class="task-list__filter"
          :class="{ 'is-active': filter === f.id }"
          role="tab"
          :aria-selected="filter === f.id"
          @click="filter = f.id"
        >
          {{ f.label }}
        </button>
      </div>
      <DqButton size="sm" type="primary" class="task-list__new" @click="openCreate">
        新建
      </DqButton>
    </div>

    <nav ref="listScroll" class="task-list__scroll" aria-label="任务列表">
      <button
        v-if="tasks.composingNew"
        type="button"
        class="teams-nav__item task-list__item task-list__item--draft is-active"
        @click="focusDraftComposer"
      >
        <span class="task-list__main">
          <span class="task-list__title">新建任务</span>
          <span class="task-list__meta">{{ draftPreview }}</span>
        </span>
        <DqTag size="small" type="info">草稿</DqTag>
      </button>

      <DqEmpty
        v-if="!tasks.tasks.length && !tasks.composingNew"
        class="task-list__empty"
        description="暂无任务"
      >
        <p class="task-list__empty-hint">点击「新建」，在 Composer 输入任务目标。</p>
      </DqEmpty>

      <p v-else-if="isFilteredEmpty && !tasks.composingNew" class="task-list__filtered-empty">
        当前筛选下没有任务
      </p>

      <template v-for="group in groups" :key="group.id">
        <section class="task-list__group">
          <h3 v-if="group.label" class="task-list__group-label">{{ group.label }}</h3>
          <div class="teams-nav task-list__nav" :aria-label="group.label || '任务'">
            <button
              v-for="t in group.tasks"
              :key="t.id"
              type="button"
              class="teams-nav__item task-list__item"
              :class="{
                'is-active': tasks.currentTaskId === t.id && !tasks.composingNew,
                'is-live': isActiveTask(t.status),
              }"
              @click="select(t.id)"
            >
              <span v-if="isActiveTask(t.status)" class="task-list__pulse" aria-hidden="true" />
              <span class="task-list__main">
                <span class="task-list__title">{{ taskTitle(t.content) }}</span>
                <span v-if="formatTaskTime(t.createdAt)" class="task-list__meta">
                  {{ formatTaskTime(t.createdAt) }}
                </span>
              </span>
              <DqTag size="small" :type="tagTypeForStatus(t.status)">
                {{ statusLabel(t.status) }}
              </DqTag>
            </button>
          </div>
        </section>
      </template>
    </nav>
  </div>
</template>
