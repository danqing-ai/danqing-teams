import { defineStore } from 'pinia'
import { ref } from 'vue'
import { asArray, fetchJSON } from '@/api/client'
import { useTeamsStore } from './teams'
import type { WorkspaceArtifact } from '@/types'

export const useWorkspaceStore = defineStore('workspace', () => {
  const artifacts = ref<WorkspaceArtifact[]>([])

  async function load(taskId?: string) {
    const teams = useTeamsStore()
    if (!teams.currentTeamId) return
    const list = asArray(
      await fetchJSON<WorkspaceArtifact[] | null>(`/teams/${teams.currentTeamId}/workspace`),
    )
    artifacts.value = taskId
      ? list.filter((a) => !a.taskId || a.taskId === taskId)
      : list
  }

  function reset() {
    artifacts.value = []
  }

  return { artifacts, load, reset }
})
