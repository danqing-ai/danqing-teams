<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { Setting, Cpu, Search, Brush } from '@danqing/dq-shell'
import { useLLMStore } from '@/stores/llm'
import { useSearchConfigStore } from '@/stores/searchConfig'
import { useRuntimeConfigStore } from '@/stores/runtimeConfig'
import { useModelLimitsStore } from '@/stores/modelLimits'
import { useThemeStore, THEME_OPTIONS } from '@/stores/theme'
import type { ThemeId } from '@/stores/theme'
import { toast } from '@/utils/feedback'
import type { LLMProviderType, LLMProviderConfig, LLMModelRef, LLMProviderPreset, SearchProvider, ModelLimit } from '@/types/mission'

type SettingsTab = 'runtime' | 'models' | 'modelLimits' | 'search' | 'appearance'

const { t } = useI18n()
const activeTab = ref<SettingsTab>('models')
const llm = useLLMStore()
const searchConfig = useSearchConfigStore()
const runtimeConfig = useRuntimeConfigStore()
const modelLimits = useModelLimitsStore()
const themeStore = useThemeStore()

const providerOptions = computed<{ value: LLMProviderType; label: string }[]>(() => [
  { value: 'openai', label: 'OpenAI' },
  { value: 'anthropic', label: 'Anthropic' },
  { value: 'local', label: t('settings.localProvider') },
  { value: 'mock', label: t('settings.mockProvider') },
])

const editingId = ref<string | null>(null)
const showForm = ref(false)
const refreshingModels = ref(false)
const dialogStep = ref<'choose' | 'configure'>('choose')

const form = ref({
  provider: 'openai' as LLMProviderType,
  name: '',
  apiKey: '',
  baseUrl: '',
  models: [] as LLMModelRef[],
})

const searchForm = ref({
  provider: 'duckduckgo' as SearchProvider,
  baseUrl: '',
  apiKey: '',
  timeoutMs: 15000,
  maxResults: 5,
})

const runtimeForm = ref({
  autoApprove: false,
  doomLoopThreshold: 3,
  maxStepsDefault: 20,
  maxDelegationDepth: 3,
  recallTopK: 3,
  searchTopK: 3,
  compactionEnabled: false,
  compactionMaxTokens: 128000,
  compactionTriggerRatio: 0.85,
  compactionCutTokens: 16000,
  compactionTurnInterval: 6,
  compactionSubInterval: 4,
  compactionToolTruncate: 2000,
})

const modelLimitsForm = ref<ModelLimit[]>([])
const showModelLimitForm = ref(false)
const editingModelLimitIdx = ref<number | null>(null)
const modelLimitForm = ref({ model: '', contextWindow: 128000, maxOutput: 8192 })

onMounted(async () => {
  await Promise.all([
    llm.loadConfigs(),
    llm.loadModels(),
    llm.loadPresets(),
    searchConfig.loadConfig(),
    runtimeConfig.loadConfig(),
    modelLimits.load(),
  ])
  if (searchConfig.config) {
    searchForm.value = {
      provider: searchConfig.config.provider,
      baseUrl: searchConfig.config.baseUrl ?? '',
      apiKey: searchConfig.config.apiKey ?? '',
      timeoutMs: searchConfig.config.timeoutMs ?? 15000,
      maxResults: searchConfig.config.maxResults ?? 5,
    }
  }
  if (runtimeConfig.config) {
    runtimeForm.value = { ...runtimeConfig.config }
  }
  modelLimitsForm.value = [...modelLimits.limits]
})

const displayedModels = computed<LLMModelRef[]>(() => {
  if (editingId.value) {
    return llm.getConfig(editingId.value)?.models ?? []
  }
  return form.value.models
})

function providerLabel(p: LLMProviderType) {
  return providerOptions.value.find((o) => o.value === p)?.label ?? p
}

function openNewForm() {
  editingId.value = null
  dialogStep.value = 'choose'
  form.value = {
    provider: 'openai',
    name: '',
    apiKey: '',
    baseUrl: '',
    models: [],
  }
  showForm.value = true
}

function selectPreset(preset: LLMProviderPreset) {
  form.value = {
    provider: preset.provider,
    name: preset.name,
    apiKey: '',
    baseUrl: preset.baseUrl,
    models: [],
  }
  dialogStep.value = 'configure'
}

function selectCustom() {
  form.value = {
    provider: 'openai',
    name: '',
    apiKey: '',
    baseUrl: '',
    models: [],
  }
  dialogStep.value = 'configure'
}

function backToChoose() {
  dialogStep.value = 'choose'
}

const presetColors: Record<string, string> = {
  openai: 'var(--dq-success)',
  anthropic: 'var(--dq-warning)',
  deepseek: 'var(--dq-accent)',
  google: 'var(--dq-info)',
  zhipu: 'var(--dq-danger)',
  qwen: 'var(--dq-system-orange)',
  moonshot: 'var(--dq-system-blue)',
  ollama: 'var(--dq-label-secondary)',
}

function presetColor(id: string) {
  return presetColors[id] ?? 'var(--dq-label-secondary)'
}

function presetAbbr(id: string) {
  const map: Record<string, string> = {
    openai: 'GPT',
    anthropic: 'C',
    deepseek: 'DS',
    google: 'G',
    zhipu: 'GLM',
    qwen: 'Q',
    moonshot: 'K',
    ollama: '🦙',
  }
  return map[id] ?? id[0]?.toUpperCase() ?? '?'
}

function openEditForm(cfg: LLMProviderConfig) {
  editingId.value = cfg.id
  dialogStep.value = 'configure'
  form.value = {
    provider: cfg.provider,
    name: cfg.name,
    apiKey: cfg.apiKey ?? '',
    baseUrl: cfg.baseUrl ?? '',
    models: cfg.models ?? [],
  }
  showForm.value = true
}

