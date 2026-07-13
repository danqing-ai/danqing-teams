<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { Monitor } from '@danqing/dq-shell'
import { useResizableWidth } from '@/composables/useResizableWidth'
import { useSessionsStore } from '@/stores/sessions'

const COLLAPSED_KEY = 'app-right-collapsed'
const { width, onResizePointerDown } = useResizableWidth('app-right-width', 280, 240, 420, 'right')

const collapsed = ref(localStorage.getItem(COLLAPSED_KEY) === '1')
const tab = ref<'monitor' | 'assets' | 'files' | 'workers'>('monitor')
const sessions = useSessionsStore()

watch(collapsed, (v) => localStorage.setItem(COLLAPSED_KEY, v ? '1' : '0'))

const railStyle = computed(() => (collapsed.value ? { width: '44px' } : { width: `${width.value}px` }))

const reports = computed(() =>
  sessions.streamEvents.filter((e) => e.type === 'report').map((e) => e.payload),
)
</script>

<template>
  <div class="teams-right-rail agent-inspector" :class="{ 'is-collapsed': collapsed }" :style="railStyle">
    <div v-if="collapsed" class="teams-right-rail__strip">
      <DqIconButton aria-label="展开 Inspector" @click="collapsed = false">
        <svg viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M15 6l-6 6 6 6" stroke-linecap="round" stroke-linejoin="round" />
        </svg>
      </DqIconButton>
      <span class="teams-right-rail__strip-label">Inspector</span>
    </div>

    <aside v-else class="agent-inspector__panel">
      <header class="agent-inspector__head">
        <DqIcon class="agent-inspector__head-icon" :size="18"><Monitor /></DqIcon>
        <h2 class="agent-inspector__title">Inspector</h2>
        <DqIconButton aria-label="收起 Inspector" @click="collapsed = true">
          <svg viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M9 6l6 6-6 6" stroke-linecap="round" stroke-linejoin="round" />
          </svg>
        </DqIconButton>
      </header>

      <nav class="agent-inspector__tabs" aria-label="Inspector 分区">
        <button type="button" :class="{ active: tab === 'monitor' }" @click="tab = 'monitor'">监控</button>
        <button type="button" :class="{ active: tab === 'assets' }" @click="tab = 'assets'">资产</button>
        <button type="button" :class="{ active: tab === 'files' }" @click="tab = 'files'">文件</button>
        <button type="button" :class="{ active: tab === 'workers' }" @click="tab = 'workers'">Workers</button>
      </nav>

      <div class="agent-inspector__body">
        <div v-if="tab === 'monitor'" class="agent-inspector__content">
          <ul>
            <li v-for="s in sessions.agentRuns" :key="s.id">
              <strong>{{ s.agentId }}</strong>
              <span>{{ s.status }} · step {{ s.stepsUsed }}</span>
            </li>
          </ul>
          <p v-if="!sessions.agentRuns.length" class="muted">暂无 Session</p>
        </div>
        <div v-else-if="tab === 'assets'" class="agent-inspector__content">
          <pre v-for="(r, i) in reports" :key="i">{{ JSON.stringify(r, null, 2) }}</pre>
          <p v-if="!reports.length" class="muted">暂无 Report</p>
        </div>
        <div v-else-if="tab === 'files'" class="agent-inspector__content">
          <p class="muted">Session Workspace（后续接入）</p>
        </div>
        <div v-else class="agent-inspector__content">
          <ul>
            <li v-for="w in sessions.workers" :key="w.runId">
              <strong>{{ w.agentId }}</strong>
              <span>{{ w.status }} · {{ w.stepsUsed }} steps</span>
            </li>
          </ul>
          <p v-if="!sessions.workers.length" class="muted">暂无 Worker</p>
        </div>
      </div>

      <button type="button" class="teams-right-rail__resize" aria-label="调整宽度" @pointerdown="onResizePointerDown" />
    </aside>
  </div>
</template>

<style scoped>
.agent-inspector__panel {
  flex: 1;
  min-height: 0;
  height: 100%;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  position: relative;
  background: transparent;
}

.agent-inspector__head {
  flex-shrink: 0;
  display: flex;
  align-items: center;
  gap: 8px;
  height: 36px;
  padding: 0 8px 0 12px;
  border-bottom: 1px solid color-mix(in srgb, var(--dq-label-primary) 8%, transparent);
}

.agent-inspector__head-icon {
  color: var(--dq-accent);
  flex-shrink: 0;
}

.agent-inspector__title {
  flex: 1;
  margin: 0;
  font-size: var(--dq-font-size-title);
  font-weight: 650;
  color: var(--dq-label-primary);
}

.agent-inspector__tabs {
  flex-shrink: 0;
  display: flex;
  gap: 0;
  padding: 0 8px;
  border-bottom: 1px solid color-mix(in srgb, var(--dq-label-primary) 8%, transparent);
}

.agent-inspector__tabs button {
  padding: 8px 10px;
  font-size: var(--dq-font-size-footnote);
  border: none;
  border-bottom: 2px solid transparent;
  background: transparent;
  color: var(--dq-label-tertiary);
  cursor: pointer;
}

.agent-inspector__tabs button:hover {
  color: var(--dq-label-secondary);
}

.agent-inspector__tabs button.active {
  color: var(--dq-label-primary);
  border-bottom-color: var(--dq-accent);
  font-weight: 600;
}

.agent-inspector__body {
  flex: 1;
  min-height: 0;
  overflow: hidden;
}

.agent-inspector__content {
  height: 100%;
  overflow: auto;
  padding: 8px 12px;
}

.agent-inspector__content ul {
  list-style: none;
  margin: 0;
  padding: 0;
}

.agent-inspector__content li {
  padding: 8px 0;
  border-bottom: 1px solid color-mix(in srgb, var(--dq-label-primary) 8%, transparent);
  font-size: var(--dq-font-size-body);
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.muted {
  color: var(--dq-label-tertiary);
  font-size: var(--dq-font-size-body);
}

pre {
  font-size: var(--dq-font-size-caption);
  overflow: auto;
  margin: 0 0 8px;
}

.teams-right-rail__resize {
  position: absolute;
  top: 0;
  left: -6px;
  z-index: 5;
  width: 12px;
  height: 100%;
  padding: 0;
  border: none;
  background: transparent;
  cursor: col-resize;
}
</style>
