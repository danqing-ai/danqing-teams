import { defineStore } from 'pinia'
import { ref } from 'vue'
import { asArray, fetchJSON } from '@/api/client'
import type { Agent, AgentRole, CreateAgentPayload, UpdateAgentPayload } from '@/types'

export const useGlobalAgentsStore = defineStore('globalAgents', () => {
  const items = ref<Agent[]>([])

  async function load(role?: AgentRole) {
    const q = role ? `?role=${role}` : ''
    items.value = asArray(await fetchJSON<Agent[] | null>(`/agents${q}`))
  }

  async function get(agentId: string) {
    return fetchJSON<Agent>(`/agents/${agentId}`)
  }

  async function create(payload: CreateAgentPayload) {
    const agent = await fetchJSON<Agent>('/agents', {
      method: 'POST',
      body: JSON.stringify(payload),
    })
    items.value.push(agent)
    return agent
  }

  async function update(agentId: string, payload: UpdateAgentPayload) {
    const agent = await fetchJSON<Agent>(`/agents/${agentId}`, {
      method: 'PATCH',
      body: JSON.stringify(payload),
    })
    const i = items.value.findIndex((a) => a.id === agentId)
    if (i >= 0) items.value[i] = agent
    return agent
  }

  async function remove(agentId: string) {
    await fetchJSON(`/agents/${agentId}`, { method: 'DELETE' })
    items.value = items.value.filter((a) => a.id !== agentId)
  }

  return { items, load, get, create, update, remove }
})