function cancelForm() {
  showForm.value = false
  editingId.value = null
}

async function handleSave() {
  if (!form.value.name.trim()) {
    toast.warning(t('settings.namePlaceholder'))
    return
  }
  const models = editingId.value ? displayedModels.value : form.value.models
  if (!models.length) {
    toast.warning(t('settings.modelsRequired'))
    return
  }
  const payload = {
    provider: form.value.provider,
    name: form.value.name.trim(),
    apiKey: form.value.apiKey.trim() || undefined,
    baseUrl: form.value.baseUrl.trim() || undefined,
    models,
  }
  try {
    if (editingId.value) {
      await llm.updateConfig(editingId.value, payload)
    } else {
      await llm.saveConfig(payload)
    }
    cancelForm()
  } catch {
    /* toast already shown in store */
  }
}

async function handleDelete(id: string) {
  try {
    await llm.deleteConfig(id)
  } catch {
    /* toast already shown in store */
  }
}

async function handleRefreshModels() {
  refreshingModels.value = true
  try {
    if (editingId.value) {
      await llm.refreshModels(editingId.value)
    } else {
      const models = await llm.fetchModels({
        provider: form.value.provider,
        name: form.value.name.trim(),
        apiKey: form.value.apiKey.trim(),
        baseUrl: form.value.baseUrl.trim(),
        models: [],
      })
      form.value.models = models
    }
  } catch {
    /* toast already shown in store */
  } finally {
    refreshingModels.value = false
  }
}

async function handleToggleModel(modelName: string, enabled: boolean) {
  if (editingId.value) {
    try {
      await llm.toggleModel(editingId.value, modelName, enabled)
    } catch {
      /* toast already shown in store */
    }
    return
  }
  const model = form.value.models.find((m) => m.name === modelName)
  if (model) {
    model.enabled = enabled
  }
}

async function handleSaveSearch() {
  const payload = {
    provider: searchForm.value.provider,
    baseUrl: searchForm.value.baseUrl.trim() || undefined,
    apiKey: searchForm.value.apiKey.trim() || undefined,
    timeoutMs: searchForm.value.timeoutMs,
    maxResults: searchForm.value.maxResults,
  }
  try {
    await searchConfig.saveConfig(payload)
  } catch {
    /* toast already shown in store */
  }
}

async function handleSaveRuntime() {
  try {
    await runtimeConfig.saveConfig(runtimeForm.value)
  } catch {
    /* toast already shown in store */
  }
}

function openAddModelLimit() {
  editingModelLimitIdx.value = null
  modelLimitForm.value = { model: '', contextWindow: 128000, maxOutput: 8192 }
  showModelLimitForm.value = true
}

function openEditModelLimit(idx: number) {
  editingModelLimitIdx.value = idx
  const item = modelLimitsForm.value[idx]
  modelLimitForm.value = { model: item.model, contextWindow: item.contextWindow, maxOutput: item.maxOutput }
  showModelLimitForm.value = true
}

async function saveModelLimitForm() {
  if (!modelLimitForm.value.model.trim()) return
  const entry: ModelLimit = {
    model: modelLimitForm.value.model.trim(),
    contextWindow: Number(modelLimitForm.value.contextWindow) || 128000,
    maxOutput: Number(modelLimitForm.value.maxOutput) || 8192,
  }
  if (editingModelLimitIdx.value !== null) {
    modelLimitsForm.value[editingModelLimitIdx.value] = entry
  } else {
    const existing = modelLimitsForm.value.findIndex((l) => l.model === entry.model)
    if (existing >= 0) {
      modelLimitsForm.value[existing] = entry
    } else {
      modelLimitsForm.value.push(entry)
    }
  }
  showModelLimitForm.value = false
  try {
    await modelLimits.save(modelLimitsForm.value)
  } catch {
    /* toast already shown in store */
  }
}

async function removeModelLimit(idx: number) {
  modelLimitsForm.value.splice(idx, 1)
  try {
    await modelLimits.save(modelLimitsForm.value)
  } catch {
    /* toast already shown in store */
  }
}


const menuItems = computed(() => [
  { id: 'appearance' as SettingsTab, label: t('settings.appearance'), icon: Brush },
  { id: 'runtime' as SettingsTab, label: t('settings.runtime'), icon: Setting },
  { id: 'models' as SettingsTab, label: t('settings.models'), icon: Cpu },
  { id: 'modelLimits' as SettingsTab, label: t('settings.modelLimits'), icon: Setting },
  { id: 'search' as SettingsTab, label: t('settings.search'), icon: Search },
])

const footerHint = computed(() => {
  switch (activeTab.value) {
    case 'runtime': return t('common.saveShortcut')
    case 'search': return t('common.saveShortcut')
    case 'models': return t('settings.modelsHint')
    case 'modelLimits': return t('settings.modelLimitsHint')
    default: return ''
  }
})

const hasFooterActions = computed(() => {
  return ['runtime', 'search', 'models', 'modelLimits'].includes(activeTab.value)
})
</script>

