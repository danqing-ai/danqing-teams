<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useSkillsStore } from '@/stores/skills'
import { confirm, toast } from '@/utils/feedback'
import type { Skill, SkillFile } from '@/types'
import MdEditor from '@/components/common/MdEditor.vue'

type SkillTab = 'info' | 'body' | 'files' | 'tools'

const { t } = useI18n()
const store = useSkillsStore()

const selectedId = ref<string | null>(null)
const isCreating = ref(false)
const saving = ref(false)
const activeTab = ref<SkillTab>('info')
const pendingKeyword = ref('')
const pendingToolId = ref('')
const showImportDialog = ref(false)
const importPath = ref('')
const skillFiles = ref<SkillFile[]>([])
const viewingFile = ref<SkillFile | null>(null)

const skillForm = ref<Skill>(emptySkill())

function emptySkill(): Skill {
  return {
    id: '',
    name: '',
    description: '',
    license: '',
    compatibility: '',
    metadata: {},
    allowedTools: '',
    keywords: [],
    toolIds: [],
    systemHint: '',
    body: '',
    sourcePath: '',
  }
}

const sortedSkills = computed(() =>
  [...store.items].sort((a, b) => a.name.localeCompare(b.name, 'zh-CN')),
)
const builtinSkills = computed(() =>
  sortedSkills.value.filter((s) => s.builtin),
)
const customSkills = computed(() =>
  sortedSkills.value.filter((s) => !s.builtin),
)
const selectedSkill = computed(() =>
  selectedId.value ? store.items.find((s) => s.id === selectedId.value) : null,
)
const hasSelection = computed(() => isCreating.value || !!selectedId.value)
const headerTitle = computed(() => {
  if (isCreating.value) return t('skills.newSkill')
  if (selectedSkill.value) return selectedSkill.value.name || t('skills.untitled')
  return ''
})
const showViewerDialog = ref(false)

const metadataEntries = computed(() => {
  return Object.entries(skillForm.value.metadata ?? {}).map(([k, v]) => ({ key: k, value: v }))
})

onMounted(() => {
  store.load()
  if (!selectedId.value && sortedSkills.value.length) {
    selectSkill(sortedSkills.value[0].id)
  }
})

watch(selectedId, async (id) => {
  if (id) {
    skillFiles.value = await store.getFiles(id)
  } else {
    skillFiles.value = []
  }
})

function selectSkill(id: string) {
  isCreating.value = false
  selectedId.value = id
  activeTab.value = 'info'
  const skill = store.items.find((s) => s.id === id)
  if (skill) {
    skillForm.value = { ...skill, keywords: skill.keywords ? [...skill.keywords] : [], toolIds: skill.toolIds ? [...skill.toolIds] : [] }
  }
}

function startCreate() {
  isCreating.value = true
  selectedId.value = null
  skillForm.value = emptySkill()
  activeTab.value = 'info'
  skillFiles.value = []
}

function initial(name: string) {
  return name.trim().charAt(0).toUpperCase() || '?'
}

async function save() {
  if (!skillForm.value.id.trim()) {
    toast.warning(t('skills.idPlaceholder'))
    return
  }
  if (!skillForm.value.name.trim()) {
    toast.warning(t('skills.namePlaceholder'))
    return
  }
  saving.value = true
  try {
    if (isCreating.value) {
      await store.create({ ...skillForm.value, id: skillForm.value.id.trim() })
      toast.success(t('skills.created'))
      isCreating.value = false
      selectedId.value = skillForm.value.id.trim()
    } else if (selectedId.value) {
      await store.update(selectedId.value, { ...skillForm.value })
      toast.success(t('skills.saved'))
    }
  } catch (e) {
    toast.error(e instanceof Error ? e.message : t('common.saveFailed'))
  } finally {
    saving.value = false
  }
}

async function removeSelected() {
  if (!selectedSkill.value) return
  try {
    await confirm(t('skills.deleteConfirm', { name: selectedSkill.value.name }), t('skills.deleteTitle'), { type: 'warning' })
    await store.remove(selectedSkill.value.id)
    selectedId.value = null
    toast.success(t('skills.deleted'))
  } catch (e) {
    if (e instanceof Error) toast.error(e.message)
  }
}

