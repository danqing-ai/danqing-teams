<script setup lang="ts">
import { computed } from 'vue'
import { useTeamsStore } from '@/stores/teams'
import { useTeamAgentsStore } from '@/stores/teamAgents'
import { confirm, toast } from '@/utils/feedback'
import { workerInitial } from '@/utils/stream-actors'
import type { WorkerAgent } from '@/types'

const open = defineModel<boolean>('open', { required: true })

const emit = defineEmits<{
  openAgents: [agentId?: string]
  add: []
}>()

const teams = useTeamsStore()
const teamAgents = useTeamAgentsStore()

const memberCount = computed(() => teams.workers.length)

function onEdit(worker: WorkerAgent) {
  open.value = false
  emit('openAgents', worker.id)
}

function onAdd() {
  open.value = false
  emit('add')
}

async function onRemove(worker: WorkerAgent) {
  try {
    await confirm(
      `确定将「${worker.name}」移出 Team？`,
      '移除成员',
      { type: 'warning' },
    )
  } catch {
    return
  }
  try {
    await teamAgents.removeMember(worker.id)
    toast.success('成员已移除')
  } catch (e) {
    toast.error(e instanceof Error ? e.message : '移除失败')
  }
}
</script>

<template>
  <DqDialog
    v-model:open="open"
    :title="`Team 成员 (${memberCount})`"
    width="min(480px, 92vw)"
  >
    <DqEmpty v-if="!teams.workers.length" description="暂无 Worker 成员">
      <p class="agent-members__empty-hint">从 Agents 库添加 Team-Worker 成员。</p>
    </DqEmpty>

    <ul v-else class="agent-members__list" role="list">
      <li v-for="worker in teams.workers" :key="worker.id" class="agent-members__item">
        <span class="agent-members__avatar" aria-hidden="true">
          {{ workerInitial(worker.name) }}
        </span>
        <div class="agent-members__meta">
          <p class="agent-members__name">{{ worker.name }}</p>
          <p class="agent-members__persona">{{ worker.persona }}</p>
        </div>
        <div class="agent-members__actions">
          <button type="button" class="agent-members__action" @click="onEdit(worker)">
            编辑
          </button>
          <button
            type="button"
            class="agent-members__action agent-members__action--danger"
            @click="onRemove(worker)"
          >
            移除
          </button>
        </div>
      </li>
    </ul>

    <template #footer>
      <DqButton type="primary" @click="onAdd">从 Agents 库添加</DqButton>
    </template>
  </DqDialog>
</template>
