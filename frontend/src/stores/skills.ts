import { defineStore } from 'pinia'
import { ref } from 'vue'
import type { Skill, Tool } from '@/types'

const SKILLS_KEY = 'danqing-skills'
const TOOLS_KEY = 'danqing-tools'

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

export const useSkillsStore = defineStore('skills', () => {
  const items = ref<Skill[]>(loadJSON<Skill>(SKILLS_KEY))
  const tools = ref<Tool[]>(loadJSON<Tool>(TOOLS_KEY))
  const loading = ref(false)

  async function load() {
    loading.value = true
    try {
      // LocalStorage only; no backend wiring for now
    } finally {
      loading.value = false
    }
  }

  function createSkill(payload: Omit<Skill, 'id'> & { id: string }) {
    const skill: Skill = { ...payload }
    items.value.push(skill)
    saveJSON(SKILLS_KEY, items.value)
    return skill
  }

  function updateSkill(skillId: string, payload: Partial<Skill>) {
    const i = items.value.findIndex((s) => s.id === skillId)
    if (i < 0) return undefined
    items.value[i] = { ...items.value[i], ...payload }
    saveJSON(SKILLS_KEY, items.value)
    return items.value[i]
  }

  function removeSkill(skillId: string) {
    items.value = items.value.filter((s) => s.id !== skillId)
    saveJSON(SKILLS_KEY, items.value)
  }

  function createTool(payload: Omit<Tool, 'id'> & { id: string }) {
    const tool: Tool = { ...payload }
    tools.value.push(tool)
    saveJSON(TOOLS_KEY, tools.value)
    return tool
  }

  function updateTool(toolId: string, payload: Partial<Tool>) {
    const i = tools.value.findIndex((t) => t.id === toolId)
    if (i < 0) return undefined
    tools.value[i] = { ...tools.value[i], ...payload }
    saveJSON(TOOLS_KEY, tools.value)
    return tools.value[i]
  }

  function removeTool(toolId: string) {
    tools.value = tools.value.filter((t) => t.id !== toolId)
    saveJSON(TOOLS_KEY, tools.value)
  }

  return { items, tools, loading, load, createSkill, updateSkill, removeSkill, createTool, updateTool, removeTool }
})
