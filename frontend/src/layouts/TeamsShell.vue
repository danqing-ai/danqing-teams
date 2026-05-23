<script setup lang="ts">
import { onMounted, onUnmounted, provide, ref } from 'vue'
import LeftRail from '@/components/left/LeftRail.vue'
import FloatingCanvas from '@/components/center/FloatingCanvas.vue'
import RightDock from '@/components/right/RightDock.vue'
import FloatingComposer from '@/components/composer/FloatingComposer.vue'
import { OPEN_CREATE_TASK_KEY, FOCUS_COMPOSER_KEY } from '@/composables/useTaskActions'
import { useTasksStore } from '@/stores/tasks'

const emit = defineEmits<{
  openAgents: [agentId?: string]
}>()

const leftRailRef = ref<InstanceType<typeof LeftRail> | null>(null)
const composerRef = ref<InstanceType<typeof FloatingComposer> | null>(null)
const tasks = useTasksStore()

function openCreateTask() {
  leftRailRef.value?.openCreateTask()
}

function focusComposer() {
  composerRef.value?.focusInput()
}

provide(OPEN_CREATE_TASK_KEY, openCreateTask)
provide(FOCUS_COMPOSER_KEY, focusComposer)

function onGlobalKeydown(e: KeyboardEvent) {
  if (e.key === 'Escape' && tasks.composingNew) {
    e.preventDefault()
    tasks.cancelCompose()
    return
  }

  const mod = e.metaKey || e.ctrlKey
  if (!mod) return

  if (e.key === 'n' || e.key === 'N') {
    e.preventDefault()
    openCreateTask()
  }
}

onMounted(() => {
  window.addEventListener('keydown', onGlobalKeydown)
})

onUnmounted(() => {
  window.removeEventListener('keydown', onGlobalKeydown)
})
</script>

<template>
  <div class="teams-shell">
    <LeftRail ref="leftRailRef" />

    <main class="float-island float-island--center">
      <div class="center-stage">
        <FloatingCanvas />
        <div class="center-stage__composer">
          <FloatingComposer ref="composerRef" />
        </div>
      </div>
    </main>

    <RightDock aria-label="Task inspector" @open-agents="emit('openAgents', $event)" />
  </div>
</template>
