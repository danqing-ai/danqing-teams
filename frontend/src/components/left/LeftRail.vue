<script setup lang="ts">
import { computed, ref, watch, nextTick } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  Plus,
  Tools,
  Grid,
  MagicStick,
  Document,
  MoreFilled,
} from '@danqing/dq-shell'
import { useResizableWidth } from '@/composables/useResizableWidth'
import { useSessionsStore } from '@/stores/sessions'
import { useProjectsStore } from '@/stores/projects'
import { confirm, toast } from '@/utils/feedback'
import { formatRelativeTime } from '@/utils/time'
import type { AppModule } from '@/types/app-module'
import type { Project } from '@/types'
import type { Session } from '@/types/mission'

const emit = defineEmits<{
  navigate: [module: AppModule]
  selectSession: [id: string]
  newSession: [projectId?: string]
}>()

const props = defineProps<{
  activeModule: AppModule
}>()

const { t } = useI18n()
const COLLAPSED_KEY = 'app-left-collapsed'
const { width, onResizePointerDown } = useResizableWidth('app-left-width', 240, 180, 320)

const collapsed = ref(localStorage.getItem(COLLAPSED_KEY) === '1')
watch(collapsed, (v) => localStorage.setItem(COLLAPSED_KEY, v ? '1' : '0'))

const railStyle = computed(() => (collapsed.value ? { width: '44px' } : { width: `${width.value}px` }))

const sessions = useSessionsStore()
const projects = useProjectsStore()

const DEFAULT_VISIBLE_TASKS = 4
const expandedProjects = ref<Set<string>>(new Set())
const expandedSessionProjects = ref<Set<string>>(new Set())

function toggleProject(id: string) {
  if (expandedProjects.value.has(id)) {
    expandedProjects.value.delete(id)
  } else {
    expandedProjects.value.add(id)
  }
}

function expandProject(id: string) {
  expandedProjects.value.add(id)
}

function toggleMoreSessions(id: string) {
  if (expandedSessionProjects.value.has(id)) {
    expandedSessionProjects.value.delete(id)
  } else {
    expandedSessionProjects.value.add(id)
  }
}

function visibleSessions(p: Project): Session[] {
  const list = projectSessions(p)
  if (expandedSessionProjects.value.has(p.id)) return list
  return list.slice(0, DEFAULT_VISIBLE_TASKS)
}

function hasMoreSessions(p: Project): boolean {
  return projectSessions(p).length > DEFAULT_VISIBLE_TASKS
}

const menuItems = computed(() => [
  { module: 'workers' as const, label: t('navigation.workers'), icon: Grid },
  { module: 'knowledge' as const, label: t('navigation.knowledge'), icon: Document },
  { module: 'skills' as const, label: t('navigation.skills'), icon: MagicStick },
  { module: 'mcpServers' as const, label: t('navigation.mcpServer'), isMcp: true },
  { module: 'automations' as const, label: t('navigation.automations'), icon: Tools },
])

function navigate(module: AppModule) {
  emit('navigate', module)
}

function onNewSession(projectId?: string) {
  emit('newSession', projectId)
}

function selectSession(id: string) {
  emit('selectSession', id)
}

async function archiveSession(id: string) {
  try {
    await sessions.updateSession(id, { status: 'archived' })
    toast.success('已归档')
  } catch (e) {
    toast.error(e instanceof Error ? e.message : '归档失败')
  }
}

async function deleteSession(id: string) {
  const confirmed = await confirm('确定删除该会话？', { title: '删除会话', confirmText: '删除', type: 'danger' })
  if (!confirmed) return
  try {
    await sessions.deleteSession(id)
    toast.success('已删除')
  } catch (e) {
    toast.error(e instanceof Error ? e.message : '删除失败')
  }
}

const renamingProjectId = ref<string | null>(null)
const renamingName = ref('')

