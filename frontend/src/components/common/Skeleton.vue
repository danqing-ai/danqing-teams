<script setup lang="ts">
defineProps<{
  /** Row height variant */
  variant?: 'text' | 'title' | 'card' | 'circle' | 'block'
  /** Width override (e.g. '60%', '120px') */
  width?: string
  /** Height override */
  height?: string
  /** Rounded corners */
  rounded?: boolean
}>()
</script>

<template>
  <div
    class="skeleton"
    :class="[`skeleton--${variant ?? 'text'}`, { 'skeleton--rounded': rounded }]"
    :style="{ width: width ?? undefined, height: height ?? undefined }"
    aria-hidden="true"
    role="presentation"
  />
</template>

<style scoped>
.skeleton {
  background: linear-gradient(
    90deg,
    color-mix(in srgb, var(--dq-label-primary) 8%, transparent) 25%,
    color-mix(in srgb, var(--dq-label-primary) 14%, transparent) 50%,
    color-mix(in srgb, var(--dq-label-primary) 8%, transparent) 75%
  );
  background-size: 200% 100%;
  animation: skeleton-shimmer 1.5s ease-in-out infinite;
  border-radius: 6px;
}

.skeleton--rounded {
  border-radius: 999px;
}

.skeleton--text {
  height: 13px;
  border-radius: 4px;
}

.skeleton--title {
  height: 16px;
  border-radius: 5px;
}

.skeleton--card {
  height: 64px;
}

.skeleton--circle {
  width: 32px;
  height: 32px;
  border-radius: 50%;
}

.skeleton--block {
  height: 120px;
}

@keyframes skeleton-shimmer {
  0% { background-position: 200% 0; }
  100% { background-position: -200% 0; }
}
</style>
