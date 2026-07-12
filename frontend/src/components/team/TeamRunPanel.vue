<script setup lang="ts">
import SessionWorkspace from '@/components/center/SessionWorkspace.vue'
</script>

<template>
  <div class="team-run-panel">
    <div class="team-run-panel__canvas">
      <SessionWorkspace />
    </div>
  </div>
</template>

<style scoped>
.team-run-panel {
  position: relative;
  display: flex;
  flex-direction: column;
  flex: 1;
  min-height: 0;
  height: 100%;
  background: transparent;
}

.team-run-panel__canvas {
  flex: 1;
  min-height: 0;
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

.team-run-panel__composer {
  position: absolute;
  left: 50%;
  bottom: 16px;
  z-index: 20;
  width: min(720px, calc(100% - 40px));
  transform: translateX(-50%);
  pointer-events: none;
}

.team-run-panel__composer :deep(.composer-float) {
  pointer-events: auto;
  overflow: hidden;
  border-radius: 16px;
  background: color-mix(in srgb, var(--dq-bg-base) 88%, transparent);
  border: 1px solid color-mix(in srgb, var(--dq-label-primary) 10%, transparent);
  box-shadow: 0 12px 40px color-mix(in srgb, var(--dq-mask) 24%, transparent);
  backdrop-filter: blur(20px) saturate(160%);
  -webkit-backdrop-filter: blur(20px) saturate(160%);
  transition: border-color 0.18s ease, box-shadow 0.18s ease;
}

.team-run-panel__composer :deep(.composer-float:focus-within) {
  border-color: color-mix(in srgb, var(--dq-accent) 28%, transparent);
  box-shadow:
    0 0 0 1px color-mix(in srgb, var(--dq-accent) 12%, transparent),
    0 12px 40px color-mix(in srgb, var(--dq-mask) 24%, transparent);
}
</style>
