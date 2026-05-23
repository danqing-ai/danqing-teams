<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useResizableWidth } from '@/composables/useResizableWidth'
import AgentMembersDialog from '@/components/right/AgentMembersDialog.vue'
import AgentPickerDialog from '@/components/right/AgentPickerDialog.vue'
import { useTeamsStore } from '@/stores/teams'
import { useTodosStore } from '@/stores/todos'
import { useWorkspaceStore } from '@/stores/workspace'
import { useTasksStore } from '@/stores/tasks'
import { workerInitial } from '@/utils/stream-actors'

const emit = defineEmits<{
  openAgents: [agentId?: string]
}>()

const COLLAPSED_KEY = 'teams-right-collapsed'

const { width, onResizePointerDown } = useResizableWidth(
  'teams-right-width',
  292,
  240,
  420,
  'right',
)

const collapsed = ref(localStorage.getItem(COLLAPSED_KEY) === '1')
watch(collapsed, (v) => {
  localStorage.setItem(COLLAPSED_KEY, v ? '1' : '0')
})

const showMembers = ref(false)
const showPicker = ref(false)
const teams = useTeamsStore()
const todos = useTodosStore()
const workspace = useWorkspaceStore()
const tasks = useTasksStore()

const workspaceEmptyHint = computed(() => {
  if (tasks.composingNew) {
    return '发送任务目标后，Worker 产物将显示在此。'
  }
  return tasks.currentTaskId
    ? '当前任务暂无 Workspace 产物。'
    : '选择任务后查看产物。'
})

const pendingTodos = computed(() => todos.items.filter((item) => !item.done))
const doneTodos = computed(() => todos.items.filter((item) => item.done))

const todoEmptyHint = computed(() => {
  if (tasks.composingNew) {
    return '发送任务目标后，Controller 将在此规划待办。'
  }
  return tasks.currentTaskId
    ? 'Controller 将在此规划待办。'
    : '选择任务后查看 Controller 规划的待办。'
})

const showTodos = computed(() => !tasks.composingNew && todos.items.length > 0)

const agentCount = computed(() => teams.workers.length)

const agentMenu = [
  { command: 'add', label: '添加 Worker' },
  { command: 'manage', label: '管理成员', divided: true },
]

const artifactCount = computed(() => workspace.artifacts.length)

const workspaceMenu = computed(() => [
  {
    command: 'refresh',
    label: '刷新产物',
    disabled: !tasks.currentTaskId,
  },
])

const railStyle = computed(() =>
  collapsed.value ? { width: '44px' } : { width: `${width.value}px` },
)

function onAgentMenu(command: string) {
  if (command === 'add') showPicker.value = true
  if (command === 'manage') showMembers.value = true
}

function kindLabel(kind: string) {
  if (kind === 'report') return '报告'
  if (kind === 'note') return '笔记'
  if (kind === 'pin') return 'Pin'
  return kind
}

async function refreshWorkspace() {
  await workspace.load(tasks.currentTaskId || undefined)
}

function onWorkspaceMenu(command: string) {
  if (command === 'refresh') refreshWorkspace()
}
</script>

