<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useSkillsStore } from '@/stores/skills'
import { confirm, toast } from '@/utils/feedback'
import type { RiskLevel, Skill, Tool } from '@/types'

type Selection = { kind: 'skill'; id: string } | { kind: 'tool'; id: string } | null
type SkillTab = 'info' | 'tools'

const { t } = useI18n()
const skills = useSkillsStore()

const selection = ref<Selection>(null)
const isCreating = ref(false)
const createKind = ref<'skill' | 'tool'>('skill')
const saving = ref(false)
const activeSkillTab = ref<SkillTab>('info')
const pendingToolId = ref('')
const pendingKeyword = ref('')

const riskOptions: { value: RiskLevel; label: string }[] = [
  { value: 'low', label: 'Low' },
  { value: 'medium', label: 'Medium' },
  { value: 'high', label: 'High' },
]

const skillForm = ref<Skill>({
  id: '',
  name: '',
  description: '',
  domainId: '',
  keywords: [],
  riskLevel: 'low',
  toolIds: [],
  systemHint: '',
})

const toolForm = ref<Tool>({
  id: '',
  name: '',
  description: '',
  type: 'builtin',
  riskLevel: 'low',
  schema: '',
})

const sortedSkills = computed(() =>
  [...skills.items].sort((a, b) => a.name.localeCompare(b.name, 'zh-CN')),
)
const sortedTools = computed(() =>
  [...skills.tools].sort((a, b) => a.name.localeCompare(b.name, 'zh-CN')),
)
const selectedSkill = computed(() =>
  selection.value?.kind === 'skill' ? skills.items.find((s) => s.id === selection.value!.id) : null,
)
const selectedTool = computed(() =>
  selection.value?.kind === 'tool' ? skills.tools.find((t) => t.id === selection.value!.id) : null,
)
const hasSelection = computed(() => isCreating.value || !!selection.value)
const headerTitle = computed(() => {
  if (isCreating.value) return createKind.value === 'skill' ? t('skills.newSkill') : t('toolsPage.newTool')
  if (selectedSkill.value) return selectedSkill.value.name || t('skills.untitled')
  if (selectedTool.value) return selectedTool.value.name || t('toolsPage.untitled')
  return ''
})

onMounted(() => {
  skills.load()
  if (sortedSkills.value.length && !selection.value) {
    selectSkill(sortedSkills.value[0].id)
  }
})

function selectSkill(id: string) {
  isCreating.value = false
  selection.value = { kind: 'skill', id }
  activeSkillTab.value = 'info'
  const skill = skills.items.find((s) => s.id === id)
  if (skill) skillForm.value = { ...skill, keywords: skill.keywords ? [...skill.keywords] : [], toolIds: skill.toolIds ? [...skill.toolIds] : [] }
}

function selectTool(id: string) {
  isCreating.value = false
  selection.value = { kind: 'tool', id }
  const tool = skills.tools.find((t) => t.id === id)
  if (tool) toolForm.value = { ...tool }
}

function startCreate(kind: 'skill' | 'tool') {
  isCreating.value = true
  createKind.value = kind
  selection.value = null
  if (kind === 'skill') {
    skillForm.value = {
      id: '',
      name: '',
      description: '',
      domainId: '',
      keywords: [],
      riskLevel: 'low',
      toolIds: [],
      systemHint: '',
    }
    activeSkillTab.value = 'info'
  } else {
    toolForm.value = { id: '', name: '', description: '', type: 'builtin', riskLevel: 'low', schema: '' }
  }
}

function save() {
  if (createKind.value === 'skill' || selectedSkill.value) {
    if (!skillForm.value.id.trim()) {
      toast.warning(t('skills.idPlaceholder'))
      return
    }
    if (!skillForm.value.name.trim()) {
      toast.warning(t('skills.namePlaceholder'))
      return
    }
  } else {
    if (!toolForm.value.id.trim()) {
      toast.warning(t('toolsPage.idPlaceholder'))
      return
    }
    if (!toolForm.value.name.trim()) {
      toast.warning(t('toolsPage.namePlaceholder'))
      return
    }
  }

  saving.value = true
  try {
    if (isCreating.value && createKind.value === 'skill') {
      skills.createSkill({ ...skillForm.value, id: skillForm.value.id.trim() })
      toast.success(t('skills.created'))
      isCreating.value = false
      selectSkill(skillForm.value.id.trim())
    } else if (selectedSkill.value) {
      skills.updateSkill(selectedSkill.value.id, { ...skillForm.value })
      toast.success(t('skills.saved'))
      selectSkill(selectedSkill.value.id)
    } else if (isCreating.value && createKind.value === 'tool') {
      skills.createTool({ ...toolForm.value, id: toolForm.value.id.trim() })
      toast.success(t('toolsPage.created'))
      isCreating.value = false
      selectTool(toolForm.value.id.trim())
    } else if (selectedTool.value) {
      skills.updateTool(selectedTool.value.id, { ...toolForm.value })
      toast.success(t('toolsPage.saved'))
      selectTool(selectedTool.value.id)
    }
  } catch (e) {
    toast.error(e instanceof Error ? e.message : t('common.saveFailed'))
  } finally {
    saving.value = false
  }
}

