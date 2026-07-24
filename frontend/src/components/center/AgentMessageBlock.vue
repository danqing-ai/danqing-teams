<script setup lang="ts">
defineProps<{
  html: string
}>()
</script>

<template>
  <div class="agent-msg dq-prose" v-html="html" />
</template>

<style scoped>
.agent-msg {
  padding: 2px 0;
}
</style>
