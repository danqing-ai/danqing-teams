<script setup lang="ts">
defineProps<{
  title: string
  count?: number
  countLabel?: string
  createLabel?: string
  hasSelection?: boolean
}>()

defineEmits<{
  create: []
}>()
</script>

<template>
  <div class="resource-shell float-island">
    <aside class="resource-rail">
      <div class="resource-rail__head">
        <span class="resource-rail__count">{{ count }}</span>
        <DqIconButton :aria-label="createLabel ?? '新建'" @click="$emit('create')">
          <svg viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M12 5v14M5 12h14" stroke-linecap="round" />
          </svg>
        </DqIconButton>
      </div>
      <div class="resource-rail__body">
        <slot name="rail" />
      </div>
    </aside>

    <main class="resource-workspace">
      <div v-if="!hasSelection" class="resource-workspace__empty">
        <slot name="empty">
          <DqEmpty description="选择或新建项目" />
        </slot>
      </div>
      <template v-else>
        <header class="resource-workspace__bar">
          <slot name="header" />
        </header>
        <div class="resource-workspace__scroll">
          <slot name="body" />
        </div>
        <footer class="resource-workspace__footer">
          <slot name="footer" />
        </footer>
      </template>
    </main>
  </div>
</template>