async function removeSelected() {
  try {
    if (selectedSkill.value) {
      await confirm(t('skills.deleteConfirm', { name: selectedSkill.value.name }), t('skills.deleteTitle'), { type: 'warning' })
      skills.removeSkill(selectedSkill.value.id)
      selection.value = null
      toast.success(t('skills.deleted'))
    } else if (selectedTool.value) {
      await confirm(t('toolsPage.deleteConfirm', { name: selectedTool.value.name }), t('toolsPage.deleteTitle'), { type: 'warning' })
      skills.removeTool(selectedTool.value.id)
      selection.value = null
      toast.success(t('toolsPage.deleted'))
    }
  } catch (e) {
    if (e instanceof Error) toast.error(e.message)
  }
}

function addKeyword() {
  if (!pendingKeyword.value.trim()) return
  skillForm.value.keywords = [...(skillForm.value.keywords ?? []), pendingKeyword.value.trim()]
  pendingKeyword.value = ''
}

function removeKeyword(idx: number) {
  skillForm.value.keywords = (skillForm.value.keywords ?? []).filter((_, i) => i !== idx)
}

function addSkillTool() {
  if (!pendingToolId.value.trim()) return
  const ids = skillForm.value.toolIds ?? []
  if (!ids.includes(pendingToolId.value.trim())) {
    skillForm.value.toolIds = [...ids, pendingToolId.value.trim()]
  }
  pendingToolId.value = ''
}

function removeSkillTool(id: string) {
  skillForm.value.toolIds = (skillForm.value.toolIds ?? []).filter((x) => x !== id)
}

function initial(name: string) {
  return name.trim().charAt(0).toUpperCase() || '?'
}

function onKeydown(e: KeyboardEvent) {
  if ((e.metaKey || e.ctrlKey) && e.key === 's') {
    e.preventDefault()
    save()
  }
}
</script>

