<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import MdEditor from '@/components/common/MdEditor.vue'
import { useGlobalAgentsStore } from '@/stores/globalAgents'
import { useSkillsStore } from '@/stores/skills'
import { useKnowledgeStore } from '@/stores/knowledge'
import { confirm, toast } from '@/utils/feedback'
import type { Agent, ToolBinding } from '@/types'

type ConfigTab = 'overview' | 'prompt' | 'skills' | 'tools' | 'knowledge'

type AgentForm = Agent

const { t } = useI18n()
const globalAgents = useGlobalAgentsStore()
const skills = useSkillsStore()
const knowledge = useKnowledgeStore()

const selectedId = ref<string | null>(null)
const isCreating = ref(false)
const saving = ref(false)
const activeTab = ref<ConfigTab>('overview')
const pendingTool = ref<ToolBinding>({ toolId: '', riskLevel: 'low' })

function emptyAgentForm(): AgentForm {
  return {
    id: '',
    name: '',
    description: '',
    persona: '',
    mode: 'primary',
    systemPrompt: '',
    skillIds: [],
    tools: [],
    knowledgeIds: [],
    steps: 10,
    canDelegate: false,
  }
}

const agentForm = ref<AgentForm>(emptyAgentForm())

const sortedAgents = computed(() =>
  [...globalAgents.items].sort((a, b) => a.name.localeCompare(b.name, 'zh-CN')),
)

const primaryAgents = computed(() =>
  sortedAgents.value.filter((a) => a.mode !== 'subagent'),
)

const subAgents = computed(() =>
  sortedAgents.value.filter((a) => a.mode === 'subagent'),
)

const selectedAgent = computed(() => globalAgents.items.find((a) => a.id === selectedId.value))
const hasSelection = computed(() => isCreating.value || !!selectedId.value)

const headerTitle = computed(() => {
  if (isCreating.value) return agentForm.value.name.trim() || t('teams.newAgent')
  return selectedAgent.value?.name.trim() || t('teams.untitled')
})

const sectionTabs = computed(() => [
  { id: 'overview' as const, label: t('teams.overview') },
  { id: 'prompt' as const, label: t('teams.prompt') },
  { id: 'skills' as const, label: t('teams.skills') },
  { id: 'tools' as const, label: t('teams.tools') },
  { id: 'knowledge' as const, label: t('teams.knowledge') },
])

onMounted(async () => {
  await Promise.all([globalAgents.load(), skills.load()])
  if (sortedAgents.value.length && !selectedId.value) {
    selectAgent(sortedAgents.value[0].id)
  }
})

function selectAgent(id: string) {
  isCreating.value = false
  selectedId.value = id
  activeTab.value = 'overview'
  const agent = globalAgents.items.find((a) => a.id === id)
  if (agent) {
    agentForm.value = {
      ...agent,
      skillIds: agent.skillIds ? [...agent.skillIds] : [],
      tools: agent.tools ? [...agent.tools] : [],
      knowledgeIds: agent.knowledgeIds ? [...agent.knowledgeIds] : [],
    }
  }
}

function openCreate() {
  isCreating.value = true
  selectedId.value = null
  activeTab.value = 'overview'
  agentForm.value = emptyAgentForm()
}

const ID_RE = /^[a-zA-Z][a-zA-Z0-9_-]{1,63}$/

