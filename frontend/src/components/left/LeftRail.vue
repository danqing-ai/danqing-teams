<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useResizableWidth } from '@/composables/useResizableWidth'
import { useTasksStore } from '@/stores/tasks'
import TeamSelector from '@/components/left/TeamSelector.vue'
import TaskList from '@/components/left/TaskList.vue'

const COLLAPSED_KEY = 'teams-left-collapsed'

const { width, onResizePointerDown } = useResizableWidth('teams-left-width', 252, 220, 360)

const collapsed = ref(localStorage.getItem(COLLAPSED_KEY) === '1')
const tasks = useTasksStore()

watch(collapsed, (v) => {
  localStorage.setItem(COLLAPSED_KEY, v ? '1' : '0')
})

const railStyle = computed(() =>
  collapsed.value ? { width: '44px' } : { width: `${width.value}px` },
)

function openCreateTask() {
  if (collapsed.value) collapsed.value = false
  tasks.startNewTask()
}

defineExpose({ openCreateTask })
</script>

<template>
  <div
    class="teams-left-rail"
    :class="{ 'is-collapsed': collapsed }"
    :style="railStyle"
  >
    <div v-if="collapsed" class="teams-left-rail__strip">
      <DqIconButton
        class="teams-left-rail__expand"
        aria-label="展开 Tasks 面板"
        @click="collapsed = false"
      >
        <svg viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M9 6l6 6-6 6" stroke-linecap="round" stroke-linejoin="round" />
        </svg>
      </DqIconButton>
      <span class="teams-left-rail__strip-label" aria-hidden="true">Tasks</span>
    </div>

    <template v-else>
      <DqSurfaceCard class="teams-left-panel float-island">
        <template #header>
          <div class="teams-left-panel__head">
            <DqIconButton
              class="teams-left-panel__collapse"
              aria-label="收起 Tasks 面板"
              @click="collapsed = true"
            >
              <svg viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M15 6l-6 6 6 6" stroke-linecap="round" stroke-linejoin="round" />
              </svg>
            </DqIconButton>
            <h2 class="teams-left-panel__title">Tasks</h2>
          </div>
        </template>

        <TeamSelector />
        <TaskList />
      </DqSurfaceCard>

      <button
        type="button"
        class="teams-left-rail__resize"
        aria-label="调整左侧面板宽度"
        @pointerdown="onResizePointerDown"
      />
    </template>
  </div>
</template>