<template>
  <div class="settings-view">
    <aside class="settings-sidebar">
      <div class="settings-sidebar__head">
        <span class="settings-sidebar__title">{{ $t('settings.title') }}</span>
      </div>
      <nav class="settings-sidebar__menu" :aria-label="$t('settings.category')">
        <button
          v-for="item in menuItems"
          :key="item.id"
          type="button"
          class="settings-sidebar__item"
          :class="{ 'is-active': activeTab === item.id }"
          @click="activeTab = item.id"
        >
          <DqIcon :size="18">
            <component :is="item.icon" />
          </DqIcon>
          <span>{{ item.label }}</span>
        </button>
      </nav>
    </aside>

    <main class="settings-panel">
      <div class="settings-panel__content">
      <div v-if="activeTab === 'appearance'" class="settings-section">
        <header class="settings-section__head">
          <h2>{{ $t('settings.appearance') }}</h2>
          <p>{{ $t('settings.appearanceDesc') }}</p>
        </header>

        <div class="theme-grid">
          <button
            v-for="theme in THEME_OPTIONS"
            :key="theme.id"
            type="button"
            class="theme-card"
            :class="{ 'is-active': themeStore.currentTheme === theme.id }"
            @click="themeStore.setTheme(theme.id as ThemeId)"
          >
            <div class="theme-card__preview">
              <div class="theme-card__swatch" :style="{ background: theme.accent }"></div>
              <div class="theme-card__bars">
                <span class="theme-card__bar" :style="{ background: theme.accent, opacity: 0.8 }"></span>
                <span class="theme-card__bar theme-card__bar--short" :style="{ background: theme.accent, opacity: 0.4 }"></span>
              </div>
            </div>
            <div class="theme-card__info">
              <span class="theme-card__name">{{ theme.label }}</span>
              <span class="theme-card__desc">{{ theme.description }}</span>
            </div>
            <div v-if="themeStore.currentTheme === theme.id" class="theme-card__check">
              <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3" stroke-linecap="round" stroke-linejoin="round">
                <polyline points="20 6 9 17 4 12"></polyline>
              </svg>
            </div>
          </button>
        </div>
      </div>

      <div v-else-if="activeTab === 'runtime'" class="settings-section">
        <header class="settings-section__head">
          <h2>{{ $t('settings.runtime') }}</h2>
          <p>{{ $t('settings.runtimeDesc') }}</p>
        </header>

        <div v-if="runtimeConfig.loading" class="settings-empty">{{ $t('common.loading') }}</div>

        <div v-else class="settings-form">
          <div class="settings-form-group">
            <h3 class="settings-form-group__title">{{ $t('settings.runtimeTurn') }}</h3>
            <p class="settings-form-group__desc">{{ $t('settings.runtimeTurnDesc') }}</p>
            <div class="settings-form-row">
              <div class="settings-field settings-field--half">
                <span class="settings-field__label">{{ $t('settings.doomLoopThreshold') }}</span>
                <div class="slider-row">
                  <DqSlider v-model="runtimeForm.doomLoopThreshold" :min="1" :max="10" :step="1" />
                  <span class="slider-row__value">{{ runtimeForm.doomLoopThreshold }}</span>
                </div>
              </div>
              <div class="settings-field settings-field--half">
                <span class="settings-field__label">{{ $t('settings.maxStepsDefault') }}</span>
                <div class="slider-row">
                  <DqSlider v-model="runtimeForm.maxStepsDefault" :min="5" :max="100" :step="1" />
                  <span class="slider-row__value">{{ runtimeForm.maxStepsDefault }}</span>
                </div>
              </div>
            </div>
          </div>

          <div class="settings-form-group">
            <h3 class="settings-form-group__title">{{ $t('settings.runtimeTeam') }}</h3>
            <p class="settings-form-group__desc">{{ $t('settings.runtimeTeamDesc') }}</p>
            <div class="settings-field">
              <span class="settings-field__label">{{ $t('settings.maxDelegationDepth') }}</span>
              <div class="slider-row">
                <DqSlider v-model="runtimeForm.maxDelegationDepth" :min="1" :max="10" :step="1" />
                <span class="slider-row__value">{{ runtimeForm.maxDelegationDepth }}</span>
              </div>
            </div>
          </div>

          <div class="settings-form-group">
            <h3 class="settings-form-group__title">{{ $t('settings.runtimeMemoryKnowledge') }}</h3>
            <p class="settings-form-group__desc">{{ $t('settings.runtimeMemoryKnowledgeDesc') }}</p>
            <div class="settings-form-row">
              <div class="settings-field settings-field--half">
                <span class="settings-field__label">{{ $t('settings.recallTopK') }}</span>
                <div class="slider-row">
                  <DqSlider v-model="runtimeForm.recallTopK" :min="1" :max="20" :step="1" />
                  <span class="slider-row__value">{{ runtimeForm.recallTopK }}</span>
                </div>
              </div>
              <div class="settings-field settings-field--half">
                <span class="settings-field__label">{{ $t('settings.searchTopK') }}</span>
                <div class="slider-row">
                  <DqSlider v-model="runtimeForm.searchTopK" :min="1" :max="20" :step="1" />
                  <span class="slider-row__value">{{ runtimeForm.searchTopK }}</span>
                </div>
              </div>
            </div>
          </div>

          <div class="settings-form-group">
            <h3 class="settings-form-group__title">{{ $t('settings.runtimeAutoApprove') }}</h3>
            <p class="settings-form-group__desc">{{ $t('settings.runtimeAutoApproveDesc') }}</p>
            <label class="settings-field settings-field--switch">
              <span class="settings-field__label">{{ $t('settings.autoApprove') }}</span>
              <DqSwitch
                :model-value="runtimeForm.autoApprove"
                size="small"
                @update:model-value="(v: boolean) => runtimeForm.autoApprove = v"
              />
            </label>
          </div>

          <div class="settings-form-group">
            <h3 class="settings-form-group__title">{{ $t('settings.runtimeCompaction') }}</h3>
            <p class="settings-form-group__desc">{{ $t('settings.runtimeCompactionDesc') }}</p>
            <label class="settings-field settings-field--switch">
              <span class="settings-field__label">{{ $t('settings.compactionEnabled') }}</span>
              <DqSwitch
                :model-value="runtimeForm.compactionEnabled"
                size="small"
                @update:model-value="(v: boolean) => runtimeForm.compactionEnabled = v"
              />
            </label>
            <template v-if="runtimeForm.compactionEnabled">
              <div class="settings-form-row">
                <div class="settings-field settings-field--half">
                  <span class="settings-field__label">{{ $t('settings.compactionMaxTokens') }}</span>
                  <div class="slider-row">
                    <DqSlider v-model="runtimeForm.compactionMaxTokens" :min="16000" :max="256000" :step="1000" />
                    <span class="slider-row__value">{{ runtimeForm.compactionMaxTokens }}</span>
                  </div>
                </div>
                <div class="settings-field settings-field--half">
                  <span class="settings-field__label">{{ $t('settings.compactionTriggerRatio') }}</span>
                  <div class="slider-row">
                    <DqSlider v-model="runtimeForm.compactionTriggerRatio" :min="0.1" :max="1.0" :step="0.05" />
                    <span class="slider-row__value">{{ runtimeForm.compactionTriggerRatio }}</span>
                  </div>
                </div>
              </div>
              <div class="settings-form-row">
                <div class="settings-field settings-field--half">
                  <span class="settings-field__label">{{ $t('settings.compactionCutTokens') }}</span>
                  <div class="slider-row">
                    <DqSlider v-model="runtimeForm.compactionCutTokens" :min="1000" :max="64000" :step="1000" />
                    <span class="slider-row__value">{{ runtimeForm.compactionCutTokens }}</span>
                  </div>
                </div>
                <div class="settings-field settings-field--half">
                  <span class="settings-field__label">{{ $t('settings.compactionToolTruncate') }}</span>
                  <div class="slider-row">
                    <DqSlider v-model="runtimeForm.compactionToolTruncate" :min="500" :max="8000" :step="500" />
                    <span class="slider-row__value">{{ runtimeForm.compactionToolTruncate }}</span>
                  </div>
                </div>
              </div>
              <div class="settings-form-row">
                <div class="settings-field settings-field--half">
                  <span class="settings-field__label">{{ $t('settings.compactionTurnInterval') }}</span>
                  <div class="slider-row">
                    <DqSlider v-model="runtimeForm.compactionTurnInterval" :min="1" :max="50" :step="1" />
                    <span class="slider-row__value">{{ runtimeForm.compactionTurnInterval }}</span>
                  </div>
                </div>
                <div class="settings-field settings-field--half">
                  <span class="settings-field__label">{{ $t('settings.compactionSubInterval') }}</span>
                  <div class="slider-row">
                    <DqSlider v-model="runtimeForm.compactionSubInterval" :min="1" :max="50" :step="1" />
                    <span class="slider-row__value">{{ runtimeForm.compactionSubInterval }}</span>
                  </div>
                </div>
              </div>
            </template>
          </div>

        </div>
      </div>

      <div v-else-if="activeTab === 'models'" class="settings-section settings-section--wide">
        <header class="settings-section__head">
          <h2>{{ $t('settings.models') }}</h2>
          <p>{{ $t('settings.modelsDesc') }}</p>
        </header>

        <div v-if="llm.loading" class="settings-empty">{{ $t('common.loading') }}</div>

        <div v-else>
          <div v-if="llm.configs.length" class="provider-list">
            <div v-for="cfg in llm.configs" :key="cfg.id" class="provider-card">
              <div class="provider-card__head">
                <div class="provider-card__info">
                  <span class="provider-card__name">{{ cfg.name }}</span>
                  <span class="provider-card__type">{{ providerLabel(cfg.provider) }}</span>
                </div>
                <div class="provider-card__actions">
                  <DqButton size="small" @click="openEditForm(cfg)">{{ $t('settings.edit') }}</DqButton>
                  <DqButton size="small" type="danger" @click="handleDelete(cfg.id)">{{ $t('common.delete') }}</DqButton>
                </div>
              </div>
              <div class="provider-card__models">
                <div
                  v-for="model in cfg.models ?? []"
                  :key="model.name"
                  class="provider-card__model"
                  :class="{ 'is-disabled': !model.enabled }"
                >
                  <span class="provider-card__model-name">{{ model.name }}</span>
                  <span v-if="!model.enabled" class="provider-card__model-status">{{ $t('settings.disabled') }}</span>
                </div>
                <div v-if="!(cfg.models && cfg.models.length)" class="provider-card__models-empty">
                  {{ $t('settings.noModels') }}
                </div>
              </div>
            </div>
          </div>

          <div v-if="!llm.configs.length" class="settings-empty">
            {{ $t('settings.noProvider') }}
          </div>
        </div>
      </div>

      <div v-else-if="activeTab === 'search'" class="settings-section">
        <header class="settings-section__head">
          <h2>{{ $t('settings.webSearch') }}</h2>
          <p>{{ $t('settings.searchDesc') }}</p>
        </header>

        <div v-if="searchConfig.loading" class="settings-empty">{{ $t('common.loading') }}</div>

        <div v-else class="settings-form">
          <label class="settings-field">
            <span class="settings-field__label">{{ $t('settings.searchProvider') }}</span>
            <DqSelect v-model="searchForm.provider">
              <DqOption
                v-for="opt in searchConfig.providerOptions"
                :key="opt.value"
                :value="opt.value"
                :label="opt.label"
              />
            </DqSelect>
          </label>

          <label class="settings-field">
            <span class="settings-field__label">{{ $t('settings.baseUrl') }}</span>
            <DqInput v-model="searchForm.baseUrl" :placeholder="$t('settings.searchPlaceholder')" />
          </label>

          <label class="settings-field">
            <span class="settings-field__label">{{ $t('settings.apiKey') }}</span>
            <DqInput v-model="searchForm.apiKey" type="password" :placeholder="$t('settings.apiKeyOptional')" />
          </label>

          <div class="settings-form-row">
            <div class="settings-field settings-field--half">
              <span class="settings-field__label">{{ $t('settings.timeout') }}</span>
              <div class="slider-row">
                <DqSlider v-model="searchForm.timeoutMs" :min="1000" :max="60000" :step="1000" />
                <span class="slider-row__value">{{ searchForm.timeoutMs }}ms</span>
              </div>
            </div>
            <div class="settings-field settings-field--half">
              <span class="settings-field__label">{{ $t('settings.maxResults') }}</span>
              <div class="slider-row">
                <DqSlider v-model="searchForm.maxResults" :min="1" :max="10" :step="1" />
                <span class="slider-row__value">{{ searchForm.maxResults }}</span>
              </div>
            </div>
          </div>

        </div>
      </div>

      <div v-else-if="activeTab === 'modelLimits'" class="settings-section settings-section--wide">
        <header class="settings-section__head">
          <h2>{{ $t('settings.modelLimits') }}</h2>
          <p>{{ $t('settings.modelLimitsDesc') }}</p>
        </header>

        <div v-if="modelLimits.loading" class="settings-empty">{{ $t('common.loading') }}</div>

        <div v-else>
          <div v-if="modelLimitsForm.length" class="model-limit-list">
            <div class="model-limit-list__head">
              <span class="model-limit-list__col">{{ $t('settings.modelName') }}</span>
              <span class="model-limit-list__col">{{ $t('settings.contextWindow') }}</span>
              <span class="model-limit-list__col">{{ $t('settings.maxOutput') }}</span>
              <span class="model-limit-list__col model-limit-list__col--actions"></span>
            </div>
            <div v-for="(item, idx) in modelLimitsForm" :key="item.model" class="model-limit-list__row">
              <span class="model-limit-list__col model-limit-list__col--name">{{ item.model }}</span>
              <span class="model-limit-list__col">{{ item.contextWindow.toLocaleString() }}</span>
              <span class="model-limit-list__col">{{ item.maxOutput.toLocaleString() }}</span>
              <span class="model-limit-list__col model-limit-list__col--actions">
                <DqButton size="small" @click="openEditModelLimit(idx)">{{ $t('settings.edit') }}</DqButton>
                <DqButton size="small" type="text" @click="removeModelLimit(idx)">{{ $t('common.delete') }}</DqButton>
              </span>
            </div>
          </div>

          <div v-else class="settings-empty">{{ $t('settings.noModelLimits') }}</div>
        </div>
      </div>

      </div>

      <footer v-if="hasFooterActions" class="settings-panel__footer">
        <span class="settings-panel__footer-hint">{{ footerHint }}</span>
        <div class="settings-panel__footer-actions">
          <DqButton v-if="activeTab === 'runtime'" type="primary" :disabled="runtimeConfig.saving" @click="handleSaveRuntime">
            {{ runtimeConfig.saving ? $t('common.saving') : $t('common.save_') }}
          </DqButton>
          <DqButton v-else-if="activeTab === 'search'" type="primary" :disabled="searchConfig.saving" @click="handleSaveSearch">
            {{ searchConfig.saving ? $t('common.saving') : $t('common.save_') }}
          </DqButton>
          <DqButton v-else-if="activeTab === 'models'" type="primary" @click="openNewForm">{{ $t('settings.addProvider') }}</DqButton>
          <DqButton v-else-if="activeTab === 'modelLimits'" type="primary" @click="openAddModelLimit">{{ $t('settings.addModelLimit') }}</DqButton>
        </div>
      </footer>

      <DqDialog
        v-model:open="showModelLimitForm"
        :title="editingModelLimitIdx !== null ? $t('settings.editModelLimit') : $t('settings.addModelLimit')"
        variant="glass"
        width="480px"
        :closable="true"
      >
        <div class="settings-form">
          <label class="settings-field">
            <span class="settings-field__label">{{ $t('settings.modelName') }}</span>
            <DqInput v-model="modelLimitForm.model" :placeholder="$t('settings.modelNamePlaceholder')" :disabled="editingModelLimitIdx !== null" />
          </label>
          <label class="settings-field">
            <span class="settings-field__label">{{ $t('settings.contextWindow') }}</span>
            <DqInput v-model.number="modelLimitForm.contextWindow" type="number" placeholder="128000" />
          </label>
          <label class="settings-field">
            <span class="settings-field__label">{{ $t('settings.maxOutput') }}</span>
            <DqInput v-model.number="modelLimitForm.maxOutput" type="number" placeholder="8192" />
          </label>
          <div class="settings-actions">
            <DqButton type="primary" @click="saveModelLimitForm">{{ $t('common.save_') }}</DqButton>
          </div>
        </div>
      </DqDialog>

      <DqDialog
        v-model:open="showForm"
        :title="editingId ? $t('settings.editProvider') : $t('settings.addProviderTitle')"
        variant="glass"
        :width="dialogStep === 'choose' && !editingId ? '640px' : '600px'"
        :closable="true"
      >
        <!-- Step 1: Choose preset -->
        <div v-if="dialogStep === 'choose' && !editingId" class="preset-grid">
          <p class="preset-grid__hint">{{ $t('settings.chooseProviderHint') }}</p>
          <div class="preset-cards">
            <button
              v-for="preset in llm.presets"
              :key="preset.id"
              type="button"
              class="preset-card"
              @click="selectPreset(preset)"
            >
              <span class="preset-card__badge" :style="{ background: presetColor(preset.id) }">{{ presetAbbr(preset.id) }}</span>
              <span class="preset-card__name">{{ preset.name }}</span>
              <span class="preset-card__desc">{{ preset.description }}</span>
            </button>
            <button type="button" class="preset-card preset-card--custom" @click="selectCustom">
              <span class="preset-card__badge preset-card__badge--custom">{{ $t('settings.customProvider')[0] }}</span>
              <span class="preset-card__name">{{ $t('settings.customProvider') }}</span>
              <span class="preset-card__desc">{{ $t('settings.customProviderDesc') }}</span>
            </button>
          </div>
        </div>

        <!-- Step 2: Configure -->
        <div v-else class="settings-form">
          <div v-if="!editingId" class="settings-form__back">
            <button type="button" class="settings-form__back-btn" @click="backToChoose">
              ← {{ $t('settings.chooseProvider') }}
            </button>
          </div>

          <label v-if="!editingId" class="settings-field">
            <span class="settings-field__label">{{ $t('settings.protocolType') }}</span>
            <DqSelect v-model="form.provider">
              <DqOption
                v-for="opt in providerOptions"
                :key="opt.value"
                :value="opt.value"
                :label="opt.label"
              />
            </DqSelect>
          </label>

          <label class="settings-field">
            <span class="settings-field__label">{{ $t('common.name') }}</span>
            <DqInput v-model="form.name" :placeholder="$t('settings.nameExample')" :disabled="!!editingId" />
          </label>

          <label class="settings-field">
            <span class="settings-field__label">{{ $t('settings.apiKey') }}</span>
            <DqInput v-model="form.apiKey" type="password" :placeholder="$t('settings.apiKeyPlaceholder')" />
          </label>

          <label class="settings-field">
            <span class="settings-field__label">{{ $t('settings.baseUrl') }}</span>
            <DqInput v-model="form.baseUrl" :placeholder="$t('settings.baseUrlPlaceholder')" />
          </label>

          <div class="settings-field settings-field--toolbar">
            <div class="settings-field__toolbar">
              <DqButton size="small" :disabled="refreshingModels" @click="handleRefreshModels">
                {{ refreshingModels ? $t('common.refreshing') : $t('settings.refreshModels') }}
              </DqButton>
              <span class="settings-field__hint">{{ $t('settings.refreshHint') }}</span>
            </div>
          </div>

          <div v-if="displayedModels.length" class="model-list">
            <div class="model-list__head">
              <span class="model-list__title">{{ $t('settings.modelList') }}</span>
              <span class="model-list__hint">{{ $t('settings.modelToggleHint') }}</span>
            </div>
            <div class="model-list__items">
              <div
                v-for="m in displayedModels"
                :key="m.name"
                class="model-list__item"
                :class="{ 'is-disabled': !m.enabled }"
              >
                <span class="model-list__name">{{ m.name }}</span>
                <div class="model-list__actions">
                  <DqSwitch
                    :model-value="m.enabled"
                    size="small"
                    @update:model-value="(v: boolean) => handleToggleModel(m.name, v)"
                  />
                </div>
              </div>
            </div>
          </div>
        </div>

        <template #footer>
          <div v-if="dialogStep === 'choose' && !editingId" class="settings-actions">
            <DqButton @click="cancelForm">{{ $t('common.cancel') }}</DqButton>
          </div>
          <div v-else class="settings-actions">
            <DqButton @click="cancelForm">{{ $t('common.cancel') }}</DqButton>
            <DqButton type="primary" :disabled="llm.saving" @click="handleSave">
              {{ llm.saving ? $t('common.saving') : (editingId ? $t('common.update') : $t('common.save_')) }}
            </DqButton>
          </div>
        </template>
      </DqDialog>
    </main>
  </div>