async function save() {
  if (!agentForm.value.id || !ID_RE.test(agentForm.value.id)) {
    toast.warning(t('teams.idRule'))
    activeTab.value = 'overview'
    return
  }
  if (!agentForm.value.name) {
    toast.warning(t('teams.namePlaceholder'))
    activeTab.value = 'overview'
    return
  }

  saving.value = true
  try {
    if (isCreating.value) {
      await globalAgents.create({
        id: agentForm.value.id,
        name: agentForm.value.name,
        description: agentForm.value.description,
        persona: agentForm.value.persona,
        mode: agentForm.value.mode ?? 'primary',
        systemPrompt: agentForm.value.systemPrompt,
        steps: agentForm.value.steps ?? 10,
        skillIds: agentForm.value.skillIds,
        tools: agentForm.value.tools,
        knowledgeIds: agentForm.value.knowledgeIds,
        canDelegate: agentForm.value.canDelegate ?? false,
      })
      toast.success(t('teams.created'))
      isCreating.value = false
      selectAgent(agentForm.value.id)
    } else if (selectedAgent.value) {
      await globalAgents.update(selectedAgent.value.id, {
        name: agentForm.value.name,
        description: agentForm.value.description,
        persona: agentForm.value.persona,
        mode: agentForm.value.mode ?? 'primary',
        systemPrompt: agentForm.value.systemPrompt,
        steps: agentForm.value.steps ?? 10,
        skillIds: agentForm.value.skillIds,
        tools: agentForm.value.tools,
        knowledgeIds: agentForm.value.knowledgeIds,
        canDelegate: agentForm.value.canDelegate ?? false,
      })

      toast.success(t('teams.saved'))
      selectAgent(selectedAgent.value.id)
    }
  } catch (e) {
    toast.error(e instanceof Error ? e.message : t('common.saveFailed'))
  } finally {
    saving.value = false
  }
}

async function removeSelected() {
  if (!selectedAgent.value) return
  try {
    await confirm(t('teams.deleteConfirm', { name: selectedAgent.value.name }), t('teams.deleteAgent'), { type: 'warning' })
    await globalAgents.remove(selectedAgent.value.id)
    selectedId.value = null
    isCreating.value = false
    toast.success(t('teams.deleted'))
  } catch (e) {
    if (e instanceof Error) toast.error(e.message)
  }
}

async function resetSelected() {
  if (!selectedAgent.value) return
  try {
    await confirm(t('teams.resetConfirm', { name: selectedAgent.value.name }), t('teams.resetAgent'), { type: 'warning' })
  } catch {
    return
  }
  try {
    const a = await globalAgents.reset(selectedAgent.value.id)
    selectAgent(a.id)
    toast.success(t('teams.reset'))
  } catch (e) {
    toast.error(e instanceof Error ? e.message : t('teams.resetFailed'))
  }
}

function addTool() {
  if (!pendingTool.value.toolId.trim()) return
  agentForm.value.tools = [...(agentForm.value.tools ?? []), { ...pendingTool.value }]
  pendingTool.value = { toolId: '', riskLevel: 'low' }
}

function removeTool(idx: number) {
  agentForm.value.tools = (agentForm.value.tools ?? []).filter((_, i) => i !== idx)
}

function toggleSkill(id: string) {
  const ids = agentForm.value.skillIds ?? []
  agentForm.value.skillIds = ids.includes(id) ? ids.filter((x) => x !== id) : [...ids, id]
}

function toggleKnowledge(id: string) {
  const ids = agentForm.value.knowledgeIds ?? []
  agentForm.value.knowledgeIds = ids.includes(id) ? ids.filter((x) => x !== id) : [...ids, id]
}

function agentInitial(name: string) {
  return name.trim().charAt(0).toUpperCase() || '?'
}

function compactId(id: string) {
  if (id.length <= 20) return id
  return `${id.slice(0, 8)}…${id.slice(-4)}`
}

function onWorkspaceKeydown(e: KeyboardEvent) {
  if ((e.metaKey || e.ctrlKey) && e.key === 's') {
    e.preventDefault()
    void save()
  }
}
</script>

