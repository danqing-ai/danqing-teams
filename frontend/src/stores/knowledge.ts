import { defineStore } from 'pinia'
import { ref } from 'vue'
import type { KnowledgeBase, KnowledgeDocument } from '@/types'

const KB_KEY = 'danqing-knowledge-bases'
const DOC_KEY = 'danqing-knowledge-documents'

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

export const useKnowledgeStore = defineStore('knowledge', () => {
  const bases = ref<KnowledgeBase[]>(loadJSON<KnowledgeBase>(KB_KEY))
  const documents = ref<KnowledgeDocument[]>(loadJSON<KnowledgeDocument>(DOC_KEY))

  function saveBases() {
    saveJSON(KB_KEY, bases.value)
  }

  function saveDocs() {
    saveJSON(DOC_KEY, documents.value)
  }

  function createBase(payload: Omit<KnowledgeBase, 'id' | 'updatedAt'>) {
    const now = new Date().toISOString()
    const base: KnowledgeBase = {
      ...payload,
      id: `kb-${Date.now()}`,
      updatedAt: now,
    }
    bases.value.push(base)
    saveBases()
    return base
  }

  function updateBase(id: string, payload: Partial<KnowledgeBase>) {
    const i = bases.value.findIndex((b) => b.id === id)
    if (i < 0) return undefined
    bases.value[i] = { ...bases.value[i], ...payload, updatedAt: new Date().toISOString() }
    saveBases()
    return bases.value[i]
  }

  function removeBase(id: string) {
    bases.value = bases.value.filter((b) => b.id !== id)
    documents.value = documents.value.filter((d) => d.knowledgeBaseId !== id)
    saveBases()
    saveDocs()
  }

  function addDocument(baseId: string, title: string, content: string) {
    const doc: KnowledgeDocument = {
      id: `doc-${Date.now()}`,
      knowledgeBaseId: baseId,
      title,
      content,
      updatedAt: new Date().toISOString(),
    }
    documents.value.push(doc)
    const base = bases.value.find((b) => b.id === baseId)
    if (base) {
      base.documentCount = documents.value.filter((d) => d.knowledgeBaseId === baseId).length
      base.updatedAt = doc.updatedAt
    }
    saveDocs()
    saveBases()
    return doc
  }

  function removeDocument(docId: string) {
    const doc = documents.value.find((d) => d.id === docId)
    documents.value = documents.value.filter((d) => d.id !== docId)
    if (doc) {
      const base = bases.value.find((b) => b.id === doc.knowledgeBaseId)
      if (base) {
        base.documentCount = documents.value.filter((d) => d.knowledgeBaseId === base.id).length
        base.updatedAt = new Date().toISOString()
      }
    }
    saveDocs()
    saveBases()
  }

  function documentsFor(baseId: string) {
    return documents.value.filter((d) => d.knowledgeBaseId === baseId)
  }

  return {
    bases,
    documents,
    createBase,
    updateBase,
    removeBase,
    addDocument,
    removeDocument,
    documentsFor,
  }
})
