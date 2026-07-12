<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import WorkspaceShell from '@/components/common/WorkspaceShell.vue'
import { useMcpServersStore } from '@/stores/mcpServers'
import { confirm, toast } from '@/utils/feedback'
import type { MCPServer, Tool } from '@/types'

type Transport = 'stdio' | 'sse' | 'streamable-http'

const { t } = useI18n()
const mcp = useMcpServersStore()

const selectedId = ref<string | null>(null)
const isCreating = ref(false)
const saving = ref(false)
const activeTab = ref<'info' | 'tools'>('info')

const transportOptions: { value: Transport; label: string }[] = [
  { value: 'stdio', label: 'STDIO' },
  { value: 'sse', label: 'SSE' },
  { value: 'streamable-http', label: 'Streamable HTTP' },
]

const form = ref<MCPServer>({
  id: '',
  name: '',
  description: '',
  transport: 'stdio',
  command: '',
  args: '',
  url: '',
  env: '',
  tools: [],
  status: 'disconnected',
})

const pendingTool = ref<Omit<Tool, 'id'>>({
  name: '',
  description: '',
  type: 'mcp',
  riskLevel: 'low',
  schema: '',
})

const sortedServers = computed(() =>
  [...mcp.items].sort((a, b) => a.name.localeCompare(b.name, 'zh-CN')),
)
const selected = computed(() => mcp.items.find((s) => s.id === selectedId.value))
const hasSelection = computed(() => isCreating.value || !!selectedId.value)
const headerTitle = computed(() => {
  if (isCreating.value) return form.value.name.trim() || t('mcpServers.newServer')
  return selected.value?.name.trim() || t('mcpServers.untitled')
})

onMounted(() => {
  if (sortedServers.value.length && !selectedId.value) {
    selectServer(sortedServers.value[0].id)
  }
})

function selectServer(id: string) {
  isCreating.value = false
  selectedId.value = id
  activeTab.value = 'info'
  const server = mcp.items.find((s) => s.id === id)
  if (server) form.value = { ...server }
}

function openCreate() {
  isCreating.value = true
  selectedId.value = null
  activeTab.value = 'info'
  form.value = {
    id: '',
    name: '',
    description: '',
    transport: 'stdio',
    command: '',
    args: '',
    url: '',
    env: '',
    tools: [],
    status: 'disconnected',
  }
}

function save() {
  if (!form.value.name.trim()) {
    toast.warning(t('mcpServers.namePlaceholder'))
    return
  }
  saving.value = true
  try {
    if (isCreating.value) {
      const server = mcp.create({ ...form.value, name: form.value.name.trim() })
      toast.success(t('mcpServers.created'))
      isCreating.value = false
      selectServer(server.id)
    } else if (selected.value) {
      mcp.update(selected.value.id, { ...form.value })
      toast.success(t('mcpServers.saved'))
      selectServer(selected.value.id)
    }
  } catch (e) {
    toast.error(e instanceof Error ? e.message : t('common.saveFailed'))
  } finally {
    saving.value = false
  }
}

async function removeSelected() {
  if (!selected.value) return
  try {
    await confirm(t('mcpServers.deleteConfirm', { name: selected.value.name }), t('mcpServers.deleteTitle'), { type: 'warning' })
  } catch {
    return
  }
  mcp.remove(selected.value.id)
  selectedId.value = null
  isCreating.value = false
  toast.success(t('mcpServers.deleted'))
}

function addTool() {
  if (!selected.value || !pendingTool.value.name.trim() || !pendingTool.value.description?.trim()) return
  mcp.addTool(selected.value.id, { ...pendingTool.value, name: pendingTool.value.name.trim() })
  pendingTool.value = { name: '', description: '', type: 'mcp', riskLevel: 'low', schema: '' }
  toast.success(t('mcpServers.toolAdded'))
}

function removeTool(toolId: string) {
  if (!selected.value) return
  mcp.removeTool(selected.value.id, toolId)
  toast.success(t('mcpServers.toolDeleted'))
}