<template>
  <div class="resource-shell float-island" @keydown="onWorkspaceKeydown">
    <aside class="resource-rail">
      <div class="resource-rail__section">
        <div class="resource-rail__section-head">
          <span class="resource-rail__section-title">{{ $t('teams.workerAgent') }}</span>
          <DqIconButton :aria-label="$t('teams.newAgent')" @click="openCreate">
            <svg viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M12 5v14M5 12h14" stroke-linecap="round" />
            </svg>
          </DqIconButton>
        </div>
        <DqEmpty v-if="!sortedAgents.length" class="resource-rail__empty" :description="$t('teams.noWorkers')" />
        <template v-else>
          <div v-if="primaryAgents.length" class="resource-rail__group">
            <div class="resource-rail__group-title">{{ $t('teams.primaryAgents') }}</div>
            <nav class="resource-rail__list" :aria-label="$t('teams.primaryList')">
              <button
                v-for="agent in primaryAgents"
                :key="agent.id"
                type="button"
                class="resource-rail__row"
                :class="{ 'is-active': selectedId === agent.id && !isCreating }"
                @click="selectAgent(agent.id)"
              >
                <span class="resource-rail__avatar">{{ agentInitial(agent.name) }}</span>
                <span class="resource-rail__meta">
                  <span class="resource-rail__name">{{ agent.name }}</span>
                  <span class="resource-rail__desc">{{ compactId(agent.id) }}</span>
                </span>
              </button>
            </nav>
          </div>
          <div v-if="subAgents.length" class="resource-rail__group">
            <div class="resource-rail__group-title">{{ $t('teams.subAgents') }}</div>
            <nav class="resource-rail__list" :aria-label="$t('teams.subList')">
              <button
                v-for="agent in subAgents"
                :key="agent.id"
                type="button"
                class="resource-rail__row"
                :class="{ 'is-active': selectedId === agent.id && !isCreating }"
                @click="selectAgent(agent.id)"
              >
                <span class="resource-rail__avatar">{{ agentInitial(agent.name) }}</span>
                <span class="resource-rail__meta">
                  <span class="resource-rail__name">{{ agent.name }}</span>
                  <span class="resource-rail__desc">{{ compactId(agent.id) }}</span>
                </span>
              </button>
            </nav>
          </div>
        </template>
      </div>
    </aside>

    <main class="resource-workspace">
      <div v-if="!hasSelection" class="resource-workspace__empty">
        <DqEmpty :description="$t('teams.emptySelection')">
          <p class="resource-workspace__hint">{{ $t('teams.emptySelectionHint') }}</p>
        </DqEmpty>
      </div>

      <template v-else>
        <header class="resource-workspace__bar">
          <div class="resource-workspace__identity">
            <h1 class="resource-workspace__title">{{ headerTitle }}</h1>
            <div v-if="!isCreating" class="resource-workspace__badges">
              <code v-if="selectedAgent?.id" class="resource-workspace__id">
                {{ compactId(selectedAgent.id) }}
              </code>
            </div>
          </div>
          <nav class="resource-workspace__tabs" role="tablist">
            <button
              v-for="tab in sectionTabs"
              :key="tab.id"
              type="button"
              class="resource-workspace__tab"
              :class="{ 'is-active': activeTab === tab.id }"
              role="tab"
              :aria-selected="activeTab === tab.id"
              @click="activeTab = tab.id"
            >
              {{ tab.label }}
            </button>
          </nav>
        </header>

        <div class="resource-workspace__scroll">
          <section v-show="activeTab === 'overview'" class="resource-section">
            <div class="resource-form-grid resource-form-grid--2">
              <label class="resource-field">
                <span class="resource-field__label">{{ $t('teams.agentId') }}</span>
                <DqInput v-model="agentForm.id" class="resource-input-mono" placeholder="alert-analyst" :disabled="!isCreating" />
                <span v-if="isCreating" class="resource-field__hint">{{ $t('teams.idHint') }}</span>
              </label>
              <label class="resource-field">
                <span class="resource-field__label">{{ $t('teams.displayName') }}</span>
                <DqInput v-model="agentForm.name" placeholder="Alert Analyst" />
              </label>
            </div>
            <div class="resource-form-grid resource-form-grid--2">
              <label class="resource-field">
                <span class="resource-field__label">{{ $t('teams.mode') }}</span>
                <select v-model="agentForm.mode" class="resource-field__select">
                  <option value="primary">{{ $t('teams.primary') }}</option>
                  <option value="subagent">{{ $t('teams.subagent') }}</option>
                </select>
              </label>
              <div class="resource-field">
                <span class="resource-field__label">{{ $t('teams.maxSteps') }}</span>
                <div class="slider-row">
                  <DqSlider v-model="agentForm.steps" :min="1" :max="100" :step="1" />
                  <span class="slider-row__value">{{ agentForm.steps }}</span>
                </div>
              </div>
            </div>
            <label class="resource-field resource-field--block">
              <span class="resource-field__label">{{ $t('teams.persona') }}</span>
              <DqInput v-model="agentForm.persona" :placeholder="$t('teams.personaPlaceholder')" />
            </label>
            <label class="resource-field resource-field--block">
              <span class="resource-field__label">{{ $t('common.description') }}</span>
              <DqInput v-model="agentForm.description" type="textarea" :rows="4" :placeholder="$t('teams.descriptionPlaceholder')" />
            </label>
            <div class="resource-field resource-field--block resource-field--inline" @click="agentForm.canDelegate = !agentForm.canDelegate">
              <DqSwitch :model-value="agentForm.canDelegate" size="small" />
              <div class="resource-field__inline-meta">
                <span class="resource-field__label">{{ $t('teams.canDelegate') }}</span>
              </div>
            </div>
          </section>

          <section v-show="activeTab === 'prompt'" class="resource-section resource-section--prompt">
            <MdEditor v-model="agentForm.systemPrompt" :rows="18" :placeholder="$t('teams.promptPlaceholder')" />
          </section>

          <section v-show="activeTab === 'skills'" class="resource-section">
            <DqEmpty v-if="!skills.items.length" :description="$t('teams.noSkills')" />
            <div v-else class="resource-list-card">
              <div
                v-for="skill in skills.items"
                :key="skill.id"
                class="resource-list-card__item"
                :class="{ 'is-active': agentForm.skillIds?.includes(skill.id) }"
                @click="toggleSkill(skill.id)"
              >
                <DqCheckbox :model-value="agentForm.skillIds?.includes(skill.id)" />
                <div class="resource-list-card__meta">
                  <span class="resource-list-card__name">{{ skill.name }}</span>
                  <span class="resource-list-card__desc">{{ skill.description }}</span>
                </div>
              </div>
            </div>
          </section>

          <section v-show="activeTab === 'tools'" class="resource-section">
            <div class="resource-form-grid resource-form-grid--3">
              <label class="resource-field">
                <span class="resource-field__label">Tool ID</span>
                <DqInput v-model="pendingTool.toolId" class="resource-input-mono" placeholder="search_kb" @keydown.enter.prevent="addTool" />
              </label>
              <label class="resource-field">
                <span class="resource-field__label">MCP Server</span>
                <DqInput v-model="pendingTool.mcpServer" class="resource-input-mono" :placeholder="$t('teams.riskOptional')" />
              </label>
              <label class="resource-field">
                <span class="resource-field__label">{{ $t('common.riskLevel') }}</span>
                <select v-model="pendingTool.riskLevel" class="resource-field__select">
                  <option value="low">Low</option>
                  <option value="medium">Medium</option>
                  <option value="high">High</option>
                </select>
              </label>
              <div class="resource-field resource-field--action">
                <DqButton @click="addTool">{{ $t('common.addTool') }}</DqButton>
              </div>
            </div>
            <div class="resource-list-card">
              <div v-for="(tool, idx) in agentForm.tools" :key="idx" class="resource-list-card__item">
                <div class="resource-list-card__meta">
                  <span class="resource-list-card__name">{{ tool.toolId }}</span>
                  <span class="resource-list-card__desc">{{ tool.mcpServer ? `${tool.mcpServer} · ` : '' }}{{ tool.riskLevel || 'low' }}</span>
                </div>
                <div class="resource-list-card__actions">
                  <select v-model="tool.riskLevel" class="resource-field__select resource-list-card__risk">
                    <option value="low">Low</option>
                    <option value="medium">Medium</option>
                    <option value="high">High</option>
                  </select>
                  <button type="button" class="resource-list-card__action resource-list-card__action--danger" @click="removeTool(idx)">{{ $t('common.delete') }}</button>
                </div>
              </div>
            </div>
          </section>

          <section v-show="activeTab === 'knowledge'" class="resource-section">
            <DqEmpty v-if="!knowledge.bases.length" :description="$t('teams.noKnowledge')" />
            <div v-else class="resource-list-card">
              <div
                v-for="base in knowledge.bases"
                :key="base.id"
                class="resource-list-card__item"
                @click="toggleKnowledge(base.id)"
              >
                <DqCheckbox :model-value="agentForm.knowledgeIds?.includes(base.id)" />
                <div class="resource-list-card__meta">
                  <span class="resource-list-card__name">{{ base.name }}</span>
                  <span class="resource-list-card__desc">{{ $t('teams.documents', { count: base.documentCount }) }}</span>
                </div>
              </div>
            </div>
          </section>
        </div>

        <footer class="resource-workspace__footer">
          <span class="resource-workspace__hint">{{ $t('common.saveShortcut') }}</span>
          <div class="resource-workspace__footer-actions">
            <DqButton v-if="isCreating" @click="isCreating = false; selectedId = null">{{ $t('common.cancel') }}</DqButton>
            <DqButton v-if="!isCreating" @click="removeSelected">{{ $t('common.delete') }}</DqButton>
            <DqButton v-if="!isCreating" @click="resetSelected">{{ $t('common.reset') }}</DqButton>
            <DqButton type="primary" :disabled="saving" @click="save">
              {{ isCreating ? $t('teams.createAgent') : $t('common.save') }}
            </DqButton>
          </div>
        </footer>
      </template>
    </main>
  </div>