</template>

<style scoped>
.settings-view {
  display: flex;
  height: 100%;
  min-width: 0;
  min-height: 0;
  overflow: hidden;
  background: var(--teams-glass-bg);
}

/* ── Form control consistency ── */

/* Make DqInput visible with solid bg + clear border */
.settings-view :deep(.dq-input) {
  background: var(--dq-glass-control-bg-solid);
  border-color: var(--teams-glass-border);
  height: 34px;
  min-height: 34px;
  font-size: var(--dq-font-size-body);
}

.settings-view :deep(.dq-input:hover:not(:disabled):not(:focus):not(:focus-visible)) {
  border-color: color-mix(in srgb, var(--dq-label-primary) 22%, transparent);
}

.settings-view :deep(.dq-input:focus),
.settings-view :deep(.dq-input:focus-visible) {
  background: var(--dq-bg-elevated);
  border-color: var(--dq-accent);
}

.settings-view :deep(.dq-input:disabled) {
  opacity: 0.5;
}

.settings-sidebar {
  flex-shrink: 0;
  width: 200px;
  border-right: 1px solid var(--teams-glass-border);
  background: var(--teams-glass-bg);
  display: flex;
  flex-direction: column;
  padding: 16px 12px;
}

.settings-sidebar__head {
  padding: 0 8px 16px;
}