const showNewProjectForm = ref(false)
const newProjectName = ref('')
const newProjectDirectory = ref('')
const newProjectNameInput = ref<HTMLInputElement | null>(null)

function openNewProjectForm() {
  showNewProjectForm.value = true
  newProjectName.value = ''
  newProjectDirectory.value = ''
  nextTick(() => newProjectNameInput.value?.focus())
}

function cancelNewProject() {
  showNewProjectForm.value = false
}

async function createProject() {
  const name = newProjectName.value.trim()
  if (!name) {
    toast.error(t('navigation.projectNameRequired'))
    return
  }
  try {
    const dir = newProjectDirectory.value.trim() || undefined
    await projects.createProject(name, dir)
    showNewProjectForm.value = false
    toast.success(t('navigation.projectCreated'))
  } catch (e) {
    toast.error(e instanceof Error ? e.message : t('navigation.projectCreateFailed'))
  }
}

function startRenameProject(p: Project) {
  renamingProjectId.value = p.id
  renamingName.value = p.name
}

async function confirmRenameProject(id: string) {
  const name = renamingName.value.trim()
  if (!name) return
  try {
    await projects.renameProject(id, name)
    renamingProjectId.value = null
    toast.success(t('navigation.renamed'))
  } catch (e) {
    toast.error(e instanceof Error ? e.message : t('navigation.renameFailed'))
  }
}

async function removeProject(p: Project) {
  try {
    await confirm(t('navigation.deleteProjectConfirm', { name: p.name }), t('navigation.deleteProject'), { type: 'warning' })
  } catch {
    return
  }
  try {
    await projects.deleteProject(p.id)
    toast.success(t('navigation.deleted'))
  } catch (e) {
    toast.error(e instanceof Error ? e.message : t('navigation.deleteFailed'))
  }
}

function projectSessions(p: Project): Session[] {
  return sessions.sessionsByProject.get(p.id) ?? []
}

function sessionTitle(t_: Session): string {
  return (t_.title ?? t_.content).trim().slice(0, 40) || t('navigation.untitledTask')
}

const userLabel = computed(() => 'nil luo')
const userInitial = computed(() => userLabel.value.slice(0, 1).toUpperCase())
const userPlan = computed(() => 'DanQing')

const defaultDirectoryHint = computed(() => {
  const name = newProjectName.value.trim()
  if (name) {
    return `./data/${name}/`
  }
  return './data/<project-name>/'
})

watch(() => projects.projects.length, (len) => {
  if (len && !expandedProjects.value.size) {
    projects.sortedProjects.forEach((p) => expandProject(p.id))
  }
})
</script>