function toggleStatus() {
  if (!selected.value) return
  const next = selected.value.status === 'connected' ? 'disconnected' : 'connected'
  mcp.update(selected.value.id, { status: next })
  toast.success(next === 'connected' ? t('mcpServers.connected') : t('mcpServers.disconnected'))
  selectServer(selected.value.id)
}

function initial(name: string) {
  return name.trim().charAt(0).toUpperCase() || 'M'
}

function onKeydown(e: KeyboardEvent) {
  if ((e.metaKey || e.ctrlKey) && e.key === 's') {
    e.preventDefault()
    save()
  }
}
</script>

<template>
  <WorkspaceShell
    :title="$t('mcpServers.title')"
    :count="sortedServers.length"
    count-label="MCP Servers"
    :create-label="$t('mcpServers.newServer')"
    :has-selection="hasSelection"
    @create="openCreate"
    @keydown="onKeydown"
  >
    <template #rail>
      <DqEmpty v-if="!sortedServers.length" class="resource-rail__empty" :description="$t('mcpServers.noServers')" />
      <nav v-else class="resource-rail__list" :aria-label="$t('mcpServers.serverList')">
        <button
          v-for="server in sortedServers"
          :key="server.id"
          type="button"
          class="resource-rail__row"
          :class="{ 'is-active': selectedId === server.id && !isCreating }"
          @click="selectServer(server.id)"
        >
          <span class="resource-rail__avatar">{{ initial(server.name) }}</span>
          <span class="resource-rail__meta">
            <span class="resource-rail__name">{{ server.name }}</span>
            <span class="resource-rail__desc">{{ server.transport }}</span>
          </span>
          <span
            class="resource-rail__tag"
            :class="server.status === 'connected' ? 'is-accent' : ''"
          >
            {{ server.status === 'connected' ? $t('mcpServers.connected') : $t('mcpServers.notConnected') }}
          </span>
        </button>
      </nav>
    </template>

    <template #empty>
      <DqEmpty :description="$t('mcpServers.emptySelection')">
        <p class="resource-workspace__hint">{{ $t('mcpServers.emptySelectionHint') }}</p>
      </DqEmpty>
    </template>

    <template #header>
      <div class="resource-workspace__identity">
        <h1 class="resource-workspace__title">{{ headerTitle }}</h1>
        <div v-if="!isCreating && selected" class="resource-workspace__badges">
          <span class="resource-status" :class="`resource-status--${selected.status}`">
            <span class="resource-status__dot" />
            {{ selected.status === 'connected' ? $t('mcpServers.connected') : selected.status === 'error' ? $t('mcpServers.error') : $t('mcpServers.disconnected') }}
          </span>
        </div>
      </div>
      <nav v-if="!isCreating" class="resource-workspace__tabs" role="tablist">
        <button
          type="button"
          class="resource-workspace__tab"
          :class="{ 'is-active': activeTab === 'info' }"
          role="tab"
          :aria-selected="activeTab === 'info'"
          @click="activeTab = 'info'"
        >
          {{ $t('mcpServers.connectionConfig') }}
        </button>
        <button
          type="button"
          class="resource-workspace__tab"
          :class="{ 'is-active': activeTab === 'tools' }"
          role="tab"
          :aria-selected="activeTab === 'tools'"
          @click="activeTab = 'tools'"
        >
          {{ $t('common.tools') }}
        </button>
      </nav>
    </template>

    <template #body>
      <section v-show="activeTab === 'info'" class="resource-section">
        <div class="resource-form-grid resource-form-grid--2">
          <label class="resource-field">
            <span class="resource-field__label">{{ $t('common.name') }}</span>
            <DqInput v-model="form.name" placeholder="Prometheus MCP" />
          </label>
          <label class="resource-field">
            <span class="resource-field__label">{{ $t('mcpServers.transport') }}</span>
            <select v-model="form.transport" class="resource-field__select">
              <option v-for="opt in transportOptions" :key="opt.value" :value="opt.value">{{ opt.label }}</option>
            </select>
          </label>
        </div>
        <label class="resource-field resource-field--block">
          <span class="resource-field__label">{{ $t('common.description') }}</span>
          <DqInput v-model="form.description" type="textarea" :rows="3" :placeholder="$t('mcpServers.descriptionPlaceholder')" />
        </label>
        <div v-if="form.transport === 'stdio'" class="resource-form-grid resource-form-grid--2">
          <label class="resource-field">
            <span class="resource-field__label">Command</span>
            <DqInput v-model="form.command" class="resource-input-mono" placeholder="npx" />
          </label>
          <label class="resource-field">
            <span class="resource-field__label">Args</span>
            <DqInput v-model="form.args" class="resource-input-mono" placeholder="-y @modelcontextprotocol/server-memory" />
          </label>
        </div>
        <div v-if="form.transport !== 'stdio'" class="resource-form-grid resource-form-grid--2">
          <label class="resource-field">
            <span class="resource-field__label">URL</span>
            <DqInput v-model="form.url" class="resource-input-mono" placeholder="http://localhost:3000/sse" />
          </label>
        </div>
        <label class="resource-field resource-field--block">
          <span class="resource-field__label">{{ $t('mcpServers.envVars') }}</span>
          <DqInput v-model="form.env" class="resource-input-mono" type="textarea" :rows="4" :placeholder="$t('mcpServers.envVarsPlaceholder')" />
        </label>
      </section>

      <section v-show="activeTab === 'tools'" class="resource-section">
        <div class="resource-form-grid resource-form-grid--2">
          <label class="resource-field">
            <span class="resource-field__label">{{ $t('mcpServers.toolName') }}</span>
            <DqInput v-model="pendingTool.name" placeholder="query" />
          </label>
          <label class="resource-field">
            <span class="resource-field__label">{{ $t('common.riskLevel') }}</span>
            <select v-model="pendingTool.riskLevel" class="resource-field__select">
              <option value="low">Low</option>
              <option value="medium">Medium</option>
              <option value="high">High</option>
            </select>
          </label>
        </div>
        <label class="resource-field resource-field--block">
          <span class="resource-field__label">{{ $t('common.description') }}</span>
          <DqInput v-model="pendingTool.description" type="textarea" :rows="3" :placeholder="$t('mcpServers.toolDescriptionPlaceholder')" />
        </label>
        <label class="resource-field resource-field--block">
          <span class="resource-field__label">{{ $t('common.jsonSchema') }}</span>
          <DqInput v-model="pendingTool.schema" class="resource-input-mono" type="textarea" :rows="5" placeholder="{}" />
        </label>
        <div class="resource-form-grid resource-form-grid--2" style="margin-bottom: var(--space-md);">
          <div />
          <div class="resource-field resource-field--action">
            <DqButton @click="addTool">{{ $t('common.addTool') }}</DqButton>
          </div>
        </div>
        <div class="resource-list-card">
          <div v-for="tool in selected?.tools" :key="tool.id" class="resource-list-card__item">
            <div class="resource-list-card__meta">
              <span class="resource-list-card__name">{{ tool.name }}</span>
              <span class="resource-list-card__desc">{{ tool.description }}</span>
            </div>
            <div class="resource-list-card__actions">
              <button type="button" class="resource-list-card__action resource-list-card__action--danger" @click="removeTool(tool.id)">{{ $t('common.delete') }}</button>
            </div>
          </div>
        </div>
      </section>
    </template>

    <template #footer>
      <span class="resource-workspace__hint">{{ $t('common.saveShortcut') }}</span>
      <div class="resource-workspace__footer-actions">
        <DqButton v-if="isCreating" @click="isCreating = false; selectedId = null">{{ $t('common.cancel') }}</DqButton>
        <DqButton v-if="!isCreating" @click="toggleStatus">
          {{ selected?.status === 'connected' ? $t('mcpServers.disconnect') : $t('mcpServers.connect') }}
        </DqButton>
        <DqButton v-if="!isCreating" @click="removeSelected">{{ $t('common.delete') }}</DqButton>
        <DqButton type="primary" :disabled="saving" @click="save">
          {{ isCreating ? $t('mcpServers.createServer') : $t('common.save') }}
        </DqButton>
      </div>
    </template>
  </WorkspaceShell>
</template>