.settings-sidebar__title {
  font-size: var(--dq-font-size-body);
  font-weight: 600;
  color: var(--dq-label-tertiary);
}

.settings-sidebar__menu {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.settings-sidebar__item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 8px 10px;
  border: none;
  border-radius: 8px;
  background: transparent;
  color: var(--dq-label-primary);
  font-size: var(--dq-font-size-body);
  font-weight: 500;
  cursor: pointer;
  text-align: left;
  transition: background 0.12s ease, color 0.12s ease;
}

.settings-sidebar__item:hover {
  background: color-mix(in srgb, var(--dq-label-primary) 5%, transparent);
}

.settings-sidebar__item.is-active {
  background: color-mix(in srgb, var(--dq-accent) 12%, var(--dq-fill-tertiary));
  color: var(--dq-accent);
}

.settings-panel {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

.settings-panel__content {
  flex: 1;
  min-height: 0;
  overflow: auto;
  padding: 28px 36px 20px;
  background: color-mix(in srgb, var(--dq-bg-elevated) 30%, transparent);
}

.settings-panel__footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 12px 24px;
  border-top: 1px solid var(--dq-separator-light);
  background: color-mix(in srgb, var(--dq-bg-elevated) 50%, transparent);
  flex-shrink: 0;
}

.settings-panel__footer-hint {
  font-size: var(--dq-font-size-caption);
  color: var(--dq-label-quaternary);
}

