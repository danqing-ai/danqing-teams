<script setup lang="ts">
import { computed, ref } from 'vue'
import { renderMarkdown } from '@/utils/markdown-render'

const model = defineModel<string>({ default: '' })

withDefaults(
  defineProps<{
    placeholder?: string
    rows?: number
    label?: string
  }>(),
  {
    placeholder: '支持 Markdown…',
    rows: 12,
    label: '',
  },
)

type MdMode = 'edit' | 'preview' | 'split'

const mode = ref<MdMode>('edit')

const modes: { id: MdMode; label: string }[] = [
  { id: 'edit', label: '编辑' },
  { id: 'preview', label: '预览' },
  { id: 'split', label: '分屏' },
]

const previewHtml = computed(() => renderMarkdown(model.value))

const isEmpty = computed(() => !model.value.trim())
</script>

<template>
  <div class="md-editor">
    <div v-if="label" class="md-editor__label-row">
      <span class="md-editor__label">{{ label }}</span>
      <div class="md-editor__tabs" role="tablist" aria-label="Markdown 视图">
        <button
          v-for="m in modes"
          :key="m.id"
          type="button"
          class="md-editor__tab"
          :class="{ 'is-active': mode === m.id }"
          role="tab"
          :aria-selected="mode === m.id"
          @click="mode = m.id"
        >
          {{ m.label }}
        </button>
      </div>
    </div>

    <div v-else class="md-editor__tabs md-editor__tabs--solo" role="tablist" aria-label="Markdown 视图">
      <button
        v-for="m in modes"
        :key="m.id"
        type="button"
        class="md-editor__tab"
        :class="{ 'is-active': mode === m.id }"
        role="tab"
        :aria-selected="mode === m.id"
        @click="mode = m.id"
      >
        {{ m.label }}
      </button>
    </div>

    <div
      class="md-editor__body"
      :class="{
        'is-edit': mode === 'edit',
        'is-preview': mode === 'preview',
        'is-split': mode === 'split',
      }"
    >
      <div v-if="mode !== 'preview'" class="md-editor__pane md-editor__pane--edit">
        <textarea
          v-model="model"
          class="md-editor__textarea"
          :rows="rows"
          :placeholder="placeholder"
          spellcheck="false"
        />
      </div>

      <div v-if="mode !== 'edit'" class="md-editor__pane md-editor__pane--preview">
        <div v-if="isEmpty" class="md-editor__empty">暂无内容，切换「编辑」开始编写 Markdown。</div>
        <div
          v-else
          class="md-editor__preview markdown-body"
          v-html="previewHtml"
        />
      </div>
    </div>
  </div>
</template>