<template>
  <div
    class="teams-right-rail"
    :class="{ 'is-collapsed': collapsed }"
    :style="railStyle"
    aria-label="Task inspector"
  >
    <div v-if="collapsed" class="teams-right-rail__strip">
      <DqIconButton
        class="teams-right-rail__expand"
        aria-label="展开 Inspector 面板"
        @click="collapsed = false"
      >
        <svg viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M15 18l-6-6 6-6" stroke-linecap="round" stroke-linejoin="round" />
        </svg>
      </DqIconButton>
      <span class="teams-right-rail__strip-label" aria-hidden="true">Inspector</span>
    </div>

    <template v-else>
      <button
        type="button"
        class="teams-right-rail__resize"
        aria-label="调整右侧面板宽度"
        @pointerdown="onResizePointerDown"
      />

      <DqSurfaceCard class="teams-right-panel float-island">
        <template #header>
          <div class="teams-right-panel__head">
            <h2 class="teams-right-panel__title">Inspector</h2>
            <DqIconButton
              class="teams-right-panel__collapse"
              aria-label="收起 Inspector 面板"
              @click="collapsed = true"
            >
              <svg viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M9 18l6-6-6-6" stroke-linecap="round" stroke-linejoin="round" />
              </svg>
            </DqIconButton>
          </div>
        </template>

        <div class="teams-right-panel__scroll">
          <section class="teams-right-section">
            <div class="teams-right-section__head">
              <h3 class="teams-right-section__title">Todo</h3>
            </div>

            <template v-if="showTodos">
              <div v-if="pendingTodos.length" class="teams-todo-group">
                <h4 class="teams-todo-group__label">未完成</h4>
                <label
                  v-for="item in pendingTodos"
                  :key="item.id"
                  class="teams-todo-item"
                >
                  <DqCheckbox
                    :model-value="item.done"
                    @update:model-value="(v: boolean) => todos.toggle(item.id, v)"
                  />
                  <span>{{ item.title }}</span>
                </label>
              </div>

              <div v-if="doneTodos.length" class="teams-todo-group">
                <h4 class="teams-todo-group__label">已完成</h4>
                <label
                  v-for="item in doneTodos"
                  :key="item.id"
                  class="teams-todo-item is-done"
                >
                  <DqCheckbox
                    :model-value="item.done"
                    @update:model-value="(v: boolean) => todos.toggle(item.id, v)"
                  />
                  <span>{{ item.title }}</span>
                </label>
              </div>
            </template>

            <div v-else class="teams-todo-empty">
              <DqEmpty :description="todoEmptyHint" />
            </div>
          </section>

          <section class="teams-right-section">
            <div class="teams-right-section__head">
              <h3 class="teams-right-section__title">
                Agents
                <span v-if="agentCount" class="teams-right-section__count">{{ agentCount }}</span>
              </h3>
              <DqDropdown trigger="click" @command="onAgentMenu">
                <button
                  type="button"
                  class="teams-right-section__more"
                  aria-label="Agents 更多操作"
                  @click.stop
                >
                  ···
                </button>
                <template #dropdown>
                  <DqDropdownMenu>
                    <DqDropdownItem
                      v-for="item in agentMenu"
                      :key="item.command"
                      :command="item.command"
                      :divided="item.divided"
                    >
                      {{ item.label }}
                    </DqDropdownItem>
                  </DqDropdownMenu>
                </template>
              </DqDropdown>
            </div>

            <div
              v-for="w in teams.workers"
              :key="w.id"
              class="agent-member-card agent-member-card--static"
            >
              <span class="agent-member-card__avatar" aria-hidden="true">
                {{ workerInitial(w.name) }}
              </span>
              <span class="agent-member-card__body">
                <span class="agent-member-card__name">{{ w.name }}</span>
                <span class="agent-member-card__persona">{{ w.persona }}</span>
              </span>
            </div>

            <DqEmpty v-if="!teams.workers.length" description="暂无 Worker 成员">
              <p class="agent-member-card__empty-hint">点击 ··· 从 Agents 库添加成员。</p>
            </DqEmpty>
          </section>

          <section class="teams-right-section">
            <div class="teams-right-section__head">
              <h3 class="teams-right-section__title">
                Workspace
                <span v-if="artifactCount" class="teams-right-section__count">{{ artifactCount }}</span>
              </h3>
              <DqDropdown trigger="click" @command="onWorkspaceMenu">
                <button
                  type="button"
                  class="teams-right-section__more"
                  aria-label="Workspace 更多操作"
                  @click.stop
                >
                  ···
                </button>
                <template #dropdown>
                  <DqDropdownMenu>
                    <DqDropdownItem
                      v-for="item in workspaceMenu"
                      :key="item.command"
                      :command="item.command"
                      :disabled="item.disabled"
                    >
                      {{ item.label }}
                    </DqDropdownItem>
                  </DqDropdownMenu>
                </template>
              </DqDropdown>
            </div>
            <DqEmpty
              v-if="tasks.composingNew || !workspace.artifacts.length"
              :description="workspaceEmptyHint"
            />
            <DqSurfaceCard
              v-for="a in workspace.artifacts"
              v-show="!tasks.composingNew"
              :key="a.id"
              class="workspace-card"
            >
              <div class="workspace-card__head">
                <p class="workspace-card__title">{{ a.title }}</p>
                <DqTag size="small" type="info">{{ kindLabel(a.kind) }}</DqTag>
              </div>
              <p v-if="a.content" class="workspace-card__content">{{ a.content }}</p>
            </DqSurfaceCard>
          </section>
        </div>
      </DqSurfaceCard>
    </template>

    <AgentMembersDialog
      v-model:open="showMembers"
      @open-agents="emit('openAgents', $event)"
      @add="showPicker = true"
    />
    <AgentPickerDialog v-model:open="showPicker" />
  </div>
</template>