.settings-panel__footer-actions {
  display: flex;
  align-items: center;
  gap: 8px;
}

.settings-section {
  max-width: 640px;
}

.settings-section--wide {
  max-width: 720px;
}

.settings-section__head {
  margin-bottom: 24px;
}

.settings-section__head h2 {
  margin: 0 0 6px;
  font-size: var(--dq-font-size-heading);
  font-weight: 600;
  color: var(--dq-label-primary);
}

.settings-section__head p {
  margin: 0;
  font-size: var(--dq-font-size-body);
  color: var(--dq-label-tertiary);
}

.settings-form {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.settings-form-group {
  padding: 0;
  border-radius: 0;
  border: none;
  background: transparent;
}

.settings-form-group + .settings-form-group {
  margin-top: 24px;
}

.settings-form-group__title {
  margin: 0 0 4px;
  font-size: var(--dq-font-size-secondary);
  font-weight: 600;
  color: var(--dq-label-primary);
}

.settings-form-group__desc {
  margin: 0 0 16px;
  font-size: var(--dq-font-size-footnote);
  color: var(--dq-label-tertiary);
  line-height: 1.5;
}

.settings-form-row {
  display: flex;
  gap: 16px;
}

.settings-form-row + .settings-form-row {
  margin-top: 12px;
}

.settings-field--switch + .settings-form-row {
  margin-top: 12px;
}

.settings-field--half {
  flex: 1;
  min-width: 0;
}

.settings-field {
  display: flex;
  flex-direction: column;
  gap: 5px;
}

.settings-field--switch {
  flex-direction: row;
  align-items: center;
  justify-content: space-between;
}

.settings-field--block {
  width: 100%;
}

.settings-field--inline {
  flex-direction: row;
  align-items: center;
  gap: 12px;
}

.settings-field--toolbar {
  padding: 10px 12px;
  border-radius: 10px;
  background: color-mix(in srgb, var(--dq-label-primary) 3%, transparent);
  border: 1px solid var(--teams-glass-border);
}

.settings-field__toolbar {
  display: flex;
  align-items: center;
  gap: 12px;
  flex-wrap: wrap;
}

.settings-field__hint {
  font-size: var(--dq-font-size-footnote);
  color: var(--dq-label-tertiary);
  line-height: 1.4;
}

.settings-field__label {
  font-size: var(--dq-font-size-body);
  font-weight: 500;
  color: var(--dq-label-primary);
}

.model-list {
  display: flex;
  flex-direction: column;
  gap: 10px;
  padding: 14px;
  border-radius: 12px;
  background: var(--dq-bg-elevated);
  border: 1px solid var(--teams-glass-border);
}

.model-list__head {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.model-list__title {
  font-size: var(--dq-font-size-secondary);
  font-weight: 600;
  color: var(--dq-label-primary);
}

.model-list__hint {
  font-size: var(--dq-font-size-footnote);
  color: var(--dq-label-tertiary);
  line-height: 1.4;
}

.model-list__items {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.model-list__item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 8px 10px;
  border-radius: 8px;
  border: 1px solid transparent;
  background: color-mix(in srgb, var(--dq-label-primary) 3%, transparent);
  transition: background 0.12s ease, border-color 0.12s ease, opacity 0.12s ease;
}

.model-list__item:hover {
  background: color-mix(in srgb, var(--dq-label-primary) 6%, transparent);
  border-color: var(--teams-glass-border);
}

.model-list__item.is-disabled {
  opacity: 0.55;
}

.model-list__name {
  flex: 1;
  min-width: 0;
  color: var(--dq-label-primary);
  font-size: var(--dq-font-size-body);
  font-weight: 500;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.model-list__actions {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-shrink: 0;
}

.settings-actions {
  display: flex;
  justify-content: flex-end;
  gap: 10px;
  margin-top: 8px;
}

.settings-empty {
  padding: 20px;
  font-size: var(--dq-font-size-body);
  color: var(--dq-label-tertiary);
  text-align: center;
  background: color-mix(in srgb, var(--dq-label-primary) 4%, transparent);
  border-radius: 10px;
}

.provider-list-actions {
  margin-bottom: 16px;
}

.provider-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.provider-card {
  border-radius: 0;
  border: none;
  background: transparent;
  overflow: visible;
  padding: 16px 0;
  border-bottom: 1px solid var(--teams-glass-border);
}

.provider-card:last-child {
  border-bottom: none;
}

.provider-card__head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0;
  border-bottom: none;
}

.provider-card__info {
  display: flex;
  align-items: center;
  gap: 10px;
}

.provider-card__name {
  font-size: var(--dq-font-size-secondary);
  font-weight: 600;
  color: var(--dq-label-primary);
}

.provider-card__type {
  font-size: var(--dq-font-size-footnote);
  color: var(--dq-label-tertiary);
}

.provider-card__actions {
  display: flex;
  gap: 6px;
}

.provider-card__models {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  padding: 12px 0 0;
}

.provider-card__model {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 4px 10px;
  border-radius: 6px;
  background: color-mix(in srgb, var(--dq-label-primary) 6%, transparent);
  border: 1px solid var(--teams-glass-border);
  font-size: var(--dq-font-size-footnote);
}

.provider-card__model.is-disabled {
  opacity: 0.5;
}

.provider-card__model-name {
  color: var(--dq-label-primary);
  font-weight: 500;
}

.provider-card__model-status {
  font-size: var(--dq-font-size-caption);
  color: var(--dq-label-quaternary);
}

.provider-card__models-empty {
  padding: 8px 0;
  font-size: var(--dq-font-size-footnote);
  color: var(--dq-label-tertiary);
  width: 100%;
}

.preset-grid {
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.preset-grid__hint {
  margin: 0;
  font-size: var(--dq-font-size-body);
  color: var(--dq-label-tertiary);
}

.preset-cards {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 10px;
}

.preset-card {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 6px;
  padding: 16px 10px;
  border-radius: 12px;
  border: 1px solid var(--teams-glass-border);
  background: var(--dq-bg-elevated);
  cursor: pointer;
  transition: border-color 0.12s ease, background 0.12s ease, box-shadow 0.12s ease;
  text-align: center;
}

.preset-card:hover {
  border-color: var(--dq-accent);
  background: color-mix(in srgb, var(--dq-accent) 4%, var(--dq-bg-elevated));
  box-shadow: 0 0 0 3px color-mix(in srgb, var(--dq-accent) 10%, transparent);
}

.preset-card__badge {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  border-radius: 8px;
  font-size: var(--dq-font-size-footnote);
  font-weight: 700;
  color: var(--dq-color-white);
  letter-spacing: -0.5px;
}

.preset-card__badge--custom {
  background: var(--dq-label-tertiary);
}

.preset-card__name {
  font-size: var(--dq-font-size-body);
  font-weight: 600;
  color: var(--dq-label-primary);
}

.preset-card__desc {
  font-size: var(--dq-font-size-caption);
  color: var(--dq-label-tertiary);
  line-height: 1.3;
}

.preset-card--custom {
  border-style: dashed;
}

.preset-card--custom:hover {
  border-style: solid;
}

.settings-form__back {
  margin-bottom: 4px;
}

.settings-form__back-btn {
  border: none;
  background: none;
  padding: 4px 0;
  font-size: var(--dq-font-size-body);
  color: var(--dq-accent);
  cursor: pointer;
}

.settings-form__back-btn:hover {
  opacity: 0.8;
}

.slider-row {
  display: flex;
  align-items: center;
  gap: 10px;
  height: 36px;
}

.slider-row :deep(.dq-slider) {
  flex: 1;
  min-width: 0;
}

.slider-row :deep(.dq-slider__track) {
  display: flex;
  align-items: center;
}

.slider-row__value {
  flex-shrink: 0;
  min-width: 36px;
  text-align: right;
  font-size: var(--dq-font-size-footnote);
  font-weight: 500;
  font-family: ui-monospace, SFMono-Regular, 'SF Mono', Menlo, monospace;
  color: var(--dq-label-secondary);
}

/* Model limits list */
.model-limit-list {
  margin-top: 12px;
  border: 1px solid var(--teams-glass-border);
  border-radius: 10px;
  overflow: hidden;
}

.model-limit-list__head {
  display: flex;
  align-items: center;
  padding: 8px 12px;
  background: color-mix(in srgb, var(--dq-label-primary) 4%, transparent);
  border-bottom: 1px solid var(--teams-glass-border);
  font-size: var(--dq-font-size-footnote);
  font-weight: 600;
  color: var(--dq-label-secondary);
}

.model-limit-list__row {
  display: flex;
  align-items: center;
  padding: 10px 12px;
  border-bottom: 1px solid color-mix(in srgb, var(--dq-label-primary) 5%, transparent);
  font-size: var(--dq-font-size-body);
}

.model-limit-list__row:last-child {
  border-bottom: none;
}

.model-limit-list__col {
  flex: 1;
  min-width: 0;
}

.model-limit-list__col--name {
  flex: 2;
  font-weight: 500;
  font-family: ui-monospace, SFMono-Regular, 'SF Mono', Menlo, monospace;
  font-size: var(--dq-font-size-footnote);
}

.model-limit-list__col--actions {
  flex: 0 0 auto;
  display: flex;
  gap: 6px;
  justify-content: flex-end;
}

/* Provider card border visibility */
.provider-card {
  border-color: color-mix(in srgb, var(--dq-label-primary) 10%, transparent);
}

/* Theme grid */
.theme-grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 14px;
}

.theme-card {
  position: relative;
  display: flex;
  flex-direction: column;
  padding: 16px;
  border-radius: 14px;
  border: 1.5px solid var(--teams-glass-border);
  background: var(--dq-bg-elevated);
  cursor: pointer;
  text-align: left;
  transition: border-color 0.15s ease, background 0.15s ease, box-shadow 0.15s ease, transform 0.12s ease;
}

.theme-card:hover {
  border-color: color-mix(in srgb, var(--dq-label-primary) 18%, transparent);
  background: color-mix(in srgb, var(--dq-label-primary) 4%, var(--dq-bg-elevated));
}

.theme-card.is-active {
  border-color: var(--dq-accent);
  box-shadow: 0 0 0 3px color-mix(in srgb, var(--dq-accent) 15%, transparent);
}

.theme-card__preview {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 14px;
  padding: 12px;
  border-radius: 10px;
  background: color-mix(in srgb, var(--dq-label-primary) 4%, transparent);
}

.theme-card__swatch {
  width: 32px;
  height: 32px;
  border-radius: 8px;
  flex-shrink: 0;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.2);
}

.theme-card__bars {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 5px;
}

.theme-card__bar {
  display: block;
  height: 6px;
  border-radius: 3px;
  width: 100%;
}

.theme-card__bar--short {
  width: 60%;
}

.theme-card__info {
  display: flex;
  flex-direction: column;
  gap: 3px;
}

.theme-card__name {
  font-size: var(--dq-font-size-body);
  font-weight: 600;
  color: var(--dq-label-primary);
}

.theme-card__desc {
  font-size: var(--dq-font-size-caption);
  color: var(--dq-label-tertiary);
  line-height: 1.35;
}

.theme-card__check {
  position: absolute;
  top: 12px;
  right: 12px;
  width: 22px;
  height: 22px;
  border-radius: 50%;
  background: var(--dq-accent);
  color: var(--dq-color-white);
  display: flex;
  align-items: center;
  justify-content: center;
}
</style>
