<script setup lang="ts">
defineProps<{
  text?: string
  images?: string[]
}>()
</script>

<template>
  <div v-if="text || images?.length" class="user-msg">
    <div class="user-msg__bubble">
      <div v-if="images?.length" class="user-msg__images">
        <img
          v-for="(src, i) in images"
          :key="`img-${i}`"
          class="user-msg__image"
          :src="src"
          alt="attachment"
        />
      </div>
      <p v-if="text && text !== '[Image attachment]'">{{ text }}</p>
    </div>
  </div>
</template>

<style scoped>
.user-msg {
  display: flex;
  justify-content: flex-end;
}

.user-msg__bubble {
  max-width: min(85%, 520px);
  padding: 10px 14px;
  border-radius: 14px 14px 4px 14px;
  font-size: var(--dq-font-size-prose, var(--dq-font-size-secondary));
  line-height: var(--dq-line-height-prose, 1.55);
  color: var(--dq-label-primary);
  background: color-mix(in srgb, var(--dq-label-primary) 6%, transparent);
  border: 1px solid color-mix(in srgb, var(--dq-label-primary) 10%, transparent);
  word-break: break-word;
}

.user-msg__bubble p {
  margin: 0;
}

.user-msg__images {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  margin-bottom: 6px;
}

.user-msg__images:last-child {
  margin-bottom: 0;
}

.user-msg__image {
  max-width: min(220px, 100%);
  max-height: 160px;
  border-radius: 8px;
  object-fit: contain;
  background: color-mix(in srgb, var(--dq-label-primary) 4%, transparent);
}
</style>
