import { defineStore } from 'pinia'
import { fetchJSON } from '@/api/client'
import { useTeamsStore } from './teams'

export const useTeamAgentsStore = defineStore('teamAgents', () => {
  async function addMember(agentId: string) {
    const teams = useTeamsStore()
    if (!teams.currentTeamId) return
    await fetchJSON(`/teams/${teams.currentTeamId}/agent-members/${agentId}`, {
      method: 'POST',
    })
    await teams.loadWorkers()
  }

  async function removeMember(agentId: string) {
    const teams = useTeamsStore()
    if (!teams.currentTeamId) return
    await fetchJSON(`/teams/${teams.currentTeamId}/agent-members/${agentId}`, {
      method: 'DELETE',
    })
    await teams.loadWorkers()
  }

  return { addMember, removeMember }
})
