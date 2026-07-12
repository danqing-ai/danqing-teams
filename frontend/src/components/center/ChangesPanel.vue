<script setup lang="ts">
import { ref, watch, computed, onMounted } from 'vue'
import { useSessionsStore } from '@/stores/sessions'
import { fetchJSON } from '@/api/client'

interface GitFileChange {
  status: string
  file: string
  origFile?: string
  staged: boolean
}

interface GitChanges {
  branch: string
  changes: GitFileChange[]
  error?: string
}

const sessions = useSessionsStore()
const data = ref<GitChanges | null>(null)
const loading = ref(false)

async function loadChanges() {
  if (!sessions.selectedProjectId) {
    data.value = null
    return
  }
  loading.value = true
  try {
    data.value = await fetchJSON<GitChanges>(`/projects/${sessions.selectedProjectId}/git-changes`)
  } catch {
    data.value = { branch: '', changes: [], error: '加载失败' }
  } finally {
    loading.value = false
  }
}

watch(() => sessions.selectedProjectId, loadChanges)
onMounted(loadChanges)

function statusLabel(s: string): string {
  switch (s) {
    case 'M': return 'M'
    case 'A': return 'A'
    case 'D': return 'D'
    case 'R': return 'R'
    case 'C': return 'C'
    case '??': return '?'
    default: return s
  }
}

function statusType(s: string): string {
  switch (s) {
    case 'M': return 'modified'
    case 'A': return 'added'
    case 'D': return 'deleted'
    case 'R': return 'renamed'
    case 'C': return 'copied'
    case '??': return 'untracked'
    default: return 'modified'
  }
}

const stagedChanges = computed(() => data.value?.changes?.filter(c => c.staged) ?? [])
const unstagedChanges = computed(() => data.value?.changes?.filter(c => !c.staged) ?? [])

function changeLabel(c: GitFileChange): string {
  if (c.origFile) return `${c.file} ← ${c.origFile}`
  return c.file
}
</script>

<template>
  <aside class="changes-panel">
    <div v-if="!sessions.selectedProjectId" class="changes-panel__empty">
      <p>未关联项目</p>
    </div>

    <div v-else-if="loading" class="changes-panel__empty">
      <p>加载中...</p>
    </div>

    <div v-else-if="data?.error" class="changes-panel__empty">
      <p>{{ data.error }}</p>
    </div>

    <div v-else-if="!data?.changes?.length" class="changes-panel__empty">
      <p>没有文件变更</p>
      <p class="changes-panel__hint">暂存区和工作区都是干净的</p>
    </div>

    <div v-else class="changes-panel__list">
      <div v-if="data.branch" class="changes-panel__branch">
        <span class="changes-panel__branch-icon">⎇</span>
        <span class="changes-panel__branch-name">{{ data.branch }}</span>
      </div>

      <template v-if="stagedChanges.length">
        <div class="changes-panel__group-label">已暂存</div>
        <div
          v-for="c in stagedChanges"
          :key="c.file"
          class="changes-panel__item"
          :class="`is-${statusType(c.status)}`"
        >
          <span class="changes-panel__item-status">{{ statusLabel(c.status) }}</span>
          <span class="changes-panel__item-file" :title="changeLabel(c)">{{ changeLabel(c) }}</span>
        </div>
      </template>

      <template v-if="unstagedChanges.length">
        <div class="changes-panel__group-label">未暂存</div>
        <div
          v-for="c in unstagedChanges"
          :key="c.file"
          class="changes-panel__item"
          :class="`is-${statusType(c.status)}`"
        >
          <span class="changes-panel__item-status">{{ statusLabel(c.status) }}</span>
          <span class="changes-panel__item-file" :title="changeLabel(c)">{{ changeLabel(c) }}</span>
        </div>
      </template>
    </div>
  </aside>
</template>

<style scoped>
.changes-panel {
  display: flex;
  flex-direction: column;
  height: 100%;
  min-height: 0;
  overflow: hidden;
  background: var(--teams-glass-bg, rgba(255, 255, 255, 0.04));
  font-size: 13px;
}

.changes-panel__empty {
  padding: 32px 16px;
  text-align: center;
  color: var(--dq-label-tertiary);
}

.changes-panel__hint {
  margin-top: 8px;
  font-size: 11px;
  color: var(--dq-label-quaternary);
  line-height: 1.5;
}

.changes-panel__list {
  flex: 1;
  overflow-y: auto;
  padding: 8px 0;
}

.changes-panel__branch {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px 14px 12px;
  font-size: 12px;
  font-weight: 600;
  color: var(--dq-label-primary);
  border-bottom: 1px solid var(--dq-separator-light, rgba(0, 0, 0, 0.06));
}

.changes-panel__branch-icon {
  font-size: 14px;
  color: var(--dq-accent);
}

.changes-panel__branch-name {
  font-family: var(--dq-font-mono);
}

.changes-panel__group-label {
  padding: 8px 14px 4px;
  font-size: 10px;
  font-weight: 600;
  letter-spacing: 0.06em;
  text-transform: uppercase;
  color: var(--dq-label-quaternary);
}

.changes-panel__item {
  display: flex;
  align-items: flex-start;
  gap: 8px;
  padding: 6px 14px;
  line-height: 1.45;
  color: var(--dq-label-primary);
}

.changes-panel__item:hover {
  background: color-mix(in srgb, var(--dq-label-primary) 4%, transparent);
}

.changes-panel__item-status {
  flex-shrink: 0;
  width: 18px;
  height: 18px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 4px;
  font-size: 10px;
  font-weight: 700;
  font-family: var(--dq-font-mono);
  color: #fff;
  background: color-mix(in srgb, var(--dq-label-secondary) 60%, transparent);
}

.changes-panel__item.is-modified .changes-panel__item-status {
  background: #d4a017;
}

.changes-panel__item.is-added .changes-panel__item-status {
  background: var(--dq-success, #2d8a56);
}

.changes-panel__item.is-deleted .changes-panel__item-status {
  background: var(--dq-danger, #e03e2d);
}

.changes-panel__item.is-renamed .changes-panel__item-status,
.changes-panel__item.is-copied .changes-panel__item-status {
  background: var(--dq-accent, #4f80ff);
}

.changes-panel__item.is-untracked .changes-panel__item-status {
  background: color-mix(in srgb, var(--dq-label-tertiary) 80%, transparent);
}

.changes-panel__item-file {
  word-break: break-word;
  font-size: 12px;
  font-family: var(--dq-font-mono);
}
</style>
