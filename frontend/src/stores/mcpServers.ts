import { defineStore } from 'pinia'
import { ref } from 'vue'
import type { MCPServer, Tool } from '@/types'

const MCP_KEY = 'danqing-mcp-servers'

function loadJSON<T>(key: string): T[] {
  try {
    const raw = localStorage.getItem(key)
    return raw ? (JSON.parse(raw) as T[]) : []
  } catch {
    return []
  }
}

function saveJSON<T>(key: string, items: T[]) {
  localStorage.setItem(key, JSON.stringify(items))
}

export const useMcpServersStore = defineStore('mcpServers', () => {
  const items = ref<MCPServer[]>(loadJSON<MCPServer>(MCP_KEY))

  function save() {
    saveJSON(MCP_KEY, items.value)
  }

  function create(payload: Omit<MCPServer, 'id' | 'status' | 'tools'>) {
    const server: MCPServer = {
      ...payload,
      id: `mcp-${Date.now()}`,
      status: 'disconnected',
      tools: [],
    }
    items.value.push(server)
    save()
    return server
  }

  function update(id: string, payload: Partial<MCPServer>) {
    const i = items.value.findIndex((s) => s.id === id)
    if (i < 0) return undefined
    items.value[i] = { ...items.value[i], ...payload }
    save()
    return items.value[i]
  }

  function remove(id: string) {
    items.value = items.value.filter((s) => s.id !== id)
    save()
  }

  function addTool(serverId: string, tool: Omit<Tool, 'id'>) {
    const server = items.value.find((s) => s.id === serverId)
    if (!server) return undefined
    const newTool: Tool = { ...tool, id: `tool-${Date.now()}` }
    server.tools = [...(server.tools ?? []), newTool]
    save()
    return newTool
  }

  function removeTool(serverId: string, toolId: string) {
    const server = items.value.find((s) => s.id === serverId)
    if (!server) return
    server.tools = (server.tools ?? []).filter((t) => t.id !== toolId)
    save()
  }

  return { items, create, update, remove, addTool, removeTool }
})
