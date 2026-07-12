import { defineStore } from 'pinia'
import { ref } from 'vue'
import type { MCPServer, MCPToolDef } from '@/types'
import { fetchJSON, asArray } from '@/api/client'

export const useMcpServersStore = defineStore('mcpServers', () => {
  const items = ref<MCPServer[]>([])

  async function load() {
    const data = await fetchJSON<MCPServer[]>('/mcp/servers')
    items.value = asArray(data)
  }

  async function create(payload: Omit<MCPServer, 'id' | 'status'>) {
    const server = await fetchJSON<MCPServer>('/mcp/servers', {
      method: 'POST',
      body: JSON.stringify(payload),
    })
    items.value.push(server)
    return server
  }

  async function update(id: string, payload: Partial<MCPServer>) {
    const server = await fetchJSON<MCPServer>(`/mcp/servers/${id}`, {
      method: 'PUT',
      body: JSON.stringify(payload),
    })
    const i = items.value.findIndex((s) => s.id === id)
    if (i >= 0) items.value[i] = server
    return server
  }

  async function remove(id: string) {
    await fetchJSON(`/mcp/servers/${id}`, { method: 'DELETE' })
    items.value = items.value.filter((s) => s.id !== id)
  }

  async function refreshTools(id: string): Promise<MCPToolDef[]> {
    const res = await fetchJSON<{ tools: MCPToolDef[] }>(`/mcp/servers/${id}/refresh-tools`, {
      method: 'POST',
    })
    await load()
    return res?.tools ?? []
  }

  async function toggleTool(id: string, toolName: string, enabled: boolean) {
    const server = await fetchJSON<MCPServer>(`/mcp/servers/${id}/tools/${encodeURIComponent(toolName)}`, {
      method: 'PATCH',
      body: JSON.stringify({ enabled }),
    })
    const i = items.value.findIndex((s) => s.id === id)
    if (i >= 0) items.value[i] = server
    return server
  }

  return { items, load, create, update, remove, refreshTools, toggleTool }
})
