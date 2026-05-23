<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useGlobalAgentsStore } from '@/stores/globalAgents'
import { useTeamAgentsStore } from '@/stores/teamAgents'
import { useTeamsStore } from '@/stores/teams'
import { toast } from '@/utils/feedback'

const open = defineModel<boolean>('open', { required: true })

const globalAgents = useGlobalAgentsStore()
const teamAgents = useTeamAgentsStore()
const teams = useTeamsStore()

const adding = ref<string | null>(null)

const memberIds = computed(() => new Set(teams.workers.map((w) => w.id)))

const available = computed(() =>
  globalAgents.items.filter(
    (a) => a.role === 'team-worker' && !memberIds.value.has(a.id),
  ),
)

watch(open, (v) => {
  if (v) void globalAgents.load('team-worker')
})

async function add(agentId: string) {
  adding.value = agentId
  try {
    await teamAgents.addMember(agentId)
    toast.success('已加入 Team')
    if (!available.value.length) open.value = false
  } catch (e) {
    toast.error(e instanceof Error ? e.message : '加入失败')
  } finally {
    adding.value = null
  }
}
</script>

<template>
  <DqDialog v-model:open="open" title="从 Agents 库添加成员" width="min(480px, 92vw)">
    <DqEmpty v-if="!available.length" description="没有可添加的 Team-Worker">
      <p class="agent-picker__hint">请先在 Agents 页创建 Team-Worker 角色 Agent。</p>
    </DqEmpty>
    <ul v-else class="agent-picker__list" role="list">
      <li v-for="agent in available" :key="agent.id" class="agent-picker__item">
        <div class="agent-picker__meta">
          <p class="agent-picker__name">{{ agent.name }}</p>
          <p class="agent-picker__desc">{{ agent.description }}</p>
        </div>
        <DqButton
          size="sm"
          type="primary"
          :disabled="adding === agent.id"
          @click="add(agent.id)"
        >
          加入
        </DqButton>
      </li>
    </ul>
  </DqDialog>
</template>
