<script setup lang="ts">
import { ref, watch, computed } from 'vue'
import { fetchJSON } from '@/api/client'
import Skeleton from '@/components/common/Skeleton.vue'

interface FileContent {
  name: string
  path: string
  size: number
  contentType: string
  content: string
  binary: boolean
}

const props = defineProps<{
  projectId: string
  filePath: string | null
}>()

const content = ref<FileContent | null>(null)
const loading = ref(false)
const error = ref('')

watch(() => props.filePath, (path) => {
  if (path) loadFile(path)
  else content.value = null
}, { immediate: true })

async function loadFile(path: string) {
  if (!props.projectId) return
  loading.value = true
  error.value = ''
  try {
    content.value = await fetchJSON<FileContent>(
      `/projects/${props.projectId}/files/content?path=${encodeURIComponent(path)}`,
    )
  } catch (e) {
    error.value = e instanceof Error ? e.message : '加载失败'
    content.value = null
  } finally {
    loading.value = false
  }
}

const isImage = computed(() => content.value?.contentType.startsWith('image/'))
const isHTML = computed(() => content.value?.contentType.startsWith('text/html'))
const isText = computed(() =>
  content.value && !content.value.binary && !isHTML.value && !isImage.value,
)

function formatSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  return `${(bytes / 1024 / 1024).toFixed(1)} MB`
}
</script>

<template>
  <div class="file-viewer">
    <div v-if="content" class="file-viewer__meta-bar">
      {{ content.name }} · {{ formatSize(content.size) }} · {{ content.contentType }}
    </div>

    <div v-if="loading" class="file-viewer__loading">
      <Skeleton variant="title" width="40%" />
      <Skeleton variant="block" width="100%" height="180px" />
    </div>

    <div v-else-if="error" class="file-viewer__error">{{ error }}</div>

    <div v-else-if="!content" class="file-viewer__empty">
      <p>选择文件以预览</p>
    </div>

    <div v-else class="file-viewer__content">
      <div v-if="isImage" class="file-viewer__image-wrap">
        <img :src="content.content" :alt="content.name" class="file-viewer__image" />
      </div>

      <div v-else-if="isHTML" class="file-viewer__html-wrap">
        <iframe
          :srcdoc="content.content"
          class="file-viewer__iframe"
          sandbox="allow-scripts allow-same-origin"
          title="HTML Preview"
        />
      </div>

      <pre v-else-if="isText" class="file-viewer__text"><code>{{ content.content }}</code></pre>

      <div v-else class="file-viewer__binary">
        <p>二进制文件，无法预览</p>
        <p class="file-viewer__binary-meta">类型: {{ content.contentType }} · 大小: {{ formatSize(content.size) }}</p>
      </div>
    </div>
  </div>
</template>

<style scoped>
.file-viewer {
  display: flex;
  flex-direction: column;
  height: 100%;
  min-height: 0;
  overflow: hidden;
}

.file-viewer__meta-bar {
  flex-shrink: 0;
  padding: 6px 12px;
  font-size: var(--dq-font-size-caption);
  color: var(--dq-label-quaternary);
  border-bottom: 1px solid var(--dq-separator-light);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.file-viewer__loading,
.file-viewer__empty,
.file-viewer__error {
  padding: 24px 16px;
  text-align: center;
  color: var(--dq-label-tertiary);
}

.file-viewer__content {
  flex: 1;
  overflow: hidden;
  min-height: 0;
}

.file-viewer__image-wrap {
  height: 100%;
  overflow: auto;
  display: flex;
  align-items: flex-start;
  justify-content: center;
  padding: 16px;
}

.file-viewer__image {
  max-width: 100%;
  height: auto;
  border-radius: 4px;
}

.file-viewer__html-wrap {
  height: 100%;
  min-height: 0;
}

.file-viewer__iframe {
  width: 100%;
  height: 100%;
  border: none;
  background: var(--dq-bg-elevated);
}

.file-viewer__text {
  height: 100%;
  overflow: auto;
  margin: 0;
  padding: 12px 16px;
  font-family: 'SF Mono', 'Cascadia Code', 'Fira Code', monospace;
  font-size: var(--dq-font-size-footnote);
  line-height: 1.6;
  white-space: pre-wrap;
  word-break: break-all;
  color: var(--dq-label-primary);
  background: var(--dq-bg-base);
}

.file-viewer__binary {
  padding: 24px 16px;
  text-align: center;
  color: var(--dq-label-tertiary);
}

.file-viewer__binary-meta {
  margin-top: 4px;
  font-size: var(--dq-font-size-caption);
  color: var(--dq-label-quaternary);
}
</style>
