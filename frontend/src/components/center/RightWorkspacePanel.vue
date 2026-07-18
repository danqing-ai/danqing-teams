<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import PlanPanel from '@/components/center/PlanPanel.vue'
import FileTree from '@/components/center/FileTree.vue'
import ExpertsPanel from '@/components/center/ExpertsPanel.vue'
import ChangesPanel from '@/components/center/ChangesPanel.vue'
import TerminalPanel from '@/components/center/TerminalPanel.vue'
import type { StreamEvent } from '@/types/mission'

export type RightTab = 'plan' | 'files' | 'experts' | 'changes' | 'terminal' | 'browser'

const props = defineProps<{
  streamEvents: StreamEvent[]
  projectId: string | null
  changesCount?: number
  expertsRunning?: number
}>()

const rightTab = defineModel<RightTab>('tab', { required: true })

const emit = defineEmits<{
  openInBrowser: [path: string]
}>()

const { t } = useI18n()
const terminalOpened = ref(false)
const fileTreeRef = ref<InstanceType<typeof FileTree> | null>(null)
const changesPanelRef = ref<InstanceType<typeof ChangesPanel> | null>(null)

watch(rightTab, (tab) => {
  if (tab === 'terminal') terminalOpened.value = true
  if (tab === 'changes') changesPanelRef.value?.refresh?.()
})

const tabs = computed(() => [
  { id: 'plan' as const, label: t('sessions.tabPlan') },
  { id: 'files' as const, label: t('sessions.tabFiles') },
  { id: 'experts' as const, label: t('sessions.tabExperts'), badge: props.expertsRunning },
  { id: 'changes' as const, label: t('sessions.tabChanges'), badge: props.changesCount },
  { id: 'terminal' as const, label: t('sessions.tabTerminal') },
  { id: 'browser' as const, label: t('sessions.tabBrowser') },
])

defineExpose({
  changesPanelRef,
  refreshChanges: () => changesPanelRef.value?.refresh?.(),
})
</script>

<template>
  <div class="right-workspace">
    <div class="right-workspace__tabs">
      <button
        v-for="tab in tabs"
        :key="tab.id"
        type="button"
        class="right-workspace__tab"
        :class="{ 'is-active': rightTab === tab.id }"
        :title="tab.label"
        @click="rightTab = tab.id"
      >
        <span>{{ tab.label }}</span>
        <span v-if="tab.badge" class="right-workspace__badge">{{ tab.badge > 99 ? '99+' : tab.badge }}</span>
      </button>
    </div>

    <div class="right-workspace__body">
      <PlanPanel v-if="rightTab === 'plan'" :stream-events="streamEvents" />

      <template v-else-if="rightTab === 'files'">
        <FileTree
          v-if="projectId"
          ref="fileTreeRef"
          :project-id="projectId"
          @open-in-browser="emit('openInBrowser', $event)"
        />
        <div v-else class="right-workspace__empty">{{ t('sessions.noProjectLinked') }}</div>
      </template>

      <ExpertsPanel v-else-if="rightTab === 'experts'" :stream-events="streamEvents" />
      <ChangesPanel v-else-if="rightTab === 'changes'" ref="changesPanelRef" />

      <template v-else-if="rightTab === 'browser'">
        <slot name="browser" />
      </template>

      <template v-else-if="rightTab === 'terminal'">
        <div v-if="!projectId" class="right-workspace__empty">{{ t('sessions.noProjectLinked') }}</div>
      </template>

      <TerminalPanel
        v-if="terminalOpened && projectId"
        v-show="rightTab === 'terminal'"
        :key="projectId"
        :project-id="projectId"
      />
    </div>
  </div>
</template>

<style scoped>
.right-workspace {
  display: flex;
  flex-direction: column;
  min-width: 0;
  min-height: 0;
  height: 100%;
  overflow: hidden;
}

.right-workspace__tabs {
  display: flex;
  flex-shrink: 0;
  gap: 2px;
  padding: 6px 8px;
  border-bottom: 1px solid var(--dq-separator-light);
  overflow-x: auto;
}

.right-workspace__tab {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 6px 10px;
  border: none;
  border-radius: 8px;
  background: transparent;
  color: var(--dq-label-tertiary);
  font-size: var(--dq-font-size-footnote);
  font-weight: 600;
  cursor: pointer;
  white-space: nowrap;
}

.right-workspace__tab:hover {
  color: var(--dq-label-secondary);
  background: color-mix(in srgb, var(--dq-label-primary) 5%, transparent);
}

.right-workspace__tab.is-active {
  color: var(--dq-label-primary);
  background: color-mix(in srgb, var(--dq-label-primary) 8%, transparent);
}

.right-workspace__badge {
  min-width: 16px;
  height: 16px;
  padding: 0 4px;
  border-radius: 8px;
  background: var(--dq-accent);
  color: var(--dq-color-white);
  font-size: 10px;
  font-weight: 700;
  line-height: 16px;
  text-align: center;
}

.right-workspace__body {
  position: relative;
  display: flex;
  flex-direction: column;
  flex: 1;
  min-height: 0;
  overflow: hidden;
}

.right-workspace__body > :deep(*) {
  flex: 1 1 auto;
  min-height: 0;
  min-width: 0;
}

.right-workspace__empty {
  padding: 32px 16px;
  text-align: center;
  color: var(--dq-label-tertiary);
  font-size: var(--dq-font-size-body);
}
</style>