async function resetSelected() {
  if (!selectedSkill.value) return
  try {
    await confirm(t('skills.resetConfirm', { name: selectedSkill.value.name }), t('skills.resetTitle'), { type: 'warning' })
  } catch {
    return
  }
  try {
    const s = await store.reset(selectedSkill.value.id)
    selectSkill(s.id)
    toast.success(t('skills.reset'))
  } catch (e) {
    toast.error(e instanceof Error ? e.message : t('skills.resetFailed'))
  }
}

async function doImport() {
  if (!importPath.value.trim()) {
    toast.warning('请输入技能目录路径')
    return
  }
  try {
    const result = await store.importDir(importPath.value.trim())
    showImportDialog.value = false
    importPath.value = ''
    toast.success(t('skills.importSuccess', { name: result.skill.name, count: result.fileCount }))
    selectSkill(result.skill.id)
  } catch (e) {
    toast.error(e instanceof Error ? e.message : t('skills.importFailed'))
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

function addMetadata() {
  skillForm.value.metadata = { ...(skillForm.value.metadata ?? {}), '': '' }
}

function updateMetadataKey(oldKey: string, newKey: string, idx: number) {
  const entries = Object.entries(skillForm.value.metadata ?? {})
  entries[idx] = [newKey, entries[idx][1]]
  skillForm.value.metadata = Object.fromEntries(entries)
}

function updateMetadataValue(key: string, value: string, idx: number) {
  const entries = Object.entries(skillForm.value.metadata ?? {})
  entries[idx] = [entries[idx][0], value]
  skillForm.value.metadata = Object.fromEntries(entries)
}

function removeMetadata(key: string) {
  const m = { ...(skillForm.value.metadata ?? {}) }
  delete m[key]
  skillForm.value.metadata = m
}

async function viewFileContent(file: SkillFile) {
  try {
    const content = await store.getFileContent(file.skillId, file.path)
    viewingFile.value = { ...file, content }
    showViewerDialog.value = true
  } catch (e) {
    toast.error(t('skills.loadFileFailed'))
  }
}

function onViewerClose(open: boolean) {
  if (!open) {
    viewingFile.value = null
    showViewerDialog.value = false
  }
}

function onKeydown(e: KeyboardEvent) {
  if ((e.metaKey || e.ctrlKey) && e.key === 's') {
    e.preventDefault()
    save()
  }
}

function formatSize(bytes: number): string {
  if (!bytes) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB']
  let i = 0
  let size = bytes
  while (size >= 1024 && i < units.length - 1) { size /= 1024; i++ }
  return `${size.toFixed(i === 0 ? 0 : 1)} ${units[i]}`
}
</script>

<template>
  <div class="resource-shell float-island" @keydown="onKeydown">
    <aside class="resource-rail">
      <div class="resource-rail__section">
        <div class="resource-rail__section-head">
          <span class="resource-rail__section-title">{{ $t('skills.title') }}</span>
          <DqIconButton :aria-label="$t('skills.newSkill')" @click="startCreate">
            <svg viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M12 5v14M5 12h14" stroke-linecap="round" />
            </svg>
          </DqIconButton>
        </div>
        <DqEmpty v-if="!sortedSkills.length" class="resource-rail__empty" :description="$t('skills.noSkills')" />
        <template v-else>
          <div v-if="builtinSkills.length" class="resource-rail__group">
            <div class="resource-rail__group-title">{{ $t('skills.builtinSkills') }}</div>
            <nav class="resource-rail__list" :aria-label="$t('skills.builtinSkills')">
              <button
                v-for="skill in builtinSkills"
                :key="skill.id"
                type="button"
                class="resource-rail__row"
                :class="{ 'is-active': selectedSkill?.id === skill.id && !isCreating }"
                @click="selectSkill(skill.id)"
              >
                <span class="resource-rail__avatar">{{ initial(skill.name) }}</span>
                <span class="resource-rail__meta">
                  <span class="resource-rail__name">{{ skill.name }}</span>
                </span>
              </button>
            </nav>
          </div>
          <div v-if="customSkills.length" class="resource-rail__group">
            <div class="resource-rail__group-title">{{ $t('skills.customSkills') }}</div>
            <nav class="resource-rail__list" :aria-label="$t('skills.customSkills')">
              <button
                v-for="skill in customSkills"
                :key="skill.id"
                type="button"
                class="resource-rail__row"
                :class="{ 'is-active': selectedSkill?.id === skill.id && !isCreating }"
                @click="selectSkill(skill.id)"
              >
                <span class="resource-rail__avatar">{{ initial(skill.name) }}</span>
                <span class="resource-rail__meta">
                  <span class="resource-rail__name">{{ skill.name }}</span>
                  <span class="resource-rail__desc">{{ skill.sourcePath ? $t('skills.imported') : '' }}</span>
                </span>
              </button>
            </nav>
          </div>
        </template>
      </div>
    </aside>

    <main class="resource-workspace">
      <div v-if="!hasSelection" class="resource-workspace__empty">
        <DqEmpty :description="$t('skills.emptySelection')">
          <div class="resource-workspace__empty-actions">
            <DqButton @click="startCreate">{{ $t('skills.newSkill') }}</DqButton>
            <DqButton @click="showImportDialog = true">{{ $t('skills.import') }}</DqButton>
          </div>
        </DqEmpty>
      </div>

      <template v-else>
        <header class="resource-workspace__bar">
          <div class="resource-workspace__identity">
            <h1 class="resource-workspace__title">{{ headerTitle }}</h1>
          </div>
          <nav v-if="selectedSkill || isCreating" class="resource-workspace__tabs" role="tablist">
            <button type="button" class="resource-workspace__tab" :class="{ 'is-active': activeTab === 'info' }" role="tab" @click="activeTab = 'info'">{{ $t('common.basicInfo') }}</button>
            <button type="button" class="resource-workspace__tab" :class="{ 'is-active': activeTab === 'body' }" role="tab" @click="activeTab = 'body'">{{ $t('skills.instructions') }}</button>
            <button type="button" class="resource-workspace__tab" :class="{ 'is-active': activeTab === 'files' }" role="tab" @click="activeTab = 'files'">{{ $t('skills.files') }} <span v-if="skillFiles.length" class="resource-workspace__tab-badge">{{ skillFiles.length }}</span></button>
            <button type="button" class="resource-workspace__tab" :class="{ 'is-active': activeTab === 'tools' }" role="tab" @click="activeTab = 'tools'">{{ $t('common.tools') }} <span v-if="(skillForm.toolIds ?? []).length" class="resource-workspace__tab-badge">{{ (skillForm.toolIds ?? []).length }}</span></button>
          </nav>
        </header>

        <div class="resource-workspace__scroll">
          <!-- Info Tab -->
          <section v-show="activeTab === 'info'" class="resource-section">
            <div class="resource-form-grid resource-form-grid--3">
              <label class="resource-field">
                <span class="resource-field__label">{{ $t('skills.skillId') }}</span>
                <DqInput v-model="skillForm.id" class="resource-input-mono" placeholder="my-skill" :disabled="!isCreating" />
                <span v-if="isCreating" class="resource-field__hint">{{ $t('skills.idHint') }}</span>
              </label>
              <label class="resource-field">
                <span class="resource-field__label">{{ $t('common.name') }}</span>
                <DqInput v-model="skillForm.name" :placeholder="$t('skills.namePlaceholder')" />
              </label>
              <label class="resource-field">
                <span class="resource-field__label">License</span>
                <DqInput v-model="skillForm.license" placeholder="MIT" />
              </label>
            </div>
            <div class="resource-form-grid resource-form-grid--2">
              <label class="resource-field">
                <span class="resource-field__label">Compatibility</span>
                <DqInput v-model="skillForm.compatibility" placeholder="Requires git, python3" />
              </label>
              <label class="resource-field">
                <span class="resource-field__label">Allowed Tools</span>
                <DqInput v-model="skillForm.allowedTools" placeholder="Bash Read" />
              </label>
            </div>
            <div class="resource-form-grid resource-form-grid--2">
              <label class="resource-field">
                <span class="resource-field__label">{{ $t('skills.keywords') }}</span>
                <div class="resource-chip-list">
                  <span v-for="(kw, idx) in (skillForm.keywords ?? [])" :key="idx" class="resource-chip">
                    {{ kw }}
                    <button type="button" class="resource-chip__remove" @click="removeKeyword(idx)">×</button>
                  </span>
                  <input v-model="pendingKeyword" class="resource-chip-list__add" placeholder="+" @keydown.enter.prevent="addKeyword" />
                </div>
              </label>
              <label class="resource-field">
                <span class="resource-field__label">Metadata</span>
                <div class="resource-meta-list">
                  <div v-for="(entry, idx) in metadataEntries" :key="idx" class="resource-meta-row">
                    <input v-model="entry.key" class="resource-meta-key" placeholder="key" @change="updateMetadataKey(metadataEntries[idx].key, entry.key, idx)" />
                    <input v-model="entry.value" class="resource-meta-value" placeholder="value" @change="updateMetadataValue(entry.key, entry.value, idx)" />
                    <button type="button" class="resource-meta-remove" @click="removeMetadata(entry.key)">×</button>
                  </div>
                  <button type="button" class="resource-meta-add" @click="addMetadata">+ {{ $t('skills.add') }}</button>
                </div>
              </label>
            </div>
            <label class="resource-field resource-field--block">
              <span class="resource-field__label">{{ $t('common.description') }}</span>
              <DqInput v-model="skillForm.description" type="textarea" :rows="4" :placeholder="$t('skills.descriptionPlaceholder')" />
            </label>
            <label class="resource-field resource-field--block">
              <span class="resource-field__label">{{ $t('skills.systemHint') }}</span>
              <DqInput v-model="skillForm.systemHint" type="textarea" :rows="4" :placeholder="$t('skills.systemHintPlaceholder')" />
            </label>
          </section>

          <!-- Body Tab (Markdown Editor) -->
          <section v-show="activeTab === 'body'" class="resource-section resource-section--body">
            <MdEditor v-model="skillForm.body" :label="$t('skills.bodyLabel')" :rows="20" :placeholder="$t('skills.bodyPlaceholder')" />
          </section>

          <!-- Files Tab -->
          <section v-show="activeTab === 'files'" class="resource-section">
            <div v-if="!skillFiles.length" class="resource-section__empty">
              <DqEmpty :description="$t('skills.noFiles')" />
            </div>
            <div v-else class="resource-list-card">
              <div v-for="file in skillFiles" :key="file.id" class="resource-list-card__item">
                <div class="resource-list-card__meta">
                  <span class="resource-list-card__name">{{ file.path }}</span>
                  <span class="resource-list-card__desc">{{ formatSize(file.size) }}</span>
                </div>
                <div class="resource-list-card__actions">
                  <DqButton size="small" @click="viewFileContent(file)">{{ $t('skills.view') }}</DqButton>
                </div>
              </div>
            </div>
          </section>

          <!-- Tools Tab -->
          <section v-show="activeTab === 'tools'" class="resource-section">
            <div class="resource-form-grid resource-form-grid--2">
              <label class="resource-field">
                <span class="resource-field__label">{{ $t('toolsPage.toolId') }}</span>
                <DqInput v-model="pendingToolId" class="resource-input-mono" placeholder="search_kb" @keydown.enter.prevent="addSkillTool" />
              </label>
              <div class="resource-field resource-field--action">
                <DqButton @click="addSkillTool">{{ $t('common.addTool') }}</DqButton>
              </div>
            </div>
            <div class="resource-list-card">
              <div v-for="id in (skillForm.toolIds ?? [])" :key="id" class="resource-list-card__item">
                <div class="resource-list-card__meta">
                  <span class="resource-list-card__name">{{ id }}</span>
                </div>
                <div class="resource-list-card__actions">
                  <button type="button" class="resource-list-card__action resource-list-card__action--danger" @click="removeSkillTool(id)">{{ $t('common.delete') }}</button>
                </div>
              </div>
            </div>
          </section>
        </div>

        <footer class="resource-workspace__footer">
          <span class="resource-workspace__hint">{{ $t('common.saveShortcut') }}</span>
          <div class="resource-workspace__footer-actions">
            <DqButton v-if="isCreating" @click="isCreating = false; selectedId = null">{{ $t('common.cancel') }}</DqButton>
            <DqButton v-if="!isCreating && selectedSkill?.builtin" @click="resetSelected">{{ $t('common.reset') }}</DqButton>
            <DqButton v-if="!isCreating" @click="removeSelected">{{ $t('common.delete') }}</DqButton>
            <DqButton type="primary" :disabled="saving" @click="save">
              {{ isCreating ? $t('common.create') : $t('common.save') }}
            </DqButton>
          </div>
        </footer>
      </template>
    </main>
  </div>

  <!-- File Viewer Dialog -->
  <DqDialog v-model:open="showViewerDialog" :title="viewingFile?.path ?? ''" variant="glass" width="700px" :closable="true" @update:open="onViewerClose">
    <pre class="file-content">{{ viewingFile?.content }}</pre>
  </DqDialog>

  <!-- Import Dialog -->
  <DqDialog v-model:open="showImportDialog" :title="$t('skills.importTitle')" variant="glass" width="460px" :closable="true">
    <div class="import-form">
      <label class="import-field">
        <span class="import-field__label">{{ $t('skills.importPath') }}</span>
        <input v-model="importPath" class="import-field__input" type="text" placeholder="/path/to/skill-directory" spellcheck="false" @keydown.enter="doImport" />
        <span class="import-field__hint">{{ $t('skills.importHint') }}</span>
      </label>
    </div>
    <template #footer>
      <div class="import-actions">
        <DqButton @click="showImportDialog = false">{{ $t('common.cancel') }}</DqButton>
        <DqButton type="primary" :disabled="!importPath.trim()" @click="doImport">{{ $t('skills.import') }}</DqButton>
      </div>
    </template>
  </DqDialog>
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
  font-size: var(--dq-font-size-caption);
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
  font-size: var(--dq-font-size-caption);
  font-weight: 600;
  letter-spacing: 0.04em;
  text-transform: uppercase;
  color: var(--dq-label-tertiary);
}

.resource-workspace__empty-actions {
  display: flex;
  gap: 8px;
  margin-top: 12px;
  justify-content: center;
}

.resource-workspace__tab-badge {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 18px;
  height: 18px;
  padding: 0 4px;
  border-radius: 9px;
  background: color-mix(in srgb, var(--dq-accent) 18%, transparent);
  color: var(--dq-accent);
  font-size: var(--dq-font-size-caption);
  font-weight: 600;
  margin-left: 4px;
}

.resource-section--body {
  display: flex;
  flex-direction: column;
  flex: 1;
  min-height: 0;
}

.resource-section--body .md-editor {
  flex: 1;
  min-height: 360px;
}

.resource-section--body .md-editor__body {
  flex: 1;
  min-height: 320px;
}

.resource-section--body .md-editor__textarea {
  min-height: 320px;
  resize: vertical;
}

.resource-meta-list {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.resource-meta-row {
  display: flex;
  gap: 4px;
  align-items: center;
}

.resource-meta-key,
.resource-meta-value {
  flex: 1;
  padding: 5px 8px;
  border: 1px solid var(--dq-border);
  border-radius: 6px;
  background: var(--dq-fill-on-glass-subtle);
  color: var(--dq-label-primary);
  font-size: var(--dq-font-size-footnote);
  font-family: inherit;
  outline: none;
}

.resource-meta-key:focus,
.resource-meta-value:focus {
  border-color: var(--dq-accent);
}

.resource-meta-remove {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 22px;
  height: 22px;
  border: none;
  border-radius: 4px;
  background: transparent;
  color: var(--dq-label-tertiary);
  font-size: var(--dq-font-size-title);
  cursor: pointer;
}

.resource-meta-add {
  align-self: flex-start;
  padding: 4px 8px;
  border: none;
  border-radius: 6px;
  background: transparent;
  color: var(--dq-accent);
  font-size: var(--dq-font-size-footnote);
  font-weight: 500;
  cursor: pointer;
  transition: background 0.12s ease;
}

.resource-meta-add:hover {
  background: color-mix(in srgb, var(--dq-accent) 8%, transparent);
}

.import-form {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.import-field {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.import-field__label {
  font-size: var(--dq-font-size-footnote);
  font-weight: 500;
  color: var(--dq-label-secondary);
}

.import-field__input {
  width: 100%;
  padding: 8px 12px;
  border: 1px solid var(--dq-border);
  border-radius: 8px;
  background: var(--dq-fill-on-glass-subtle);
  color: var(--dq-label-primary);
  font-size: var(--dq-font-size-body);
  font-family: ui-monospace, monospace;
  outline: none;
}

.import-field__hint {
  font-size: var(--dq-font-size-caption);
  color: var(--dq-label-tertiary);
}

.import-actions {
  display: flex;
  gap: 8px;
  justify-content: flex-end;
}

.file-content {
  max-height: 400px;
  overflow: auto;
  padding: 14px;
  border-radius: 8px;
  background: var(--dq-bg-page);
  font-size: var(--dq-font-size-footnote);
  font-family: ui-monospace, monospace;
  line-height: 1.5;
  white-space: pre-wrap;
  word-break: break-all;
}
</style>
