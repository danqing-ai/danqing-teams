<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import MdEditor from '@/components/common/MdEditor.vue'
import { useGlobalAgentsStore } from '@/stores/globalAgents'
import { confirm, toast } from '@/utils/feedback'
import type { Agent, AgentRole, CreateAgentPayload } from '@/types'

const props = defineProps<{
  initialAgentId?: string
}>()

const globalAgents = useGlobalAgentsStore()

const selectedId = ref<string | null>(null)
const isCreating = ref(false)
const saving = ref(false)
const activeSection = ref<'prompt' | 'settings'>('prompt')

type AgentForm = CreateAgentPayload & { apiKey: string; allModelsText: string }

const emptyForm = (): AgentForm => ({
  id: '',
  name: '',
  description: '',
  role: 'team-worker',
  apiKey: '',
  allModelsText: '',
  systemPrompt: '',
  minFunctionCallingRounds: 1,
  llm: { url: '', defaultModel: '' },
})

const form = ref<AgentForm>(emptyForm())

const roleOptions: { value: AgentRole; label: string }[] = [
  { value: 'team-worker', label: 'Team-Worker' },
  { value: 'team-controller', label: 'Team-Controller' },
]

const sortedAgents = computed(() =>
  [...globalAgents.items].sort((a, b) => a.name.localeCompare(b.name, 'zh-CN')),
)

const hasSelection = computed(() => isCreating.value || !!selectedId.value)

const headerTitle = computed(() => {
  if (isCreating.value) return form.value.name.trim() || '新建 Agent'
  return form.value.name.trim() || '未命名 Agent'
})

const sectionTabs = [
  { id: 'prompt' as const, label: 'System Prompt' },
  { id: 'settings' as const, label: '配置' },
]

onMounted(async () => {
  if (props.initialAgentId) {
    selectAgent(props.initialAgentId)
  } else if (sortedAgents.value.length) {
    selectAgent(sortedAgents.value[0].id)
  }
  await globalAgents.load()
  if (props.initialAgentId) {
    selectAgent(props.initialAgentId)
  } else if (!selectedId.value && !isCreating.value && sortedAgents.value.length) {
    selectAgent(sortedAgents.value[0].id)
  }
})

watch(
  () => props.initialAgentId,
  (id) => {
    if (id) selectAgent(id)
  },
)

function roleLabel(role: AgentRole) {
  return role === 'team-controller' ? 'Controller' : 'Worker'
}

function agentInitial(name: string) {
  return name.trim().charAt(0).toUpperCase() || '?'
}

function compactId(id: string) {
  if (id.length <= 20) return id
  return `${id.slice(0, 8)}…${id.slice(-4)}`
}

function agentToForm(agent: Agent): AgentForm {
  return {
    id: agent.id,
    name: agent.name,
    description: agent.description,
    role: agent.role,
    apiKey: '',
    allModelsText: (agent.llm.allModels ?? []).join('\n'),
    systemPrompt: agent.systemPrompt ?? '',
    minFunctionCallingRounds: agent.minFunctionCallingRounds || 1,
    llm: {
      url: agent.llm.url ?? '',
      defaultModel: agent.llm.defaultModel ?? '',
    },
  }
}

function selectAgent(id: string) {
  isCreating.value = false
  selectedId.value = id
  const agent = globalAgents.items.find((a) => a.id === id)
  if (agent) form.value = agentToForm(agent)
}

function openCreate() {
  isCreating.value = true
  selectedId.value = null
  activeSection.value = 'settings'
  form.value = emptyForm()
}

function buildPayload(): CreateAgentPayload {
  const allModels = form.value.allModelsText
    .split(/[\n,]/)
    .map((s) => s.trim())
    .filter(Boolean)
  return {
    id: form.value.id.trim(),
    name: form.value.name.trim(),
    description: form.value.description.trim(),
    role: form.value.role,
    systemPrompt: form.value.systemPrompt.trim(),
    minFunctionCallingRounds: form.value.minFunctionCallingRounds || 1,
    llm: {
      url: form.value.llm?.url?.trim() ?? '',
      defaultModel: form.value.llm?.defaultModel?.trim() ?? '',
      allModels,
      ...(form.value.apiKey.trim() ? { apiKey: form.value.apiKey.trim() } : {}),
    },
  }
}

const AGENT_ID_RE = /^[a-zA-Z][a-zA-Z0-9_-]{1,63}$/

