<script setup lang="ts">
import { onMounted, ref } from 'vue'
import TopNav, { type AppModule } from '@/components/shell/TopNav.vue'
import TeamsShell from '@/layouts/TeamsShell.vue'
import AgentsManagement from '@/views/AgentsManagement.vue'
import SettingsPlaceholder from '@/views/SettingsPlaceholder.vue'
import { useTeamsStore } from '@/stores/teams'
import { useTasksStore } from '@/stores/tasks'
import { useGlobalAgentsStore } from '@/stores/globalAgents'

const activeModule = ref<AppModule>('teams')
const pendingAgentId = ref<string | undefined>()

const teams = useTeamsStore()
const tasks = useTasksStore()
const globalAgents = useGlobalAgentsStore()

onMounted(async () => {
  await teams.loadTeams()
  await globalAgents.load()
  await tasks.loadTasks()
  if (tasks.tasks.length && !tasks.currentTaskId) {
    tasks.selectTask(tasks.tasks[0].id)
  }
})

function openAgents(agentId?: string) {
  pendingAgentId.value = agentId
  activeModule.value = 'agents'
}
</script>

<template>
  <div class="teams-app">
    <TopNav :active-module="activeModule" @navigate="activeModule = $event" />
    <main class="teams-app__main" :class="{ 'teams-app__main--agents': activeModule === 'agents' }">
      <TeamsShell v-if="activeModule === 'teams'" @open-agents="openAgents" />
      <AgentsManagement
        v-else-if="activeModule === 'agents'"
        :initial-agent-id="pendingAgentId"
      />
      <div v-else class="float-island float-island--settings">
        <SettingsPlaceholder />
      </div>
    </main>
  </div>
</template>
