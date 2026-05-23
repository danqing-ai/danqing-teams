<script setup lang="ts">
import { computed, ref } from 'vue'
import { Check } from '@danqing/dq-shell'
import { useTeamsStore } from '@/stores/teams'
import { useTasksStore } from '@/stores/tasks'
import { useTodosStore } from '@/stores/todos'
import { useWorkspaceStore } from '@/stores/workspace'
import { confirm, toast } from '@/utils/feedback'

const teams = useTeamsStore()
const tasks = useTasksStore()
const todos = useTodosStore()
const workspace = useWorkspaceStore()

const showCreate = ref(false)
const showEdit = ref(false)
const formName = ref('')
const formDescription = ref('')
const saving = ref(false)

const currentTeam = computed(() =>
  teams.teams.find((t) => t.id === teams.currentTeamId),
)

const teamMenu = computed(() => [
  { command: 'create', label: '新建 Team' },
  {
    command: 'edit',
    label: '编辑 Team',
    disabled: !teams.currentTeamId,
  },
  {
    command: 'delete',
    label: '删除 Team',
    disabled: !teams.currentTeamId || teams.teams.length <= 1,
  },
])

async function switchTeam(id: string) {
  if (id === teams.currentTeamId) return
  await teams.selectTeam(id)
  tasks.clearForTeamSwitch()
  await tasks.loadTasks()
  if (tasks.tasks.length) {
    tasks.selectTask(tasks.tasks[0].id)
  } else {
    await todos.load()
    await workspace.load()
  }
}

function onTeamMenuCommand(cmd: string) {
  if (cmd.startsWith('select:')) {
    switchTeam(cmd.slice('select:'.length))
  }
}

function onMoreMenu(command: string) {
  if (command === 'create') openCreate()
  if (command === 'edit') openEdit()
  if (command === 'delete') removeTeam()
}

function openCreate() {
  formName.value = ''
  formDescription.value = ''
  showCreate.value = true
}

function openEdit() {
  if (!currentTeam.value) return
  formName.value = currentTeam.value.name
  formDescription.value = currentTeam.value.description ?? ''
  showEdit.value = true
}

async function saveCreate() {
  const name = formName.value.trim()
  if (!name) {
    toast.warning('请输入 Team 名称')
    return
  }
  saving.value = true
  try {
    await teams.createTeam(name, formDescription.value.trim() || undefined)
    tasks.clearForTeamSwitch()
    await tasks.loadTasks()
    if (tasks.tasks.length) {
      tasks.selectTask(tasks.tasks[0].id)
    } else {
      await todos.load()
      await workspace.load()
    }
    showCreate.value = false
    toast.success('Team 已创建')
  } finally {
    saving.value = false
  }
}

async function saveEdit() {
  if (!currentTeam.value) return
  const name = formName.value.trim()
  if (!name) {
    toast.warning('请输入 Team 名称')
    return
  }
  saving.value = true
  try {
    await teams.updateTeam(currentTeam.value.id, {
      name,
      description: formDescription.value.trim() || undefined,
    })
    showEdit.value = false
    toast.success('Team 已更新')
  } finally {
    saving.value = false
  }
}

async function removeTeam() {
  if (!currentTeam.value) return
  if (teams.teams.length <= 1) {
    toast.warning('至少保留一个 Team')
    return
  }
  try {
    await confirm(
      `确定删除「${currentTeam.value.name}」？相关任务与配置将不可恢复。`,
      '删除 Team',
      { type: 'warning' },
    )
  } catch {
    return
  }
  const id = currentTeam.value.id
  await teams.deleteTeam(id)
  tasks.clearForTeamSwitch()
  await tasks.loadTasks()
  if (tasks.tasks.length) {
    tasks.selectTask(tasks.tasks[0].id)
  } else {
    await todos.load()
    await workspace.load()
  }
  toast.success('Team 已删除')
}
</script>

<template>
  <div class="team-selector">
    <div class="team-selector__row">
      <DqDropdown class="team-selector__picker" trigger="click" @command="onTeamMenuCommand">
        <button type="button" class="team-selector__trigger" :disabled="!teams.teams.length">
          <span class="team-selector__name">
            {{ currentTeam?.name ?? '选择 Team' }}
          </span>
          <DqIcon class="team-selector__chevron" :size="16" aria-hidden="true">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round">
              <path d="M6 9l6 6 6-6" />
            </svg>
          </DqIcon>
        </button>
        <template #dropdown>
          <DqDropdownMenu class="team-selector__menu">
            <DqDropdownItem
              v-for="t in teams.teams"
              :key="t.id"
              :command="`select:${t.id}`"
            >
              <span class="team-selector__option">
                <DqIcon
                  v-if="teams.currentTeamId === t.id"
                  class="team-selector__check"
                  :size="14"
                >
                  <Check />
                </DqIcon>
                <span v-else class="team-selector__check-placeholder" aria-hidden="true" />
                <span class="team-selector__option-label">{{ t.name }}</span>
              </span>
            </DqDropdownItem>
          </DqDropdownMenu>
        </template>
      </DqDropdown>

      <DqDropdown trigger="click" @command="onMoreMenu">
        <button
          type="button"
          class="team-selector__more"
          aria-label="Team 更多操作"
          @click.stop
        >
          ···
        </button>
        <template #dropdown>
          <DqDropdownMenu>
            <DqDropdownItem
              v-for="item in teamMenu"
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

    <DqDialog v-model:open="showCreate" title="新建 Team" width="min(400px, 92vw)">
      <DqInput v-model="formName" placeholder="名称" style="margin-bottom: 8px" />
      <DqInput
        v-model="formDescription"
        type="textarea"
        :rows="2"
        placeholder="描述（可选）"
      />
      <template #footer>
        <DqButton @click="showCreate = false">取消</DqButton>
        <DqButton type="primary" :disabled="saving" @click="saveCreate">创建</DqButton>
      </template>
    </DqDialog>

    <DqDialog v-model:open="showEdit" title="编辑 Team" width="min(400px, 92vw)">
      <DqInput v-model="formName" placeholder="名称" style="margin-bottom: 8px" />
      <DqInput
        v-model="formDescription"
        type="textarea"
        :rows="2"
        placeholder="描述（可选）"
      />
      <template #footer>
        <DqButton @click="showEdit = false">取消</DqButton>
        <DqButton type="primary" :disabled="saving" @click="saveEdit">保存</DqButton>
      </template>
    </DqDialog>
  </div>
</template>