<template>
  <div class="resource-shell float-island" @keydown="onKeydown">
    <aside class="resource-rail">
      <div class="resource-rail__section">
        <div class="resource-rail__section-head">
          <span class="resource-rail__section-title">{{ $t('skills.title') }}</span>
          <DqIconButton :aria-label="$t('skills.newSkill')" @click="startCreate('skill')">
            <svg viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M12 5v14M5 12h14" stroke-linecap="round" />
            </svg>
          </DqIconButton>
        </div>
        <DqEmpty v-if="!sortedSkills.length" class="resource-rail__empty" :description="$t('skills.noSkills')" />
        <nav v-else class="resource-rail__list" :aria-label="$t('skills.skillList')">
          <button
            v-for="skill in sortedSkills"
            :key="skill.id"
            type="button"
            class="resource-rail__row"
            :class="{ 'is-active': selectedSkill?.id === skill.id && !isCreating }"
            @click="selectSkill(skill.id)"
          >
            <span class="resource-rail__avatar">{{ initial(skill.name) }}</span>
            <span class="resource-rail__meta">
              <span class="resource-rail__name">{{ skill.name }}</span>
              <span class="resource-rail__desc">{{ skill.domainId || $t('skills.domain') }}</span>
            </span>
          </button>
        </nav>
      </div>

      <div class="resource-rail__section">
        <div class="resource-rail__section-head">
          <span class="resource-rail__section-title">{{ $t('toolsPage.title') }}</span>
          <DqIconButton :aria-label="$t('toolsPage.newTool')" @click="startCreate('tool')">
            <svg viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M12 5v14M5 12h14" stroke-linecap="round" />
            </svg>
          </DqIconButton>
        </div>
        <DqEmpty v-if="!sortedTools.length" class="resource-rail__empty" :description="$t('toolsPage.noTools')" />
        <nav v-else class="resource-rail__list" :aria-label="$t('toolsPage.toolList')">
          <button
            v-for="tool in sortedTools"
            :key="tool.id"
            type="button"
            class="resource-rail__row"
            :class="{ 'is-active': selectedTool?.id === tool.id && !isCreating }"
            @click="selectTool(tool.id)"
          >
            <span class="resource-rail__avatar">{{ initial(tool.name) }}</span>
            <span class="resource-rail__meta">
              <span class="resource-rail__name">{{ tool.name }}</span>
              <span class="resource-rail__desc">{{ tool.type }}</span>
            </span>
          </button>
        </nav>
      </div>
    </aside>

    <main class="resource-workspace">
      <div v-if="!hasSelection" class="resource-workspace__empty">
        <DqEmpty :description="$t('skills.emptySelection')">
          <p class="resource-workspace__hint">{{ $t('skills.emptySelectionHint') }}</p>
        </DqEmpty>
      </div>

      <template v-else>
        <header class="resource-workspace__bar">
          <div class="resource-workspace__identity">
            <h1 class="resource-workspace__title">{{ headerTitle }}</h1>
          </div>
          <nav v-if="selectedSkill || (isCreating && createKind === 'skill')" class="resource-workspace__tabs" role="tablist">
            <button
              type="button"
              class="resource-workspace__tab"
              :class="{ 'is-active': activeSkillTab === 'info' }"
              role="tab"
              :aria-selected="activeSkillTab === 'info'"
              @click="activeSkillTab = 'info'"
            >
              {{ $t('common.basicInfo') }}
            </button>
            <button
              type="button"
              class="resource-workspace__tab"
              :class="{ 'is-active': activeSkillTab === 'tools' }"
              role="tab"
              :aria-selected="activeSkillTab === 'tools'"
              @click="activeSkillTab = 'tools'"
            >
              {{ $t('common.tools') }}
            </button>
          </nav>
        </header>

        <div class="resource-workspace__scroll">
          <template v-if="selectedSkill || (isCreating && createKind === 'skill')">
            <section v-show="activeSkillTab === 'info'" class="resource-section">
              <div class="resource-form-grid resource-form-grid--3">
                <label class="resource-field"
                >
                  <span class="resource-field__label">{{ $t('skills.skillId') }}</span>
                  <DqInput v-model="skillForm.id" class="resource-input-mono" placeholder="web-search" :disabled="!isCreating" />
                  <span v-if="isCreating" class="resource-field__hint">{{ $t('skills.idHint') }}</span>
                </label>
                <label class="resource-field"
                >
                  <span class="resource-field__label">{{ $t('common.name') }}</span>
                  <DqInput v-model="skillForm.name" placeholder="Web Search" />
                </label>
                <label class="resource-field"
                >
                  <span class="resource-field__label">Domain</span>
                  <DqInput v-model="skillForm.domainId" class="resource-input-mono" placeholder="default" />
                </label>
              </div>
              <div class="resource-form-grid resource-form-grid--2">
                <label class="resource-field"
                >
                  <span class="resource-field__label">{{ $t('common.riskLevel') }}</span>
                  <select v-model="skillForm.riskLevel" class="resource-field__select">
                    <option v-for="opt in riskOptions" :key="opt.value" :value="opt.value">{{ opt.label }}</option>
                  </select>
                </label>
                <label class="resource-field"
                >
                  <span class="resource-field__label">{{ $t('skills.keywords') }}</span>
                  <div class="resource-chip-list"
                  >
                    <span v-for="(kw, idx) in skillForm.keywords" :key="idx" class="resource-chip"
                    >
                      {{ kw }}
                      <button type="button" class="resource-chip__remove" @click="removeKeyword(idx)">×</button>
                    </span>
                    <input
                      v-model="pendingKeyword"
                      class="resource-chip-list__add"
                      placeholder="+"
                      @keydown.enter.prevent="addKeyword"
                    />
                  </div>
                </label>
              </div>
              <label class="resource-field resource-field--block"
              >
                <span class="resource-field__label">{{ $t('common.description') }}</span>
                <DqInput v-model="skillForm.description" type="textarea" :rows="4" :placeholder="$t('skills.descriptionPlaceholder')" />
              </label>
              <label class="resource-field resource-field--block"
              >
                <span class="resource-field__label">{{ $t('skills.systemHint') }}</span>
                <DqInput v-model="skillForm.systemHint" type="textarea" :rows="6" :placeholder="$t('skills.systemHintPlaceholder')" />
              </label>
            </section>

            <section v-show="activeSkillTab === 'tools'" class="resource-section">
              <div class="resource-form-grid resource-form-grid--2">
                <label class="resource-field"
                >
                  <span class="resource-field__label">{{ $t('toolsPage.toolId') }}</span>
                  <DqInput v-model="pendingToolId" class="resource-input-mono" placeholder="search_kb" @keydown.enter.prevent="addSkillTool" />
                </label>
                <div class="resource-field resource-field--action">
                >
                  <DqButton @click="addSkillTool">{{ $t('common.addTool') }}</DqButton>
                </div>
              </div>
              <div class="resource-list-card">
                <div v-for="id in skillForm.toolIds" :key="id" class="resource-list-card__item"
                >
                  <div class="resource-list-card__meta"
                  >
                    <span class="resource-list-card__name">{{ sortedTools.find((t) => t.id === id)?.name ?? id }}</span>
                    <span class="resource-list-card__desc">{{ id }}</span>
                  </div>
                  <div class="resource-list-card__actions"
                  >
                    <button type="button" class="resource-list-card__action resource-list-card__action--danger" @click="removeSkillTool(id)">{{ $t('common.delete') }}</button>
                  </div>
                </div>
              </div>
            </section>
          </template>

          <template v-else>
            <section class="resource-section">
              <div class="resource-form-grid resource-form-grid--3">
                <label class="resource-field"
                >
                  <span class="resource-field__label">{{ $t('toolsPage.toolId') }}</span>
                  <DqInput v-model="toolForm.id" class="resource-input-mono" placeholder="search_kb" :disabled="!isCreating" />
                </label>
                <label class="resource-field"
                >
                  <span class="resource-field__label">{{ $t('common.name') }}</span>
                  <DqInput v-model="toolForm.name" placeholder="Search KB" />
                </label>
                <label class="resource-field"
                >
                  <span class="resource-field__label">{{ $t('common.type') }}</span>
                  <select v-model="toolForm.type" class="resource-field__select">
                    <option value="builtin">Builtin</option>
                    <option value="mcp">MCP</option>
                  </select>
                </label>
              </div>
              <div class="resource-form-grid resource-form-grid--2">
                <label class="resource-field"
                >
                  <span class="resource-field__label">{{ $t('common.riskLevel') }}</span>
                  <select v-model="toolForm.riskLevel" class="resource-field__select">
                    <option v-for="opt in riskOptions" :key="opt.value" :value="opt.value">{{ opt.label }}</option>
                  </select>
                </label>
                <label class="resource-field"
                >
                  <span class="resource-field__label">{{ $t('common.mcpserverOptional') }}</span>
                  <DqInput v-model="toolForm.mcpServer" class="resource-input-mono" placeholder="mcp-server-id" />
                </label>
              </div>
              <label class="resource-field resource-field--block"
              >
                <span class="resource-field__label">{{ $t('common.description') }}</span>
                <DqInput v-model="toolForm.description" type="textarea" :rows="3" :placeholder="$t('toolsPage.descriptionPlaceholder')" />
              </label>
              <label class="resource-field resource-field--block"
              >
                <span class="resource-field__label">{{ $t('common.jsonSchema') }}</span>
                <DqInput v-model="toolForm.schema" class="resource-input-mono" type="textarea" :rows="8" placeholder="{}" />
                <span class="resource-field__hint">{{ $t('toolsPage.schemaHint') }}</span>
              </label>
            </section>
          </template>
        </div>

        <footer class="resource-workspace__footer">
          <span class="resource-workspace__hint">{{ $t('common.saveShortcut') }}</span>
          <div class="resource-workspace__footer-actions">
            <DqButton v-if="isCreating" @click="isCreating = false; selection = null">{{ $t('common.cancel') }}</DqButton>
            <DqButton v-if="!isCreating" @click="removeSelected">{{ $t('common.delete') }}</DqButton>
            <DqButton type="primary" :disabled="saving" @click="save">
              {{ isCreating ? $t('common.create') : $t('common.save') }}
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

.resource-rail__section + .resource-rail__section {
  border-top: 1px solid var(--dq-separator-light);
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

.resource-chip-list__add {
  width: 60px;
  height: 28px;
  text-align: center;
}
</style>