</template>

<style scoped>
.resource-rail__section {
  display: flex;
  flex-direction: column;
  min-height: 0;
  flex: 1;
  overflow: hidden;
}

.resource-rail__section-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 10px 10px 6px 14px;
  flex-shrink: 0;
}

.resource-rail__section-title {
  font-size: 11px;
  font-weight: 600;
  letter-spacing: 0.06em;
  text-transform: uppercase;
  color: var(--dq-label-tertiary);
}

.resource-rail__list {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  padding: 0 6px 6px;
}

.resource-rail__group + .resource-rail__group {
  margin-top: 8px;
}

.resource-rail__group-title {
  padding: 8px 12px 4px;
  font-size: 11px;
  font-weight: 600;
  letter-spacing: 0.04em;
  text-transform: uppercase;
  color: var(--dq-label-tertiary);
}

.resource-section--prompt {
  display: flex;
  flex-direction: column;
  flex: 1;
  min-height: 0;
}

.resource-section--prompt .md-editor {
  flex: 1;
  min-height: 360px;
}

.resource-list-card__risk {
  width: 90px;
  height: 28px;
  padding: 0 6px;
  border: 1px solid var(--dq-border-subtle);
  border-radius: 6px;
  background: color-mix(in srgb, var(--dq-bg-elevated) 65%, transparent);
  color: var(--dq-label-primary);
  font: inherit;
  font-size: 12px;
}

.resource-list-card__actions .resource-list-card__risk {
  margin-right: 4px;
}

.slider-row {
  display: flex;
  align-items: center;
  gap: 10px;
}

.slider-row :deep(.dq-slider) {
  flex: 1;
  min-width: 0;
}

.slider-row__value {
  flex-shrink: 0;
  min-width: 36px;
  text-align: right;
  font-size: 12px;
  font-weight: 500;
  font-family: ui-monospace, SFMono-Regular, 'SF Mono', Menlo, monospace;
  color: var(--dq-label-secondary);
}
</style>