<template>
  <div class="module-sidebar" :class="{ 'is-collapsed': collapsed }" :style="railStyle">
    <div v-if="collapsed" class="module-sidebar__strip">
      <DqIconButton :aria-label="$t('navigation.newSession')" @click="onNewSession()">
        <DqIcon :size="18"><Plus /></DqIcon>
      </DqIconButton>
      <DqIconButton aria-label="展开侧栏" @click="collapsed = false">
        <svg viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M9 6l6 6-6 6" stroke-linecap="round" stroke-linejoin="round" />
        </svg>
      </DqIconButton>
    </div>

    <template v-else>
      <aside class="module-sidebar__panel">
        <div class="module-sidebar__top">
          <DqButton type="primary" class="module-sidebar__new-session" @click="onNewSession()">
            <DqIcon :size="16"><Plus /></DqIcon>
            {{ $t('navigation.newSession') }}
          </DqButton>

          <div class="module-sidebar__modules">
            <nav class="module-sidebar__menu" aria-label="模块导航">
              <button
                v-for="item in menuItems"
                :key="item.module"
                type="button"
                class="module-sidebar__nav"
                :class="{ 'is-active': props.activeModule === item.module }"
                @click="navigate(item.module)"
              >
                <DqIcon :size="16">
                  <component :is="item.icon" v-if="item.icon" />
                  <svg v-else viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" stroke-width="2">
                    <rect x="3" y="3" width="7" height="7" rx="1.5" />
                    <rect x="14" y="3" width="7" height="7" rx="1.5" />
                    <rect x="3" y="14" width="7" height="7" rx="1.5" />
                    <rect x="14" y="14" width="7" height="7" rx="1.5" />
                  </svg>
                </DqIcon>
                <span>{{ item.label }}</span>
              </button>
            </nav>
          </div>

          <div class="module-sidebar__divider" />

          <div class="module-sidebar__section">
            <div class="module-sidebar__section-head">
              <span class="module-sidebar__section-title">{{ $t('navigation.projects') }}</span>
              <button type="button" class="module-sidebar__section-add" :aria-label="$t('navigation.newProject')" @click="openNewProjectForm">
                <svg viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                  <path d="M12 5v14M5 12h14" />
                </svg>
              </button>
            </div>

            <DqDialog
              v-model:open="showNewProjectForm"
              :title="$t('navigation.newProject')"
              variant="glass"
              width="380px"
              :closable="true"
            >
              <div class="new-project-form">
                <label class="new-project-field">
                  <span class="new-project-field__label">{{ $t('navigation.projectName') }}</span>
                  <input
                    ref="newProjectNameInput"
                    v-model="newProjectName"
                    class="new-project-field__input"
                    type="text"
                    :placeholder="$t('navigation.projectName')"
                    @keydown.enter="createProject"
                  />
                </label>
                <label class="new-project-field">
                  <span class="new-project-field__label">
                    <svg class="new-project-field__icon" viewBox="0 0 24 24" width="12" height="12" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                      <path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z" />
                    </svg>
                    {{ $t('navigation.projectPath') }}
                  </span>
                  <input
                    v-model="newProjectDirectory"
                    class="new-project-field__input"
                    type="text"
                    :placeholder="$t('navigation.projectPathPlaceholder')"
                    spellcheck="false"
                    @keydown.enter="createProject"
                  />
                  <span v-if="!newProjectDirectory.trim()" class="new-project-field__hint">{{ defaultDirectoryHint }}</span>
                </label>
              </div>
              <template #footer>
                <div class="new-project-actions">
                  <DqButton @click="cancelNewProject">{{ $t('common.cancel') }}</DqButton>
                  <DqButton type="primary" :disabled="!newProjectName.trim()" @click="createProject">{{ $t('common.create') }}</DqButton>
                </div>
              </template>
            </DqDialog>

            <div v-if="projects.loading" class="module-sidebar__empty">{{ $t('navigation.loading_') }}</div>
            <div v-else-if="!projects.sortedProjects.length" class="module-sidebar__empty">{{ $t('navigation.noProjects') }}</div>
            <nav v-else class="project-tree" aria-label="项目列表">
              <div v-for="p in projects.sortedProjects" :key="p.id" class="project-tree__group">
                <div class="project-tree__row" :class="{ 'is-active': false }" @click="toggleProject(p.id)">
                  <span class="project-tree__toggle" :class="{ 'is-expanded': expandedProjects.has(p.id) }">
                    <svg viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                      <path d="M9 18l6-6-6-6" />
                    </svg>
                  </span>
                  <svg class="project-tree__folder-icon" viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                    <path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z" />
                  </svg>
                  <template v-if="renamingProjectId === p.id">
                    <DqInput v-model="renamingName" size="small" class="project-tree__rename-input" @keydown.enter="confirmRenameProject(p.id)" @blur="renamingProjectId = null" @click.stop />
                  </template>
                  <span v-else class="project-tree__name">{{ p.name }}</span>
                  <span @click.stop>
                    <DqDropdown class="project-tree__menu">
                      <DqIconButton aria-label="项目菜单" @click.stop>
                        <DqIcon :size="14"><MoreFilled /></DqIcon>
                      </DqIconButton>
                      <template #dropdown>
                        <DqDropdownMenu>
                          <DqDropdownItem @select="onNewSession(p.id)">{{ $t('navigation.newSession') }}</DqDropdownItem>
                          <DqDropdownItem @select="startRenameProject(p)">{{ $t('navigation.rename') }}</DqDropdownItem>
                          <DqDropdownItem @select="removeProject(p)">{{ $t('common.delete') }}</DqDropdownItem>
                        </DqDropdownMenu>
                      </template>
                    </DqDropdown>
                  </span>
                </div>

                <div v-if="expandedProjects.has(p.id)" class="project-tree__sessions">
                  <div
                    v-for="t_ in visibleSessions(p)"
                    :key="t_.id"
                    class="project-tree__session-row"
                  >
                    <button
                      type="button"
                      class="project-tree__session"
                      :class="{ 'is-active': sessions.currentSessionId === t_.id && !sessions.composingNew }"
                      @click="selectSession(t_.id)"
                    >
                      <span class="project-tree__session-dot" />
                      <span class="project-tree__session-name">{{ sessionTitle(t_) }}</span>
                      <span class="project-tree__session-time">{{ formatRelativeTime(t_.updatedAt || t_.createdAt) }}</span>
                    </button>
                    <DqDropdown>
                      <button type="button" class="project-tree__session-action" title="会话操作" @click.stop>
                        <svg viewBox="0 0 24 24" width="14" height="14" fill="currentColor">
                          <circle cx="12" cy="5" r="1.5" />
                          <circle cx="12" cy="12" r="1.5" />
                          <circle cx="12" cy="19" r="1.5" />
                        </svg>
                      </button>
                      <template #dropdown>
                        <DqDropdownMenu>
                          <DqDropdownItem @select="archiveSession(t_.id)">归档</DqDropdownItem>
                          <DqDropdownItem @select="deleteSession(t_.id)">
                            <span style="color:var(--dq-color-danger)">删除</span>
                          </DqDropdownItem>
                        </DqDropdownMenu>
                      </template>
                    </DqDropdown>
                  </div>
                  <button
                    v-if="hasMoreSessions(p)"
                    type="button"
                    class="project-tree__session project-tree__session--more"
                    @click.stop="toggleMoreSessions(p.id)"
                  >
                    <span class="project-tree__session-more-icon">
                      <svg v-if="expandedSessionProjects.has(p.id)" viewBox="0 0 24 24" width="12" height="12" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                        <path d="M18 15l-6-6-6 6" />
                      </svg>
                      <svg v-else viewBox="0 0 24 24" width="12" height="12" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                        <path d="M6 9l6 6 6-6" />
                      </svg>
                    </span>
                    <span class="project-tree__session-name">{{ expandedSessionProjects.has(p.id) ? $t('navigation.showLess') : $t('navigation.showMore') }}</span>
                  </button>
                  <button v-else-if="!projectSessions(p).length" type="button" class="project-tree__session project-tree__session--empty" @click="onNewSession(p.id)">
                    {{ $t('navigation.newSessionPrompt') }}
                  </button>
                </div>
              </div>
            </nav>
          </div>
        </div>

        <footer class="module-sidebar__footer">
          <button type="button" class="module-sidebar__user" @click="navigate('knowledge')">
            <span class="module-sidebar__avatar" aria-hidden="true">{{ userInitial }}</span>
            <span class="module-sidebar__info">
              <span class="module-sidebar__name">{{ userLabel }}</span>
              <span class="module-sidebar__plan">{{ userPlan }}</span>
            </span>
          </button>
          <DqIconButton class="module-sidebar__settings" :aria-label="$t('navigation.settings')" @click="navigate('settings')">
            <svg viewBox="0 0 24 24" width="18" height="18" fill="none" stroke="currentColor" stroke-width="2">
              <circle cx="12" cy="12" r="3" />
              <path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1 0 2.83 2 2 0 0 1-2.83 0l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-2 2 2 2 0 0 1-2-2v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83 0 2 2 0 0 1 0-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1-2-2 2 2 0 0 1 2-2h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 0-2.83 2 2 0 0 1 2.83 0l.06.06a1.65 1.65 0 0 0 1.82.33H9a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 2-2 2 2 0 0 1 2 2v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 0 2 2 0 0 1 0 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 2 2 2 2 0 0 1-2 2h-.09a1.65 1.65 0 0 0-1.51 1z" />
            </svg>
          </DqIconButton>
        </footer>
      </aside>

      <button type="button" class="module-sidebar__resize" aria-label="调整宽度" @pointerdown="onResizePointerDown" />
    </template>
  </div>
