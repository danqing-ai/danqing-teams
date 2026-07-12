<script setup lang="ts">
import { watch, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import SessionWorkspace from '@/components/center/SessionWorkspace.vue'
import { useSessionsStore } from '@/stores/sessions'

const route = useRoute()
const sessions = useSessionsStore()

function loadIfNew(id: string) {
  if (sessions.currentSessionId !== id) {
    sessions.selectSession(id)
  }
}

onMounted(() => {
  const id = route.params.id
  if (id && typeof id === 'string') {
    loadIfNew(id)
  }
})

watch(
  () => route.params.id,
  (id) => {
    if (id && typeof id === 'string') {
      loadIfNew(id)
    }
  },
)
</script>

<template>
  <SessionWorkspace />
</template>
