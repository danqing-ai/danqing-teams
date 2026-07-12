<script setup lang="ts">
import { computed, onMounted, watch } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import LeftRail from '@/components/left/LeftRail.vue'
import { useSessionsStore } from '@/stores/sessions'
import { useProjectsStore } from '@/stores/projects'
import { useLLMStore } from '@/stores/llm'
import type { AppModule } from '@/types/app-module'

const router = useRouter()
const route = useRoute()
const sessions = useSessionsStore()
const projects = useProjectsStore()
const llm = useLLMStore()

const activeModule = computed<AppModule>(() => {
  const name = route.name as string
  if (name === 'sessions') return 'sessions'
  if (['workers', 'knowledge', 'skills', 'mcpServers', 'automations', 'settings'].includes(name)) {
    return name as AppModule
  }
  return 'sessions'
})

function navigateTo(module: AppModule) {
  if (module === 'sessions') {
    router.push({ name: 'sessions' })
  } else {
    router.push({ name: module })
  }
}

function onSelectSession(id: string) {
  sessions.selectSession(id)
  router.push({ name: 'sessions', params: { id } })
}

onMounted(async () => {
  await sessions.loadCatalog()
  await Promise.all([projects.loadProjects(), llm.loadConfigs(), llm.loadModels()])
  if (projects.projects.length) {
    sessions.selectedProjectId = projects.projects[0].id
  }
  sessions.syncModelSelection(llm.models, new Set())
  await sessions.loadSessions()
})

watch(() => llm.models, (newModels, oldModels) => {
  const oldIds = new Set((oldModels ?? []).map((m) => m.id))
  sessions.syncModelSelection(newModels, oldIds)
})
</script>

<template>
  <div class="app-layout">
    <LeftRail :active-module="activeModule" @navigate="navigateTo" @select-session="onSelectSession" @new-session="(pid?: string) => { sessions.startCompose(pid ?? projects.projects[0]?.id ?? null); router.push({ name: 'sessions' }) }" />
    <main class="app-workspace">
      <RouterView />
    </main>
  </div>
</template>

<style scoped>
.app-layout {
  display: grid;
  grid-template-columns: auto minmax(0, 1fr);
  grid-template-rows: minmax(0, 1fr);
  height: 100vh;
  overflow: hidden;
  background: var(--dq-bg-page);
}

.app-workspace {
  position: relative;
  display: flex;
  flex-direction: column;
  min-width: 0;
  min-height: 0;
  overflow: hidden;
  padding: 0;
  background: transparent;
}

.app-workspace :deep(.resource-shell) {
  border-radius: 0;
  border-top: none;
  border-bottom: none;
  border-right: none;
  height: 100%;
}
</style>