</template>

<style scoped>
.module-sidebar {
  position: relative;
  display: flex;
  flex-direction: column;
  min-width: 44px;
  max-width: 320px;
  height: 100%;
  min-height: 0;
  transition: width 0.2s ease;
  border-right: 1px solid var(--teams-glass-border);
  background: var(--teams-glass-bg);
}

.module-sidebar.is-collapsed {
  min-width: 44px;
  max-width: 44px;
}

.module-sidebar__strip {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: var(--space-sm);
  flex: 1;
  min-height: 0;
  padding: 10px 0;
  background: transparent;
}

.module-sidebar__panel {
  flex: 1;
  min-height: 0;
  height: 100%;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  background: transparent;
}

.module-sidebar__brand {
  flex-shrink: 0;
  display: flex;
  align-items: center;
  gap: 8px;
  height: 44px;
  padding: 0 12px;
  border-bottom: 1px solid color-mix(in srgb, var(--dq-label-primary) 8%, transparent);
}

.module-sidebar__brand-icon {
  color: var(--dq-accent);
  flex-shrink: 0;
}

.module-sidebar__brand-title {
  font-size: 14px;
  font-weight: 650;
  letter-spacing: -0.02em;
  color: var(--dq-label-primary);
  white-space: nowrap;
}

.module-sidebar__top {
  flex-shrink: 1;
  min-height: 0;
  overflow-y: auto;
  padding: 12px 10px 10px;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.module-sidebar__new-session {
  width: 100%;
  justify-content: center;
  gap: 6px;
}

.module-sidebar__section {
  display: flex;
  flex-direction: column;
  gap: 6px;
  margin-top: 4px;
}

.module-sidebar__section-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 4px;
}

