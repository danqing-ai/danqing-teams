import { defineStore } from 'pinia'
import { ref } from 'vue'
import { fetchJSON, asArray } from '@/api/client'
import type { Skill, SkillFile } from '@/types'

export const useSkillsStore = defineStore('skills', () => {
  const items = ref<Skill[]>([])
  const loading = ref(false)

  async function load() {
    loading.value = true
    try {
      items.value = asArray(await fetchJSON<Skill[]>('/skills'))
    } finally {
      loading.value = false
    }
  }

  async function get(skillId: string) {
    return fetchJSON<Skill>(`/skills/${skillId}`).catch(() => undefined)
  }

  async function create(payload: Omit<Skill, 'id'> & { id: string }) {
    const saved = await fetchJSON<Skill>('/skills', {
      method: 'POST',
      body: JSON.stringify(payload),
    })
    items.value.push(saved)
    return saved
  }

  async function update(skillId: string, payload: Skill) {
    const i = items.value.findIndex((s) => s.id === skillId)
    if (i < 0) throw new Error('Skill not found')
    const updated = await fetchJSON<Skill>(`/skills/${skillId}`, {
      method: 'PUT',
      body: JSON.stringify({ ...payload, id: skillId }),
    })
    items.value[i] = updated
    return updated
  }

  async function remove(skillId: string) {
    await fetchJSON(`/skills/${skillId}`, { method: 'DELETE' })
    items.value = items.value.filter((s) => s.id !== skillId)
  }

  async function importDir(dirPath: string) {
    const result = await fetchJSON<{ skill: Skill; fileCount: number }>('/skills/import', {
      method: 'POST',
      body: JSON.stringify({ path: dirPath }),
    })
    items.value.push(result.skill)
    return result
  }

  async function getFiles(skillId: string) {
    return fetchJSON<SkillFile[]>(`/skills/${skillId}/files`).catch(() => [] as SkillFile[])
  }

  async function getFileContent(skillId: string, filePath: string): Promise<string> {
    const resp = await fetch(`/api/v1/skills/${skillId}/files/${encodeURIComponent(filePath)}`)
    if (!resp.ok) throw new Error('File not found')
    return resp.text()
  }

  async function getExportMD(skillId: string) {
    const resp = await fetch(`/api/v1/skills/${skillId}/export`)
    if (!resp.ok) throw new Error('Export failed')
    return resp.text()
  }

  async function reset(skillId: string) {
    const updated = await fetchJSON<Skill>(`/skills/${skillId}/reset`, {
      method: 'POST',
    })
    const i = items.value.findIndex((s) => s.id === skillId)
    if (i >= 0) items.value[i] = updated
    return updated
  }

  return {
    items,
    loading,
    load,
    get,
    create,
    update,
    remove,
    importDir,
    getFiles,
    getFileContent,
    getExportMD,
    reset,
  }
})
