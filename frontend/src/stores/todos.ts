import { defineStore } from 'pinia'
import { ref } from 'vue'
import { asArray, fetchJSON } from '@/api/client'
import { useTeamsStore } from './teams'
import type { TodoItem } from '@/types'

export const useTodosStore = defineStore('todos', () => {
  const items = ref<TodoItem[]>([])

  async function load(taskId?: string) {
    const teams = useTeamsStore()
    if (!teams.currentTeamId) return
    const q = taskId ? `?taskId=${taskId}` : ''
    items.value = asArray(
      await fetchJSON<TodoItem[] | null>(`/teams/${teams.currentTeamId}/todos${q}`),
    )
  }

  async function add(title: string, taskId?: string) {
    const teams = useTeamsStore()
    const item = await fetchJSON<TodoItem>(`/teams/${teams.currentTeamId}/todos`, {
      method: 'POST',
      body: JSON.stringify({ title, taskId }),
    })
    items.value.push(item)
  }

  async function toggle(id: string, done: boolean) {
    const teams = useTeamsStore()
    const item = await fetchJSON<TodoItem>(`/teams/${teams.currentTeamId}/todos/${id}`, {
      method: 'PATCH',
      body: JSON.stringify({ done }),
    })
    const i = items.value.findIndex((t) => t.id === id)
    if (i >= 0) items.value[i] = item
  }

  function reset() {
    items.value = []
  }

  return { items, load, add, toggle, reset }
})
