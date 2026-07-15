<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import WorkspaceShell from '@/components/common/WorkspaceShell.vue'
import { useMcpServersStore } from '@/stores/mcpServers'
import { confirm, toast } from '@/utils/feedback'
import type { MCPServer, MCPToolDef } from '@/types'

type Transport = 'stdio' | 'sse' | 'streamable-http'

const { t } = useI18n()
const mcp = useMcpServersStore()

const selectedId = ref<string | null>(null)
const isCreating = ref(false)
const saving = ref(false)
const refreshingTools = ref(false)

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
  status: 'disconnected',
  enabled: true,
})

/** Editable text for headers (KEY=VALUE per line) */
const headersText = computed({
  get() {
    const h = form.value.headers
    if (!h || Object.keys(h).length === 0) return ''
    return Object.entries(h).map(([k, v]) => `${k}=${v}`).join('\n')
  },
  set(val: string) {
    const map: Record<string, string> = {}
    for (const line of val.split('\n')) {
      const trimmed = line.trim()
      if (!trimmed) continue
      const eqIdx = trimmed.indexOf('=')
      if (eqIdx > 0) {
        map[trimmed.slice(0, eqIdx).trim()] = trimmed.slice(eqIdx + 1).trim()
      }
    }
    form.value.headers = map
  },
})

/** Discovered tools list from the selected server */
const discoveredTools = computed<MCPToolDef[]>(() => {
  if (!selected.value?.discoveredTools) return []
  return selected.value.discoveredTools
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

onMounted(async () => {
  await mcp.load()
  if (sortedServers.value.length && !selectedId.value) {
    selectServer(sortedServers.value[0].id)
  }
})

function selectServer(id: string) {
  isCreating.value = false
  selectedId.value = id
  const server = mcp.items.find((s) => s.id === id)
  if (server) form.value = { ...server }
}

function openCreate() {
  isCreating.value = true
  selectedId.value = null
  form.value = {
    id: '',
    name: '',
    description: '',
    transport: 'stdio',
    command: '',
    args: '',
    url: '',
    env: '',
    status: 'disconnected',
    enabled: true,
  }
}

async function save() {
  if (!form.value.name.trim()) {
    toast.warning(t('mcpServers.namePlaceholder'))
    return
  }
  saving.value = true
  try {
    if (isCreating.value) {
      const server = await mcp.create({ ...form.value, name: form.value.name.trim() })
      toast.success(t('mcpServers.created'))
      isCreating.value = false
      selectServer(server.id)
    } else if (selected.value) {
      await mcp.update(selected.value.id, { ...form.value })
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
  await mcp.remove(selected.value.id)
  selectedId.value = null
  isCreating.value = false
  toast.success(t('mcpServers.deleted'))
}

async function toggleEnabled() {
  if (!selected.value) return
  const next = !selected.value.enabled
  await mcp.update(selected.value.id, { enabled: next })
  selectServer(selected.value.id)
}

async function handleRefreshTools() {
  if (!selected.value || refreshingTools.value) return
  refreshingTools.value = true
  try {
    await mcp.refreshTools(selected.value.id)
    selectServer(selected.value.id)
    toast.success(t('mcpServers.toolsRefreshed'))
  } catch (e) {
    toast.error(e instanceof Error ? e.message : t('mcpServers.refreshToolsFailed'))
  } finally {
    refreshingTools.value = false
  }
}

async function handleToggleTool(toolName: string, enabled: boolean) {
  if (!selected.value) return
  try {
    await mcp.toggleTool(selected.value.id, toolName, enabled)
    selectServer(selected.value.id)
  } catch (e) {
    toast.error(e instanceof Error ? e.message : t('common.saveFailed'))
  }
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
    </template>

    <template #body>
      <section class="resource-section">
        <div class="resource-form-grid resource-form-grid--2">
          <label class="resource-field">
            <span class="resource-field__label">{{ $t('common.name') }}</span>
            <DqInput v-model="form.name" placeholder="Prometheus MCP" />
          </label>
          <label class="resource-field">
            <span class="resource-field__label">{{ $t('mcpServers.transport') }}</span>
            <DqSelect v-model="form.transport" :placeholder="$t('mcpServers.transport')">
              <DqOption v-for="opt in transportOptions" :key="opt.value" :value="opt.value" :label="opt.label" />
            </DqSelect>
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
        <label v-if="form.transport !== 'stdio'" class="resource-field resource-field--block">
          <span class="resource-field__label">{{ $t('mcpServers.headers') }}</span>
          <DqInput v-model="headersText" class="resource-input-mono" type="textarea" :rows="3" :placeholder="$t('mcpServers.headersPlaceholder')" />
        </label>
        <!-- Discovered Tools -->
        <div class="resource-section__tools">
          <div class="resource-section__tools-header">
            <span class="resource-field__label">{{ $t('mcpServers.discoveredTools') }}</span>
            <DqButton size="small" :disabled="refreshingTools" @click="handleRefreshTools">
              {{ refreshingTools ? $t('common.refreshing') : $t('mcpServers.refreshTools') }}
            </DqButton>
          </div>
          <div v-if="discoveredTools.length === 0" class="resource-section__tools-empty">
            {{ $t('mcpServers.noToolsDiscovered') }}
          </div>
          <div v-else class="resource-section__tools-list">
            <label v-for="tool in discoveredTools" :key="tool.name" class="resource-tool-row">
              <DqSwitch :model-value="tool.enabled" size="small" @update:model-value="(v: boolean) => handleToggleTool(tool.name, v)" />
              <span class="resource-tool-row__name">{{ tool.name }}</span>
              <span v-if="tool.description" class="resource-tool-row__desc">{{ tool.description }}</span>
            </label>
          </div>
        </div>
        <div v-if="!isCreating" class="resource-form-grid resource-form-grid--2">
          <label class="resource-field resource-field--toggle">
            <span class="resource-field__label">{{ $t('mcpServers.enabled') }}</span>
            <DqSwitch
              :model-value="form.enabled"
              size="small"
              @update:model-value="(v: boolean) => form.enabled = v"
            />
          </label>
        </div>
      </section>
    </template>

    <template #footer>
      <span class="resource-workspace__hint">{{ $t('common.saveShortcut') }}</span>
      <div class="resource-workspace__footer-actions">
        <DqButton v-if="isCreating" @click="isCreating = false; selectedId = null">{{ $t('common.cancel') }}</DqButton>
        <DqButton v-if="!isCreating" @click="toggleEnabled">
          {{ selected?.enabled ? $t('mcpServers.disable') : $t('mcpServers.enable') }}
        </DqButton>
        <DqButton v-if="!isCreating" @click="removeSelected">{{ $t('common.delete') }}</DqButton>
        <DqButton type="primary" :disabled="saving" @click="save">
          {{ isCreating ? $t('mcpServers.createServer') : $t('common.save') }}
        </DqButton>
      </div>
    </template>
  </WorkspaceShell>
</template>