async function save() {
  const payload = buildPayload()
  if (!payload.id || !AGENT_ID_RE.test(payload.id)) {
    toast.warning('Agent ID 需以字母开头，仅含字母、数字、_、-')
    activeSection.value = 'settings'
    return
  }
  if (!payload.name) {
    toast.warning('请输入显示名称')
    activeSection.value = 'settings'
    return
  }
  saving.value = true
  try {
    if (isCreating.value) {
      const agent = await globalAgents.create(payload)
      toast.success('Agent 已创建')
      isCreating.value = false
      selectAgent(agent.id)
      activeSection.value = 'prompt'
    } else if (selectedId.value) {
      const { id: _id, ...update } = payload
      await globalAgents.update(selectedId.value, update)
      toast.success('已保存')
      selectAgent(selectedId.value)
    }
  } catch (e) {
    toast.error(e instanceof Error ? e.message : '保存失败')
  } finally {
    saving.value = false
  }
}

async function removeSelected() {
  if (!selectedId.value) return
  const agent = globalAgents.items.find((a) => a.id === selectedId.value)
  if (!agent) return
  try {
    await confirm(`确定删除「${agent.name}」？`, '删除 Agent', { type: 'warning' })
  } catch {
    return
  }
  try {
    await globalAgents.remove(agent.id)
    toast.success('已删除')
    selectedId.value = null
    isCreating.value = false
    if (sortedAgents.value.length) {
      selectAgent(sortedAgents.value[0].id)
    } else {
      form.value = emptyForm()
    }
  } catch (e) {
    toast.error(e instanceof Error ? e.message : '删除失败')
  }
}

function cancelCreate() {
  isCreating.value = false
  if (sortedAgents.value.length) {
    selectAgent(sortedAgents.value[0].id)
  } else {
    form.value = emptyForm()
  }
}

function onWorkspaceKeydown(e: KeyboardEvent) {
  if ((e.metaKey || e.ctrlKey) && e.key === 's') {
    e.preventDefault()
    void save()
  }
}
</script>

