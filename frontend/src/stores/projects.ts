import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { fetchJSON, asArray } from '@/api/client'
import { toast } from '@/utils/feedback'
import type { Project } from '@/types'

export const useProjectsStore = defineStore('projects', () => {
  const projects = ref<Project[]>([])
  const loading = ref(false)

  const sortedProjects = computed(() =>
    [...projects.value].sort((a, b) => a.name.localeCompare(b.name, 'zh-CN') || a.createdAt.localeCompare(b.createdAt)),
  )

  async function loadProjects() {
    loading.value = true
    try {
      projects.value = asArray(await fetchJSON<Project[]>('/projects'))
    } catch (e) {
      toast.error(e instanceof Error ? e.message : '加载项目失败')
      projects.value = []
    } finally {
      loading.value = false
    }
  }

  async function createProject(name: string, directory?: string): Promise<Project> {
    const p = await fetchJSON<Project>('/projects', {
      method: 'POST',
      body: JSON.stringify({ name, directory }),
    })
    projects.value = [p, ...projects.value.filter((x) => x.id !== p.id)]
    return p
  }

  async function renameProject(id: string, name: string) {
    const p = await fetchJSON<Project>(`/projects/${id}`, {
      method: 'PATCH',
      body: JSON.stringify({ name }),
    })
    const idx = projects.value.findIndex((x) => x.id === id)
    if (idx >= 0) projects.value[idx] = p
  }

  async function deleteProject(id: string) {
    await fetchJSON(`/projects/${id}`, { method: 'DELETE' })
    projects.value = projects.value.filter((p) => p.id !== id)
  }

  function ensureProjects() {
    if (!projects.value.length && !loading.value) {
      void loadProjects()
    }
  }

  return {
    projects,
    loading,
    sortedProjects,
    loadProjects,
    createProject,
    renameProject,
    deleteProject,
    ensureProjects,
  }
})
