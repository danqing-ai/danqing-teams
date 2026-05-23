import { defineStore } from 'pinia'
import { ref } from 'vue'
import { asArray, fetchJSON } from '@/api/client'
import type { Team, WorkerAgent } from '@/types'

export const useTeamsStore = defineStore('teams', () => {
  const teams = ref<Team[]>([])
  const currentTeamId = ref<string>('')
  const workers = ref<WorkerAgent[]>([])

  async function loadTeams() {
    teams.value = asArray(await fetchJSON<Team[] | null>('/teams'))
    if (!currentTeamId.value && teams.value.length) {
      currentTeamId.value = teams.value[0].id
      await loadWorkers()
    }
  }

  async function loadWorkers() {
    if (!currentTeamId.value) return
    workers.value = asArray(
      await fetchJSON<WorkerAgent[] | null>(`/teams/${currentTeamId.value}/workers`),
    )
  }

  async function selectTeam(id: string) {
    currentTeamId.value = id
    await loadWorkers()
  }

  async function createTeam(name: string, description?: string) {
    const team = await fetchJSON<Team>('/teams', {
      method: 'POST',
      body: JSON.stringify({ name, description: description || undefined }),
    })
    teams.value.push(team)
    currentTeamId.value = team.id
    await loadWorkers()
    return team
  }

  async function updateTeam(id: string, patch: { name?: string; description?: string }) {
    const team = await fetchJSON<Team>(`/teams/${id}`, {
      method: 'PATCH',
      body: JSON.stringify(patch),
    })
    const list = asArray(teams.value)
    const i = list.findIndex((t) => t.id === id)
    if (i >= 0) {
      list[i] = { ...list[i], ...team }
      teams.value = list
    }
    return team
  }

  async function deleteTeam(id: string) {
    await fetchJSON(`/teams/${id}`, { method: 'DELETE' })
    teams.value = teams.value.filter((t) => t.id !== id)
    if (currentTeamId.value === id) {
      currentTeamId.value = teams.value[0]?.id ?? ''
      if (currentTeamId.value) await loadWorkers()
      else workers.value = []
    }
  }

  return {
    teams,
    currentTeamId,
    workers,
    loadTeams,
    loadWorkers,
    selectTeam,
    createTeam,
    updateTeam,
    deleteTeam,
  }
})