.module-sidebar__section-title {
  font-size: 11px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.03em;
  color: var(--dq-label-tertiary);
}

.module-sidebar__section-add {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 24px;
  height: 24px;
  border: none;
  border-radius: 5px;
  background: transparent;
  color: var(--dq-label-tertiary);
  cursor: pointer;
  transition: background 0.12s ease, color 0.12s ease;
}

.module-sidebar__section-add:hover {
  background: color-mix(in srgb, var(--dq-accent) 12%, transparent);
  color: var(--dq-accent);
}

.new-project-form {
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.new-project-field {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.new-project-field__label {
  display: flex;
  align-items: center;
  gap: 5px;
  font-size: 12px;
  font-weight: 500;
  color: var(--dq-label-secondary);
}

.new-project-field__icon {
  flex-shrink: 0;
  color: var(--dq-label-tertiary);
}

.new-project-field__input {
  width: 100%;
  padding: 8px 12px;
  border: 1px solid var(--dq-border, rgba(255, 255, 255, 0.08));
  border-radius: 8px;
  background: var(--dq-bg-container, rgba(255, 255, 255, 0.04));
  color: var(--dq-label-primary);
  font-size: 13px;
  font-family: inherit;
  outline: none;
  transition: border-color 0.15s ease, box-shadow 0.15s ease;
}

.new-project-field__input:focus {
  border-color: var(--dq-accent);
  box-shadow: 0 0 0 2px color-mix(in srgb, var(--dq-accent) 18%, transparent);
}

.new-project-field__input::placeholder {
  color: var(--dq-label-tertiary);
}

.new-project-field__hint {
  font-size: 11px;
  color: var(--dq-label-tertiary);
  font-family: ui-monospace, SFMono-Regular, 'SF Mono', Menlo, monospace;
  padding-left: 2px;
}

.new-project-actions {
  display: flex;
  gap: 8px;
  justify-content: flex-end;
}

.module-sidebar__empty {
  padding: 8px 6px;
  font-size: 12px;
  color: var(--dq-label-tertiary);
}

.project-tree {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.project-tree__group {
  display: flex;
  flex-direction: column;
}

.project-tree__row {
  display: flex;
  align-items: center;
  gap: 6px;
  height: 32px;
  border-radius: 8px;
  padding: 0 8px 0 4px;
  transition: background 0.12s ease, color 0.12s ease;
  cursor: pointer;
  color: var(--dq-label-primary);
}

.project-tree__row:hover {
  background: color-mix(in srgb, var(--dq-label-primary) 5%, transparent);
}

.project-tree__row.is-active,
.project-tree__row:hover {
  color: var(--dq-accent);
}

.project-tree__row:hover .project-tree__folder-icon {
  color: var(--dq-accent);
}

.project-tree__toggle {
  flex-shrink: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  width: 20px;
  height: 28px;
  color: var(--dq-label-tertiary);
  transition: transform 0.2s ease, color 0.12s ease;
}

.project-tree__toggle.is-expanded {
  transform: rotate(90deg);
}

.project-tree__row:hover .project-tree__toggle {
  color: var(--dq-label-primary);
}

.project-tree__folder-icon {
  flex-shrink: 0;
  color: var(--dq-label-tertiary);
  transition: color 0.12s ease;
}

.project-tree__name {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 13px;
  font-weight: 500;
}

.project-tree__rename-input {
  flex: 1;
  min-width: 0;
}

.project-tree__menu {
  flex-shrink: 0;
  opacity: 0;
  transition: opacity 0.12s ease;
}

.project-tree__row:hover .project-tree__menu {
  opacity: 1;
}

.project-tree__sessions {
  display: flex;
  flex-direction: column;
  padding-left: 20px;
  margin-left: 10px;
  border-left: 1px solid color-mix(in srgb, var(--dq-label-primary) 8%, transparent);
}

.project-tree__session-row {
  display: flex;
  align-items: center;
  border-radius: 6px;
  transition: background 0.12s ease;
}

.project-tree__session-row:hover {
  background: color-mix(in srgb, var(--dq-label-primary) 5%, transparent);
}

.project-tree__session-row:hover .project-tree__session-action {
  opacity: 1;
}

.project-tree__session {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 7px 10px;
  border: none;
  border-radius: 6px;
  background: transparent;
  color: var(--dq-label-secondary);
  font-size: 12px;
  cursor: pointer;
  text-align: left;
  flex: 1;
  min-width: 0;
  transition: color 0.12s ease;
}

.project-tree__session:hover {
  background: transparent;
  color: var(--dq-label-primary);
}

.project-tree__session.is-active {
  background: color-mix(in srgb, var(--dq-accent) 10%, var(--dq-fill-tertiary));
  color: var(--dq-accent);
}

.project-tree__session-action {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 22px;
  height: 22px;
  margin-right: 4px;
  padding: 0;
  border: none;
  border-radius: 4px;
  background: transparent;
  color: var(--dq-label-tertiary);
  cursor: pointer;
  opacity: 0;
  flex-shrink: 0;
  transition: opacity 0.15s, background 0.15s, color 0.15s;
}

.project-tree__session-action:hover {
  background: color-mix(in srgb, var(--dq-label-primary) 10%, transparent);
  color: var(--dq-label-primary);
}

.project-tree__session--empty {
  color: var(--dq-label-tertiary);
  font-style: italic;
}

.project-tree__session--more {
  color: var(--dq-label-tertiary);
  font-style: normal;
}

.project-tree__session-more-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 6px;
}

.project-tree__session-dot {
  flex-shrink: 0;
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: color-mix(in srgb, var(--dq-label-tertiary) 60%, transparent);
}

.project-tree__session.is-active .project-tree__session-dot {
  background: var(--dq-accent);
}

.project-tree__session-name {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.project-tree__session-time {
  flex-shrink: 0;
  font-size: 11px;
  color: var(--dq-label-tertiary);
  padding-left: 8px;
}

.project-tree__session:hover .project-tree__session-time {
  color: var(--dq-label-secondary);
}

.project-tree__session.is-active .project-tree__session-time {
  color: var(--dq-accent);
}

.module-sidebar__modules {
  padding: 8px 0;
}

.module-sidebar__divider {
  height: 1px;
  margin: 8px 10px;
  background: color-mix(in srgb, var(--dq-label-primary) 8%, transparent);
  flex-shrink: 0;
}

.module-sidebar__menu {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.module-sidebar__nav {
  display: flex;
  align-items: center;
  gap: 10px;
  width: 100%;
  padding: 9px 10px;
  border: none;
  border-radius: 8px;
  background: transparent;
  color: var(--dq-label-primary);
  font-size: 13px;
  font-weight: 500;
  cursor: pointer;
  text-align: left;
  transition: background 0.12s ease, color 0.12s ease;
}

.module-sidebar__nav:hover {
  background: color-mix(in srgb, var(--dq-label-primary) 6%, transparent);
}

.module-sidebar__nav.is-active {
  background: color-mix(in srgb, var(--dq-accent) 12%, var(--dq-fill-tertiary));
  color: var(--dq-accent);
}

.module-sidebar__footer {
  flex-shrink: 0;
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 8px 6px 8px;
  border-top: 1px solid color-mix(in srgb, var(--dq-label-primary) 8%, transparent);
  margin-top: auto;
}

.module-sidebar__user {
  flex: 1;
  min-width: 0;
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 6px 8px;
  border: none;
  border-radius: 8px;
  background: transparent;
  cursor: pointer;
  text-align: left;
  color: inherit;
}

.module-sidebar__user:hover {
  background: color-mix(in srgb, var(--dq-label-primary) 6%, transparent);
}

.module-sidebar__avatar {
  width: 28px;
  height: 28px;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 12px;
  font-weight: 600;
  background: color-mix(in srgb, var(--dq-accent) 20%, transparent);
  color: var(--dq-accent);
  flex-shrink: 0;
}

.module-sidebar__info {
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 1px;
}

.module-sidebar__name {
  font-size: 13px;
  font-weight: 600;
  color: var(--dq-label-primary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.module-sidebar__plan {
  font-size: 11px;
  color: var(--dq-label-tertiary);
}

.module-sidebar__settings {
  flex-shrink: 0;
  color: var(--dq-label-tertiary);
}

.module-sidebar__settings:hover {
  color: var(--dq-label-primary);
}

.module-sidebar__resize {
  position: absolute;
  top: 0;
  right: -6px;
  z-index: 5;
  width: 12px;
  height: 100%;
  padding: 0;
  border: none;
  background: transparent;
  cursor: col-resize;
}

.module-sidebar__resize::after {
  content: '';
  position: absolute;
  top: 12%;
  bottom: 12%;
  left: 50%;
  width: 2px;
  transform: translateX(-50%);
  border-radius: 1px;
  background: transparent;
  transition: background 0.15s ease;
}

.module-sidebar__resize:hover::after,
body.app-is-resizing .module-sidebar__resize::after {
  background: color-mix(in srgb, var(--dq-accent) 45%, transparent);
}
</style>
