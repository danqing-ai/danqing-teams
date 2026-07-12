<script setup lang="ts">
import { ref, watch, computed } from 'vue'
import { useSessionsStore } from '@/stores/sessions'
import { fetchJSON, asArray } from '@/api/client'

const sessions = useSessionsStore()

interface FileNode {
  name: string
  path: string
  isDir: boolean
  size?: number
  children?: FileNode[]
}

const props = defineProps<{
  projectId: string
}>()

const emit = defineEmits<{
  selectFile: [path: string]
}>()

const rootNodes = ref<FileNode[]>([])
const loading = ref(false)
const expanded = ref<Record<string, boolean>>({})
const selectedPath = ref<string | null>(null)
const childrenCache = ref<Record<string, FileNode[]>>({})

const projectId = computed(() => props.projectId)

watch(projectId, (id) => {
  if (id) loadRoot()
}, { immediate: true })

async function loadRoot() {
  if (!projectId.value) return
  loading.value = true
  try {
    rootNodes.value = asArray(await fetchJSON<FileNode[]>(`/projects/${projectId.value}/files`))
  } catch {
    rootNodes.value = []
  } finally {
    loading.value = false
  }
}

async function toggleDir(dirPath: string) {
  if (expanded.value[dirPath]) {
    expanded.value[dirPath] = false
    return
  }
  expanded.value[dirPath] = true
  if (!childrenCache.value[dirPath]) {
    try {
      childrenCache.value[dirPath] = asArray(
        await fetchJSON<FileNode[]>(`/projects/${projectId.value}/files?path=${encodeURIComponent(dirPath)}`),
      )
    } catch {
      childrenCache.value[dirPath] = []
    }
  }
}

function selectFile(path: string) {
  selectedPath.value = path
  emit('selectFile', path)
}

function fileIcon(node: FileNode): string {
  if (node.isDir) return '📁'
  const ext = node.name.split('.').pop()?.toLowerCase()
  const map: Record<string, string> = {
    html: '🌐', htm: '🌐',
    js: '📜', ts: '📘', jsx: '📜', tsx: '📘',
    vue: '💚', svelte: '🧡',
    css: '🎨', scss: '🎨', less: '🎨',
    json: '📋', yaml: '📋', yml: '📋', toml: '📋',
    md: '📝', txt: '📄',
    py: '🐍', go: '🔵', rs: '🦀',
    svg: '🖼️', png: '🖼️', jpg: '🖼️', jpeg: '🖼️', gif: '🖼️', ico: '🖼️', webp: '🖼️',
    sh: '⚡', bash: '⚡', zsh: '⚡',
    sql: '🗄️',
    lock: '🔒',
    gitignore: '🙈',
  }
  return map[ext ?? ''] ?? '📄'
}

function formatSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  return `${(bytes / 1024 / 1024).toFixed(1)} MB`
}

defineExpose({ refresh: loadRoot })
</script>

<template>
  <div class="file-tree">
    <div v-if="loading" class="file-tree__loading">加载中...</div>
    <div v-else-if="!rootNodes.length" class="file-tree__empty">
      <p>暂无文件</p>
    </div>
    <ul v-else class="file-tree__list">
      <template v-for="node in rootNodes" :key="node.path">
        <li class="file-tree__item" :class="{ 'is-dir': node.isDir, 'is-selected': selectedPath === node.path }">
          <div
            class="file-tree__row"
            @click="node.isDir ? toggleDir(node.path) : selectFile(node.path)"
          >
            <span v-if="node.isDir" class="file-tree__arrow">{{ expanded[node.path] ? '▾' : '▸' }}</span>
            <span v-else class="file-tree__arrow-spacer" />
            <span class="file-tree__icon">{{ fileIcon(node) }}</span>
            <span class="file-tree__name" :title="node.path">{{ node.name }}</span>
            <span v-if="!node.isDir && node.size" class="file-tree__size">{{ formatSize(node.size) }}</span>
          </div>
          <ul v-if="node.isDir && expanded[node.path] && childrenCache[node.path]" class="file-tree__children">
            <li
              v-for="child in childrenCache[node.path]"
              :key="child.path"
              class="file-tree__item"
              :class="{ 'is-dir': child.isDir, 'is-selected': selectedPath === child.path }"
            >
              <div
                class="file-tree__row file-tree__row--child"
                @click="child.isDir ? toggleDir(child.path) : selectFile(child.path)"
              >
                <span v-if="child.isDir" class="file-tree__arrow">{{ expanded[child.path] ? '▾' : '▸' }}</span>
                <span v-else class="file-tree__arrow-spacer" />
                <span class="file-tree__icon">{{ fileIcon(child) }}</span>
                <span class="file-tree__name" :title="child.path">{{ child.name }}</span>
                <span v-if="!child.isDir && child.size" class="file-tree__size">{{ formatSize(child.size) }}</span>
              </div>
            </li>
          </ul>
        </li>
      </template>
    </ul>
  </div>
</template>

<style scoped>
.file-tree {
  display: flex;
  flex-direction: column;
  height: 100%;
  min-height: 0;
  overflow: hidden;
  font-size: 13px;
}

.file-tree__loading,
.file-tree__empty {
  padding: 24px 16px;
  text-align: center;
  color: var(--dq-label-tertiary);
}

.file-tree__list {
  flex: 1;
  overflow-y: auto;
  list-style: none;
  margin: 0;
  padding: 4px 0;
}

.file-tree__children {
  list-style: none;
  margin: 0;
  padding: 0 0 0 12px;
}

.file-tree__item.is-selected > .file-tree__row {
  background: var(--dq-accent);
  color: var(--dq-bg-page, #fff);
}

.file-tree__row {
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 5px 16px;
  cursor: pointer;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  color: var(--dq-label-primary);
}

.file-tree__row:hover {
  background: var(--dq-bg-container-hover, rgba(0, 0, 0, 0.04));
}

.file-tree__arrow {
  flex-shrink: 0;
  width: 14px;
  font-size: 10px;
  color: var(--dq-label-tertiary);
}

.file-tree__arrow-spacer {
  width: 14px;
  flex-shrink: 0;
}

.file-tree__icon {
  flex-shrink: 0;
  width: 18px;
  font-size: 13px;
  text-align: center;
}

.file-tree__name {
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
}

.file-tree__size {
  flex-shrink: 0;
  font-size: 11px;
  color: var(--dq-label-quaternary);
}
</style>