<template>
  <div class="agents-shell float-island">
    <aside class="agents-rail">
      <div class="agents-rail__head">
        <span class="agents-rail__count">{{ sortedAgents.length }}</span>
        <DqIconButton aria-label="新建 Agent" @click="openCreate">
          <svg viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M12 5v14M5 12h14" stroke-linecap="round" />
          </svg>
        </DqIconButton>
      </div>

      <DqEmpty v-if="!sortedAgents.length && !isCreating" class="agents-rail__empty" description="暂无 Agent" />

      <nav v-else class="agents-rail__list" aria-label="Agent 列表">
        <button v-if="isCreating" type="button" class="agents-rail__row is-active">
          <span class="agents-rail__avatar agents-rail__avatar--new">+</span>
          <span class="agents-rail__meta">
            <span class="agents-rail__name">新建 Agent</span>
          </span>
        </button>
        <button
          v-for="agent in sortedAgents"
          :key="agent.id"
          type="button"
          class="agents-rail__row"
          :class="{ 'is-active': selectedId === agent.id && !isCreating }"
          @click="selectAgent(agent.id)"
        >
          <span class="agents-rail__avatar">{{ agentInitial(agent.name) }}</span>
          <span class="agents-rail__meta">
            <span class="agents-rail__name">{{ agent.name }}</span>
            <span class="agents-rail__id" :title="agent.id">{{ compactId(agent.id) }}</span>
          </span>
          <span
            class="agents-rail__role"
            :class="agent.role === 'team-controller' ? 'is-controller' : ''"
          >
            {{ roleLabel(agent.role) }}
          </span>
        </button>
      </nav>
    </aside>

    <main class="agents-workspace" @keydown="onWorkspaceKeydown">
      <div v-if="!hasSelection" class="agents-workspace__empty">
        <DqEmpty description="选择或新建 Agent">
          <p class="agents-workspace__empty-hint">左侧选择 Agent，或点击 + 新建</p>
        </DqEmpty>
      </div>

      <template v-else>
        <header class="agents-workspace__bar">
          <div class="agents-workspace__identity">
            <h1 class="agents-workspace__title">{{ headerTitle }}</h1>
            <div v-if="form.id && !isCreating" class="agents-workspace__badges">
              <code class="agents-workspace__id" :title="form.id">{{ compactId(form.id) }}</code>
              <span
                class="agents-workspace__role"
                :class="form.role === 'team-controller' ? 'is-controller' : ''"
              >
                {{ roleLabel(form.role) }}
              </span>
            </div>
          </div>
          <nav class="agents-workspace__tabs" role="tablist">
            <button
              v-for="tab in sectionTabs"
              :key="tab.id"
              type="button"
              class="agents-workspace__tab"
              :class="{ 'is-active': activeSection === tab.id }"
              role="tab"
              :aria-selected="activeSection === tab.id"
              @click="activeSection = tab.id"
            >
              {{ tab.label }}
            </button>
          </nav>
        </header>

        <div class="agents-workspace__scroll">
          <!-- Prompt tab: full width editor -->
          <section v-show="activeSection === 'prompt'" class="agents-section agents-section--prompt">
            <MdEditor
              v-model="form.systemPrompt"
              :rows="22"
              placeholder="编写 System Prompt，支持 Markdown 标题、列表、代码块…"
            />
          </section>

          <!-- Settings tab: grouped fields with size-aware layout -->
          <div v-show="activeSection === 'settings'" class="agents-settings">
            <section class="agents-section">
              <header class="agents-section__head">
                <h2 class="agents-section__title">基本信息</h2>
                <p class="agents-section__desc">Agent 标识与人设摘要，Controller 可见描述字段。</p>
              </header>

              <div class="agents-form-grid agents-form-grid--identity">
                <label class="agents-field agents-field--id">
                  <span class="agents-field__label">Agent ID</span>
                  <DqInput
                    v-model="form.id"
                    class="agents-input-mono"
                    placeholder="alert-analyst"
                    :disabled="!isCreating"
                  />
                  <span v-if="isCreating" class="agents-field__hint">字母开头，创建后不可改</span>
                </label>
                <label class="agents-field agents-field--name">
                  <span class="agents-field__label">显示名称</span>
                  <DqInput v-model="form.name" placeholder="Alert Analyst" />
                </label>
                <label class="agents-field agents-field--role">
                  <span class="agents-field__label">Role</span>
                  <select v-model="form.role" class="agents-field__select">
                    <option v-for="opt in roleOptions" :key="opt.value" :value="opt.value">
                      {{ opt.label }}
                    </option>
                  </select>
                </label>
              </div>

              <label class="agents-field agents-field--block">
                <span class="agents-field__label">描述 / 人设</span>
                <DqInput
                  v-model="form.description"
                  type="textarea"
                  :rows="4"
                  placeholder="Controller 分派时可见的职责与人设摘要"
                />
              </label>
            </section>

            <section class="agents-section">
              <header class="agents-section__head">
                <h2 class="agents-section__title">LLM 连接</h2>
                <p class="agents-section__desc">Endpoint 与鉴权；留空 API Key 表示不修改已有值。</p>
              </header>

              <label class="agents-field agents-field--block">
                <span class="agents-field__label">Endpoint URL</span>
                <DqInput
                  v-model="form.llm!.url"
                  class="agents-input-mono"
                  placeholder="https://api.openai.com/v1"
                />
              </label>

              <label class="agents-field agents-field--block">
                <span class="agents-field__label">API Key</span>
                <DqInput
                  v-model="form.apiKey"
                  class="agents-input-mono"
                  type="password"
                  :placeholder="isCreating ? 'sk-…' : '留空保持不变'"
                />
              </label>

              <div class="agents-form-grid agents-form-grid--llm">
                <label class="agents-field agents-field--model">
                  <span class="agents-field__label">Default Model</span>
                  <DqInput
                    v-model="form.llm!.defaultModel"
                    class="agents-input-mono"
                    placeholder="gpt-4o"
                  />
                </label>
                <label class="agents-field agents-field--rounds">
                  <span class="agents-field__label">Min FC Rounds</span>
                  <DqInput
                    v-model.number="form.minFunctionCallingRounds"
                    type="number"
                    min="1"
                    max="32"
                  />
                </label>
              </div>

              <label class="agents-field agents-field--block">
                <span class="agents-field__label">All Models</span>
                <DqInput
                  v-model="form.allModelsText"
                  class="agents-input-mono"
                  type="textarea"
                  :rows="5"
                  placeholder="gpt-4o&#10;gpt-4o-mini&#10;claude-sonnet-4-20250514"
                />
                <span class="agents-field__hint">每行一个模型 ID</span>
              </label>
            </section>
          </div>
        </div>

        <footer class="agents-workspace__footer">
          <span class="agents-workspace__hint">⌘S 保存</span>
          <div class="agents-workspace__footer-actions">
            <DqButton v-if="isCreating" @click="cancelCreate">取消</DqButton>
            <DqButton v-if="selectedId && !isCreating" @click="removeSelected">删除</DqButton>
            <DqButton type="primary" :disabled="saving" @click="save">
              {{ isCreating ? '创建 Agent' : '保存更改' }}
            </DqButton>
          </div>
        </footer>
      </template>
    </main>
  </div>
</template>
